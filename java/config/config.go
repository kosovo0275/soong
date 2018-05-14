package config

import (
	"path/filepath"
	"runtime"
	"strings"

	_ "github.com/google/blueprint/bootstrap"

	"android/soong/android"
)

var (
	pctx = android.NewPackageContext("android/soong/java/config")

	DefaultBootclasspathLibraries = []string{"core-oj", "core-libart"}
	DefaultSystemModules          = "core-system-modules"
	DefaultLibraries              = []string{"ext", "framework", "okhttp"}

	DefaultJacocoExcludeFilter = []string{"org.junit.*", "org.jacoco.*", "org.mockito.*"}

	InstrumentFrameworkModules = []string{
		"framework",
		"telephony-common",
		"services",
		"android.car",
		"android.car7",
	}
)

func init() {
	pctx.Import("github.com/google/blueprint/bootstrap")

	pctx.StaticVariable("JavacHeapSize", "2048M")
	pctx.StaticVariable("JavacHeapFlags", "-J-Xmx${JavacHeapSize}")

	pctx.StaticVariable("CommonJdkFlags", strings.Join([]string{
		`-Xmaxerrs 9999999`,
		`-encoding UTF-8`,
		`-sourcepath ""`,
		`-g`,

		`-XDskipDuplicateBridges=true`,

		`-XDstringConcat=inline`,
	}, " "))

	pctx.VariableConfigMethod("hostPrebuiltTag", android.Config.PrebuiltOS)

	pctx.VariableFunc("JavaHome", func(ctx android.PackageVarContext) string {

		return ctx.Config().Getenv("ANDROID_JAVA_HOME")
	})

	pctx.StaticVariable("JavaToolchain", "${JavaHome}/bin")
	pctx.SourcePathVariableWithEnvOverride("JavacCmd",
		"${JavaToolchain}/javac", "ALTERNATE_JAVAC")
	pctx.StaticVariable("JavaCmd", "${JavaToolchain}/java")
	pctx.StaticVariable("JarCmd", "${JavaToolchain}/jar")
	pctx.StaticVariable("JavadocCmd", "${JavaToolchain}/javadoc")
	pctx.StaticVariable("JlinkCmd", "${JavaToolchain}/jlink")
	pctx.StaticVariable("JmodCmd", "${JavaToolchain}/jmod")
	pctx.StaticVariable("JrtFsJar", "${JavaHome}/lib/jrt-fs.jar")
	pctx.StaticVariable("Ziptime", android.TermuxExecutable("ziptime"))

	pctx.StaticVariable("GenKotlinBuildFileCmd", "build/soong/scripts/gen-kotlin-build-file.sh")

	pctx.StaticVariable("JarArgsCmd", "build/soong/scripts/jar-args.sh")
	pctx.HostBinToolVariable("ExtractJarPackagesCmd", "extract_jar_packages")
	pctx.HostBinToolVariable("SoongZipCmd", "soong_zip")
	pctx.HostBinToolVariable("MergeZipsCmd", "merge_zips")
	pctx.HostBinToolVariable("Zip2ZipCmd", "zip2zip")
	pctx.HostBinToolVariable("ZipSyncCmd", "zipsync")
	pctx.VariableFunc("DxCmd", func(ctx android.PackageVarContext) string {
		return android.TermuxExecutable("dx")
	})
	pctx.HostBinToolVariable("D8Cmd", "d8")
	pctx.HostBinToolVariable("R8Cmd", "r8-compat-proguard")

	pctx.VariableFunc("TurbineJar", func(ctx android.PackageVarContext) string {
		return "prebuilts/build-tools/common/framework/turbine.jar"
	})

	pctx.HostJavaToolVariable("JarjarCmd", "jarjar.jar")
	pctx.HostJavaToolVariable("DesugarJar", "desugar.jar")
	pctx.HostJavaToolVariable("JsilverJar", "jsilver.jar")
	pctx.HostJavaToolVariable("DoclavaJar", "doclava.jar")

	pctx.HostBinToolVariable("SoongJavacWrapper", "soong_javac_wrapper")

	pctx.VariableFunc("JavacWrapper", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("JAVAC_WRAPPER"); override != "" {
			return override + " "
		}
		return ""
	})

	pctx.HostJavaToolVariable("JacocoCLIJar", "jacoco-cli.jar")

	hostBinToolVariableWithPrebuilt := func(name, prebuiltDir, tool string) {
		pctx.VariableFunc(name, func(ctx android.PackageVarContext) string {
			if ctx.Config().UnbundledBuild() || ctx.Config().IsPdkBuild() {
				return filepath.Join(prebuiltDir, runtime.GOOS, "bin", tool)
			} else {
				return pctx.HostBinToolPath(ctx, tool).String()
			}
		})
	}

	hostBinToolVariableWithPrebuilt("Aapt2Cmd", "prebuilts/sdk/tools", "aapt2")
}
