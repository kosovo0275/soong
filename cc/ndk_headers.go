package cc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/blueprint"

	"android/soong/android"
)

var (
	preprocessBionicHeaders = pctx.AndroidStaticRule("preprocessBionicHeaders",
		blueprint.RuleParams{

			Command:     "$versionerCmd -o $outDir $srcDir $depsPath && touch $out",
			CommandDeps: []string{"$versionerCmd"},
		},
		"depsPath", "srcDir", "outDir")
)

func init() {
	pctx.HostBinToolVariable("versionerCmd", "versioner")
}

func getCurrentIncludePath(ctx android.ModuleContext) android.OutputPath {
	return getNdkSysrootBase(ctx).Join(ctx, "usr/include")
}

type headerProperies struct {
	From *string

	To *string

	Srcs []string

	License *string
}

type headerModule struct {
	android.ModuleBase

	properties headerProperies

	installPaths android.Paths
	licensePath  android.ModuleSrcPath
}

func (m *headerModule) DepsMutator(ctx android.BottomUpMutatorContext) {
}

func getHeaderInstallDir(ctx android.ModuleContext, header android.Path, from string,
	to string) android.OutputPath {

	fullFromPath := android.PathForModuleSrc(ctx, from)

	headerDir := filepath.Dir(header.String())

	strippedHeaderDir, err := filepath.Rel(fullFromPath.String(), headerDir)
	if err != nil {
		ctx.ModuleErrorf("filepath.Rel(%q, %q) failed: %s", headerDir,
			fullFromPath.String(), err)
	}

	installDir := getCurrentIncludePath(ctx).Join(ctx, to, strippedHeaderDir)

	return installDir
}

func (m *headerModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	if String(m.properties.License) == "" {
		ctx.PropertyErrorf("license", "field is required")
	}

	m.licensePath = android.PathForModuleSrc(ctx, String(m.properties.License))

	if Bool(ctx.AConfig().Ndk_abis) && strings.Contains(ctx.ModuleName(), "mips") {
		return
	}

	srcFiles := ctx.ExpandSources(m.properties.Srcs, nil)
	for _, header := range srcFiles {
		installDir := getHeaderInstallDir(ctx, header, String(m.properties.From),
			String(m.properties.To))
		installedPath := ctx.InstallFile(installDir, header.Base(), header)
		installPath := installDir.Join(ctx, header.Base())
		if installPath != installedPath {
			panic(fmt.Sprintf(
				"expected header install path (%q) not equal to actual install path %q",
				installPath, installedPath))
		}
		m.installPaths = append(m.installPaths, installPath)
	}

	if len(m.installPaths) == 0 {
		ctx.ModuleErrorf("srcs %q matched zero files", m.properties.Srcs)
	}
}

func ndkHeadersFactory() android.Module {
	module := &headerModule{}
	module.AddProperties(&module.properties)
	android.InitAndroidModule(module)
	return module
}

type preprocessedHeaderProperies struct {
	From *string

	To *string

	License *string
}

type preprocessedHeaderModule struct {
	android.ModuleBase

	properties preprocessedHeaderProperies

	installPaths android.Paths
	licensePath  android.ModuleSrcPath
}

func (m *preprocessedHeaderModule) DepsMutator(ctx android.BottomUpMutatorContext) {
}

func (m *preprocessedHeaderModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	if String(m.properties.License) == "" {
		ctx.PropertyErrorf("license", "field is required")
	}

	m.licensePath = android.PathForModuleSrc(ctx, String(m.properties.License))

	fromSrcPath := android.PathForModuleSrc(ctx, String(m.properties.From))
	toOutputPath := getCurrentIncludePath(ctx).Join(ctx, String(m.properties.To))
	srcFiles := ctx.GlobFiles(filepath.Join(fromSrcPath.String(), "**/*.h"), nil)
	var installPaths []android.WritablePath
	for _, header := range srcFiles {
		installDir := getHeaderInstallDir(ctx, header, String(m.properties.From), String(m.properties.To))
		installPath := installDir.Join(ctx, header.Base())
		installPaths = append(installPaths, installPath)
		m.installPaths = append(m.installPaths, installPath)
	}

	if len(m.installPaths) == 0 {
		ctx.ModuleErrorf("glob %q matched zero files", String(m.properties.From))
	}

	processHeadersWithVersioner(ctx, fromSrcPath, toOutputPath, srcFiles, installPaths)
}

func processHeadersWithVersioner(ctx android.ModuleContext, srcDir, outDir android.Path, srcFiles android.Paths, installPaths []android.WritablePath) android.Path {

	depsPath := android.PathForSource(ctx, "bionic/libc/versioner-dependencies")
	depsGlob := ctx.Glob(filepath.Join(depsPath.String(), "**/*"), nil)
	for i, path := range depsGlob {
		fileInfo, err := os.Lstat(path.String())
		if err != nil {
			ctx.ModuleErrorf("os.Lstat(%q) failed: %s", path.String, err)
		}
		if fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink {
			dest, err := os.Readlink(path.String())
			if err != nil {
				ctx.ModuleErrorf("os.Readlink(%q) failed: %s",
					path.String, err)
			}

			depsGlob[i] = android.PathForSource(
				ctx, filepath.Clean(filepath.Join(path.String(), "..", dest)))
		}
	}

	timestampFile := android.PathForModuleOut(ctx, "versioner.timestamp")
	ctx.Build(pctx, android.BuildParams{
		Rule:            preprocessBionicHeaders,
		Description:     "versioner preprocess " + srcDir.Rel(),
		Output:          timestampFile,
		Implicits:       append(srcFiles, depsGlob...),
		ImplicitOutputs: installPaths,
		Args: map[string]string{
			"depsPath": depsPath.String(),
			"srcDir":   srcDir.String(),
			"outDir":   outDir.String(),
		},
	})

	return timestampFile
}

func preprocessedNdkHeadersFactory() android.Module {
	module := &preprocessedHeaderModule{}

	module.AddProperties(&module.properties)

	android.InitAndroidArchModule(module, android.HostSupportedNoCross, android.MultilibFirst)

	return module
}
