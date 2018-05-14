package cc

import (
	"fmt"
	"path/filepath"
	"strings"

	"android/soong/android"
	"android/soong/cc/config"
)

var (
	profileUseOtherFlags = []string{"-Wno-backend-plugin"}

	globalPgoProfileProjects = []string{
		"toolchain/pgo-profiles",
		"vendor/google_data/pgo-profiles",
	}
)

const pgoProfileProjectsConfigKey = "PgoProfileProjects"
const profileInstrumentFlag = "-fprofile-generate=/data/local/tmp"
const profileSamplingFlag = "-gline-tables-only"
const profileUseInstrumentFormat = "-fprofile-use=%s"
const profileUseSamplingFormat = "-fprofile-sample-use=%s"

func getPgoProfileProjects(config android.DeviceConfig) []string {
	return config.OnceStringSlice(pgoProfileProjectsConfigKey, func() []string {
		return append(globalPgoProfileProjects, config.PgoAdditionalProfileDirs()...)
	})
}

func recordMissingProfileFile(ctx BaseModuleContext, missing string) {
	getNamedMapForConfig(ctx.Config(), modulesMissingProfileFile).Store(missing, true)
}

type PgoProperties struct {
	Pgo struct {
		Instrumentation    *bool
		Sampling           *bool
		Profile_file       *string `android:"arch_variant"`
		Benchmarks         []string
		Enable_profile_use *bool `android:"arch_variant"`

		Cflags []string `android:"arch_variant"`
	} `android:"arch_variant"`

	PgoPresent          bool `blueprint:"mutated"`
	ShouldProfileModule bool `blueprint:"mutated"`
	PgoCompile          bool `blueprint:"mutated"`
}

type pgo struct {
	Properties PgoProperties
}

func (props *PgoProperties) isInstrumentation() bool {
	return props.Pgo.Instrumentation != nil && *props.Pgo.Instrumentation == true
}

func (props *PgoProperties) isSampling() bool {
	return props.Pgo.Sampling != nil && *props.Pgo.Sampling == true
}

func (pgo *pgo) props() []interface{} {
	return []interface{}{&pgo.Properties}
}

func (props *PgoProperties) addProfileGatherFlags(ctx ModuleContext, flags Flags) Flags {
	flags.CFlags = append(flags.CFlags, props.Pgo.Cflags...)

	if props.isInstrumentation() {
		flags.CFlags = append(flags.CFlags, profileInstrumentFlag)

		flags.LdFlags = append(flags.LdFlags, "-u__llvm_profile_runtime")
	}
	if props.isSampling() {
		flags.CFlags = append(flags.CFlags, profileSamplingFlag)
		flags.LdFlags = append(flags.LdFlags, profileSamplingFlag)
	}
	return flags
}

func (props *PgoProperties) getPgoProfileFile(ctx BaseModuleContext) android.OptionalPath {
	profile_file := *props.Pgo.Profile_file

	for _, profileProject := range getPgoProfileProjects(ctx.DeviceConfig()) {

		globPattern := filepath.Join(profileProject, profile_file+".*")
		versioned_profiles, err := ctx.GlobWithDeps(globPattern, nil)
		if err != nil {
			ctx.ModuleErrorf("glob: %s", err.Error())
		}

		path := android.ExistentPathForSource(ctx, profileProject, profile_file)
		if path.Valid() {
			if len(versioned_profiles) != 0 {
				ctx.PropertyErrorf("pgo.profile_file", "Profile_file has multiple versions: "+filepath.Join(profileProject, profile_file)+", "+strings.Join(versioned_profiles, ", "))
			}
			return path
		}

		if len(versioned_profiles) > 1 {
			ctx.PropertyErrorf("pgo.profile_file", "Profile_file has multiple versions: "+strings.Join(versioned_profiles, ", "))
		} else if len(versioned_profiles) == 1 {
			return android.OptionalPathForPath(android.PathForSource(ctx, versioned_profiles[0]))
		}
	}

	missing := *props.Pgo.Profile_file + ":" + ctx.ModuleDir() + "/Android.bp:" + ctx.ModuleName()
	recordMissingProfileFile(ctx, missing)

	return android.OptionalPathForPath(nil)
}

func (props *PgoProperties) profileUseFlag(ctx ModuleContext, file string) string {
	if props.isInstrumentation() {
		return fmt.Sprintf(profileUseInstrumentFormat, file)
	}
	if props.isSampling() {
		return fmt.Sprintf(profileUseSamplingFormat, file)
	}
	return ""
}

