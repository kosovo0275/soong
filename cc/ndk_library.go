package cc

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/google/blueprint"

	"android/soong/android"
)

var (
	toolPath = pctx.StaticVariable("toolPath", "build/soong/cc/gen_stub_libs.py")

	genStubSrc = pctx.AndroidStaticRule("genStubSrc", blueprint.RuleParams{
		Command:     "$toolPath --arch $arch --api $apiLevel --api-map $apiMap $vndk $in $out",
		CommandDeps: []string{"$toolPath"},
	}, "arch", "apiLevel", "apiMap", "vndk")

	ndkLibrarySuffix = ".ndk"

	ndkPrebuiltSharedLibs = []string{
		"android",
		"c",
		"dl",
		"EGL",
		"GLESv1_CM",
		"GLESv2",
		"GLESv3",
		"jnigraphics",
		"log",
		"mediandk",
		"m",
		"OpenMAXAL",
		"OpenSLES",
		"stdc++",
		"vulkan",
		"z",
	}
	ndkPrebuiltSharedLibraries = addPrefix(append([]string(nil), ndkPrebuiltSharedLibs...), "lib")

	ndkMigratedLibs     = []string{}
	ndkMigratedLibsLock sync.Mutex
)

type libraryProperties struct {
	Symbol_file *string

	First_version *string

	Unversioned_until *string

	ApiLevel string `blueprint:"mutated"`
}

type stubDecorator struct {
	*libraryDecorator

	properties libraryProperties

	versionScriptPath android.ModuleGenPath
	installPath       android.Path
}

func intMax(a int, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

func normalizeNdkApiLevel(ctx android.BaseContext, apiLevel string,
	arch android.Arch) (string, error) {

	if apiLevel == "current" {
		return apiLevel, nil
	}

	minVersion := ctx.Config().MinSupportedSdkVersion()
	firstArchVersions := map[android.ArchType]int{
		android.Arm:    minVersion,
		android.Arm64:  21,
		android.Mips:   minVersion,
		android.Mips64: 21,
		android.X86:    minVersion,
		android.X86_64: 21,
	}

	firstArchVersion, ok := firstArchVersions[arch.ArchType]
	if !ok {
		panic(fmt.Errorf("Arch %q not found in firstArchVersions", arch.ArchType))
	}

	if apiLevel == "minimum" {
		return strconv.Itoa(firstArchVersion), nil
	}

	version, err := strconv.Atoi(apiLevel)
	if err != nil {
		return "", fmt.Errorf("API level must be an integer (is %q)", apiLevel)
	}
	version = intMax(version, minVersion)

	return strconv.Itoa(intMax(version, firstArchVersion)), nil
}

func getFirstGeneratedVersion(firstSupportedVersion string, platformVersion int) (int, error) {
	if firstSupportedVersion == "current" {
		return platformVersion + 1, nil
	}

	return strconv.Atoi(firstSupportedVersion)
}

func shouldUseVersionScript(stub *stubDecorator) (bool, error) {

	if String(stub.properties.Unversioned_until) == "" {
		return true, nil
	}

	if String(stub.properties.Unversioned_until) == "current" {
		if stub.properties.ApiLevel == "current" {
			return true, nil
		} else {
			return false, nil
		}
	}

	if stub.properties.ApiLevel == "current" {
		return true, nil
	}

	unversionedUntil, err := strconv.Atoi(String(stub.properties.Unversioned_until))
	if err != nil {
		return true, err
	}

	version, err := strconv.Atoi(stub.properties.ApiLevel)
	if err != nil {
		return true, err
	}

	return version >= unversionedUntil, nil
}

func generateStubApiVariants(mctx android.BottomUpMutatorContext, c *stubDecorator) {
	platformVersion := mctx.Config().PlatformSdkVersionInt()

	firstSupportedVersion, err := normalizeNdkApiLevel(mctx, String(c.properties.First_version),
		mctx.Arch())
	if err != nil {
		mctx.PropertyErrorf("first_version", err.Error())
	}

	firstGenVersion, err := getFirstGeneratedVersion(firstSupportedVersion, platformVersion)
	if err != nil {

		mctx.PropertyErrorf("first_version", err.Error())
	}

	var versionStrs []string
	for version := firstGenVersion; version <= platformVersion; version++ {
		versionStrs = append(versionStrs, strconv.Itoa(version))
	}
	versionStrs = append(versionStrs, mctx.Config().PlatformVersionActiveCodenames()...)
	versionStrs = append(versionStrs, "current")

	modules := mctx.CreateVariations(versionStrs...)
	for i, module := range modules {
		module.(*Module).compiler.(*stubDecorator).properties.ApiLevel = versionStrs[i]
	}
}

func ndkApiMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok {
		if m.Enabled() {
			if compiler, ok := m.compiler.(*stubDecorator); ok {
				generateStubApiVariants(mctx, compiler)
			}
		}
	}
}

