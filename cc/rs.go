package cc

import (
	"android/soong/android"
	"strings"

	"github.com/google/blueprint"
)

func init() {
	pctx.HostBinToolVariable("rsCmd", "llvm-rs-cc")
}

var rsCppCmdLine = strings.Replace(`
${rsCmd} -o ${outDir} -d ${outDir} -a ${out} -MD -reflect-c++ ${rsFlags} $in &&
(echo '${out}: \' && cat ${depFiles} | awk 'start { sub(/( \\)?$$/, " \\"); print } /:/ { start=1 }') > ${out}.d &&
touch $out
`, "\n", "", -1)

var (
	rsCpp = pctx.AndroidStaticRule("rsCpp",
		blueprint.RuleParams{
			Command:     rsCppCmdLine,
			CommandDeps: []string{"$rsCmd"},
			Depfile:     "${out}.d",
			Deps:        blueprint.DepsGCC,
		},
		"depFiles", "outDir", "rsFlags", "stampFile")
)

func rsGeneratedCppFile(ctx android.ModuleContext, rsFile android.Path) android.WritablePath {
	fileName := strings.TrimSuffix(rsFile.Base(), rsFile.Ext())
	return android.PathForModuleGen(ctx, "rs", "ScriptC_"+fileName+".cpp")
}

func rsGeneratedHFile(ctx android.ModuleContext, rsFile android.Path) android.WritablePath {
	fileName := strings.TrimSuffix(rsFile.Base(), rsFile.Ext())
	return android.PathForModuleGen(ctx, "rs", "ScriptC_"+fileName+".h")
}

func rsGeneratedDepFile(ctx android.ModuleContext, rsFile android.Path) android.WritablePath {
	fileName := strings.TrimSuffix(rsFile.Base(), rsFile.Ext())
	return android.PathForModuleGen(ctx, "rs", fileName+".d")
}

func rsGenerateCpp(ctx android.ModuleContext, rsFiles android.Paths, rsFlags string) android.Paths {
	stampFile := android.PathForModuleGen(ctx, "rs", "rs.stamp")
	depFiles := make(android.WritablePaths, 0, len(rsFiles))
	genFiles := make(android.WritablePaths, 0, 2*len(rsFiles))
	for _, rsFile := range rsFiles {
		depFiles = append(depFiles, rsGeneratedDepFile(ctx, rsFile))
		genFiles = append(genFiles,
			rsGeneratedCppFile(ctx, rsFile),
			rsGeneratedHFile(ctx, rsFile))
	}

	ctx.Build(pctx, android.BuildParams{
		Rule:            rsCpp,
		Description:     "llvm-rs-cc",
		Output:          stampFile,
		ImplicitOutputs: genFiles,
		Inputs:          rsFiles,
		Args: map[string]string{
			"rsFlags":  rsFlags,
			"outDir":   android.PathForModuleGen(ctx, "rs").String(),
			"depFiles": strings.Join(depFiles.Strings(), " "),
		},
	})

	return android.Paths{stampFile}
}

func rsFlags(ctx ModuleContext, flags Flags, properties *BaseCompilerProperties) Flags {
	targetApi := String(properties.Renderscript.Target_api)
	if targetApi == "" && ctx.useSdk() {
		switch ctx.sdkVersion() {
		case "current", "system_current", "test_current":
			// Nothing
		default:
			targetApi = android.GetNumericSdkVersion(ctx.sdkVersion())
		}
	}

	if targetApi != "" {
		flags.rsFlags = append(flags.rsFlags, "-target-api "+targetApi)
	}

	flags.rsFlags = append(flags.rsFlags, "-Wall", "-Werror")
	flags.rsFlags = append(flags.rsFlags, properties.Renderscript.Flags...)
	if ctx.Arch().ArchType.Multilib == "lib64" {
		flags.rsFlags = append(flags.rsFlags, "-m64")
	} else {
		flags.rsFlags = append(flags.rsFlags, "-m32")
	}
	flags.rsFlags = append(flags.rsFlags, "${config.RsGlobalIncludes}")

	rootRsIncludeDirs := android.PathsForSource(ctx, properties.Renderscript.Include_dirs)
	flags.rsFlags = append(flags.rsFlags, includeDirsToFlags(rootRsIncludeDirs))

	flags.GlobalFlags = append(flags.GlobalFlags,
		"-I"+android.PathForModuleGen(ctx, "rs").String(),
		"-Iframeworks/rs",
		"-Iframeworks/rs/cpp",
	)

	return flags
}