func (props *PgoProperties) profileUseFlags(ctx ModuleContext, file string) []string {
	flags := []string{props.profileUseFlag(ctx, file)}
	flags = append(flags, profileUseOtherFlags...)
	return flags
}

func (props *PgoProperties) addProfileUseFlags(ctx ModuleContext, flags Flags) Flags {

	if !props.PgoPresent {
		return flags
	}

	if props.Pgo.Enable_profile_use != nil && *props.Pgo.Enable_profile_use == false {
		return flags
	}

	if profileFile := props.getPgoProfileFile(ctx); profileFile.Valid() {
		profileFilePath := profileFile.Path()
		profileUseFlags := props.profileUseFlags(ctx, profileFilePath.String())

		flags.CFlags = append(flags.CFlags, profileUseFlags...)
		flags.LdFlags = append(flags.LdFlags, profileUseFlags...)

		flags.CFlagsDeps = append(flags.CFlagsDeps, profileFilePath)
		flags.LdFlagsDeps = append(flags.LdFlagsDeps, profileFilePath)
	}
	return flags
}

func (props *PgoProperties) isPGO(ctx BaseModuleContext) bool {
	isInstrumentation := props.isInstrumentation()
	isSampling := props.isSampling()

	profileKindPresent := isInstrumentation || isSampling
	filePresent := props.Pgo.Profile_file != nil
	benchmarksPresent := len(props.Pgo.Benchmarks) > 0

	if !profileKindPresent && !filePresent && !benchmarksPresent {
		return false
	}

	if !profileKindPresent || !filePresent || !benchmarksPresent {
		var missing []string
		if !profileKindPresent {
			missing = append(missing, "profile kind (either \"instrumentation\" or \"sampling\" property)")
		}
		if !filePresent {
			missing = append(missing, "profile_file property")
		}
		if !benchmarksPresent {
			missing = append(missing, "non-empty benchmarks property")
		}
		missingProps := strings.Join(missing, ", ")
		ctx.ModuleErrorf("PGO specification is missing properties: " + missingProps)
	}

	if isSampling {
		ctx.PropertyErrorf("pgo.sampling", "\"sampling\" is not supported yet)")
	}

	if isSampling && isInstrumentation {
		ctx.PropertyErrorf("pgo", "Exactly one of \"instrumentation\" and \"sampling\" properties must be set")
	}

	return true
}

func (pgo *pgo) begin(ctx BaseModuleContext) {
	pgo.Properties.PgoPresent = pgo.Properties.isPGO(ctx)
	if !pgo.Properties.PgoPresent {
		return
	}

	pgo.Properties.ShouldProfileModule = false
	pgoBenchmarks := ctx.Config().Getenv("ANDROID_PGO_INSTRUMENT")
	pgoBenchmarksMap := make(map[string]bool)
	for _, b := range strings.Split(pgoBenchmarks, ",") {
		pgoBenchmarksMap[b] = true
	}

	if pgoBenchmarksMap["all"] == true || pgoBenchmarksMap["ALL"] == true {
		pgo.Properties.ShouldProfileModule = true
	} else {
		for _, b := range pgo.Properties.Pgo.Benchmarks {
			if pgoBenchmarksMap[b] == true {
				pgo.Properties.ShouldProfileModule = true
				break
			}
		}
	}

	if !ctx.Config().IsEnvTrue("ANDROID_PGO_NO_PROFILE_USE") {
		if profileFile := pgo.Properties.getPgoProfileFile(ctx); profileFile.Valid() {
			pgo.Properties.PgoCompile = true
		}
	}
}

func (pgo *pgo) deps(ctx BaseModuleContext, deps Deps) Deps {
	if pgo.Properties.ShouldProfileModule {
		runtimeLibrary := config.ProfileRuntimeLibrary(ctx.toolchain())
		deps.LateStaticLibs = append(deps.LateStaticLibs, runtimeLibrary)
	}
	return deps
}

func (pgo *pgo) flags(ctx ModuleContext, flags Flags) Flags {
	props := pgo.Properties

	if props.ShouldProfileModule {
		return props.addProfileGatherFlags(ctx, flags)
	}

	if !ctx.Config().IsEnvTrue("ANDROID_PGO_NO_PROFILE_USE") {
		return props.addProfileUseFlags(ctx, flags)
	}

	return flags
}