func (c *stubDecorator) compilerInit(ctx BaseModuleContext) {
	c.baseCompiler.compilerInit(ctx)

	name := ctx.baseModuleName()
	if strings.HasSuffix(name, ndkLibrarySuffix) {
		ctx.PropertyErrorf("name", "Do not append %q manually, just use the base name", ndkLibrarySuffix)
	}

	ndkMigratedLibsLock.Lock()
	defer ndkMigratedLibsLock.Unlock()
	for _, lib := range ndkMigratedLibs {
		if lib == name {
			return
		}
	}
	ndkMigratedLibs = append(ndkMigratedLibs, name)
}

func addStubLibraryCompilerFlags(flags Flags) Flags {
	flags.CFlags = append(flags.CFlags,

		"-Wno-incompatible-library-redeclaration",
		"-Wno-builtin-requires-header",
		"-Wno-invalid-noreturn",
		"-Wall",
		"-Werror",

		"-fno-unwind-tables",
	)
	return flags
}

func (stub *stubDecorator) compilerFlags(ctx ModuleContext, flags Flags, deps PathDeps) Flags {
	flags = stub.baseCompiler.compilerFlags(ctx, flags, deps)
	return addStubLibraryCompilerFlags(flags)
}

func compileStubLibrary(ctx ModuleContext, flags Flags, symbolFile, apiLevel, vndk string) (Objects, android.ModuleGenPath) {
	arch := ctx.Arch().ArchType.String()

	stubSrcPath := android.PathForModuleGen(ctx, "stub.c")
	versionScriptPath := android.PathForModuleGen(ctx, "stub.map")
	symbolFilePath := android.PathForModuleSrc(ctx, symbolFile)
	apiLevelsJson := android.GetApiLevelsJson(ctx)
	ctx.Build(pctx, android.BuildParams{
		Rule:        genStubSrc,
		Description: "generate stubs " + symbolFilePath.Rel(),
		Outputs:     []android.WritablePath{stubSrcPath, versionScriptPath},
		Input:       symbolFilePath,
		Implicits:   []android.Path{apiLevelsJson},
		Args: map[string]string{
			"arch":     arch,
			"apiLevel": apiLevel,
			"apiMap":   apiLevelsJson.String(),
			"vndk":     vndk,
		},
	})

	subdir := ""
	srcs := []android.Path{stubSrcPath}
	return compileObjs(ctx, flagsToBuilderFlags(flags), subdir, srcs, nil, nil), versionScriptPath
}

func (c *stubDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) Objects {
	if !strings.HasSuffix(String(c.properties.Symbol_file), ".map.txt") {
		ctx.PropertyErrorf("symbol_file", "must end with .map.txt")
	}

	objs, versionScript := compileStubLibrary(ctx, flags, String(c.properties.Symbol_file),
		c.properties.ApiLevel, "")
	c.versionScriptPath = versionScript
	return objs
}

func (linker *stubDecorator) linkerDeps(ctx DepsContext, deps Deps) Deps {
	return Deps{}
}

func (linker *stubDecorator) Name(name string) string {
	return name + ndkLibrarySuffix
}

func (stub *stubDecorator) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	stub.libraryDecorator.libName = ctx.baseModuleName()
	return stub.libraryDecorator.linkerFlags(ctx, flags)
}

func (stub *stubDecorator) link(ctx ModuleContext, flags Flags, deps PathDeps,
	objs Objects) android.Path {

	useVersionScript, err := shouldUseVersionScript(stub)
	if err != nil {
		ctx.ModuleErrorf(err.Error())
	}

	if useVersionScript {
		linkerScriptFlag := "-Wl,--version-script," + stub.versionScriptPath.String()
		flags.LdFlags = append(flags.LdFlags, linkerScriptFlag)
	}

	return stub.libraryDecorator.link(ctx, flags, deps, objs)
}

func (stub *stubDecorator) install(ctx ModuleContext, path android.Path) {
	arch := ctx.Target().Arch.ArchType.Name
	apiLevel := stub.properties.ApiLevel

	libDir := "lib"
	if ctx.toolchain().Is64Bit() && arch != "arm64" {
		libDir = "lib64"
	}

	installDir := getNdkInstallBase(ctx).Join(ctx, fmt.Sprintf(
		"platforms/android-%s/arch-%s/usr/%s", apiLevel, arch, libDir))
	stub.installPath = ctx.InstallFile(installDir, path.Base(), path)
}

func newStubLibrary() *Module {
	module, library := NewLibrary(android.DeviceSupported)
	library.BuildOnlyShared()
	module.stl = nil
	module.sanitize = nil
	library.StripProperties.Strip.None = BoolPtr(true)

	stub := &stubDecorator{
		libraryDecorator: library,
	}
	module.compiler = stub
	module.linker = stub
	module.installer = stub

	module.AddProperties(&stub.properties, &library.MutatedProperties)

	return module
}

func ndkLibraryFactory() android.Module {
	module := newStubLibrary()
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibBoth)
	return module
}
