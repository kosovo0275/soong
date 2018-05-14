package cc

import (
	"fmt"

	"android/soong/android"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func init() {
	android.RegisterSingletonType("cmakelists_generator", cMakeListsGeneratorSingleton)
}

func cMakeListsGeneratorSingleton() android.Singleton {
	return &cmakelistsGeneratorSingleton{}
}

type cmakelistsGeneratorSingleton struct{}

const (
	cMakeListsFilename              = "CMakeLists.txt"
	cLionAggregateProjectsDirectory = "development" + string(os.PathSeparator) + "ide" + string(os.PathSeparator) + "clion"
	cLionOutputProjectsDirectory    = "out" + string(os.PathSeparator) + cLionAggregateProjectsDirectory
	minimumCMakeVersionSupported    = "3.5"

	envVariableGenerateCMakeLists = "SOONG_GEN_CMAKEFILES"
	envVariableGenerateDebugInfo  = "SOONG_GEN_CMAKEFILES_DEBUG"
	envVariableTrue               = "1"
)

var outputDebugInfo = true

func (c *cmakelistsGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if getEnvVariable(envVariableGenerateCMakeLists, ctx) != envVariableTrue {
		return
	}

	outputDebugInfo = (getEnvVariable(envVariableGenerateDebugInfo, ctx) == envVariableTrue)

	ctx.VisitAllModules(func(module android.Module) {
		if ccModule, ok := module.(*Module); ok {
			if compiledModule, ok := ccModule.compiler.(CompiledInterface); ok {
				generateCLionProject(compiledModule, ctx, ccModule)
			}
		}
	})

	dir := filepath.Join(getAndroidSrcRootDirectory(ctx), cLionAggregateProjectsDirectory)
	filepath.Walk(dir, linkAggregateCMakeListsFiles)

	return
}

func getEnvVariable(name string, ctx android.SingletonContext) string {

	return ctx.Config().Getenv(name)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func linkAggregateCMakeListsFiles(path string, info os.FileInfo, err error) error {

	if info == nil {
		return nil
	}

	dst := strings.Replace(path, cLionAggregateProjectsDirectory, cLionOutputProjectsDirectory, 1)
	if info.IsDir() {

		os.MkdirAll(dst, os.ModePerm)
	} else {

		os.Remove(dst)
		os.Symlink(path, dst)
	}
	return nil
}

func generateCLionProject(compiledModule CompiledInterface, ctx android.SingletonContext, ccModule *Module) {
	srcs := compiledModule.Srcs()
	if len(srcs) == 0 {
		return
	}

	clionproject_location := getCMakeListsForModule(ccModule, ctx)
	projectDir := path.Dir(clionproject_location)
	os.MkdirAll(projectDir, os.ModePerm)

	f, _ := os.Create(filepath.Join(projectDir, cMakeListsFilename))
	defer f.Close()

	f.WriteString("# THIS FILE WAS AUTOMATICALY GENERATED!\n")
	f.WriteString("# ANY MODIFICATION WILL BE OVERWRITTEN!\n\n")
	f.WriteString("# To improve project view in Clion    :\n")
	f.WriteString("# Tools > CMake > Change Project Root  \n\n")
	f.WriteString(fmt.Sprintf("cmake_minimum_required(VERSION %s)\n", minimumCMakeVersionSupported))
	f.WriteString(fmt.Sprintf("project(%s)\n", ccModule.ModuleBase.Name()))
	f.WriteString(fmt.Sprintf("set(ANDROID_ROOT %s)\n\n", getAndroidSrcRootDirectory(ctx)))

	if ccModule.flags.Clang {
		f.WriteString(fmt.Sprintf("set(CMAKE_C_COMPILER \"%s%s\")\n", android.TermuxExecutable("clang-7")))
		f.WriteString(fmt.Sprintf("set(CMAKE_CXX_COMPILER \"%s%s\")\n", android.TermuxExecutable("clang++")))
	} else {
		f.WriteString(fmt.Sprintf("set(CMAKE_C_COMPILER \"%s%s\")\n", android.TermuxExecutable("gcc-7")))
		f.WriteString(fmt.Sprintf("set(CMAKE_CXX_COMPILER \"%s%s\")\n", android.TermuxExecutable("g++-7")))
	}

	f.WriteString("list(APPEND\n")
	f.WriteString("     SOURCE_FILES\n")
	for _, src := range srcs {
		f.WriteString(fmt.Sprintf("    ${ANDROID_ROOT}/%s\n", src.String()))
	}
	f.WriteString(")\n")

	f.WriteString("\n# GLOBAL FLAGS:\n")
	globalParameters := parseCompilerParameters(ccModule.flags.GlobalFlags, ctx, f)
	translateToCMake(globalParameters, f, true, true)

	f.WriteString("\n# CFLAGS:\n")
	cParameters := parseCompilerParameters(ccModule.flags.CFlags, ctx, f)
	translateToCMake(cParameters, f, true, true)

	f.WriteString("\n# C ONLY FLAGS:\n")
	cOnlyParameters := parseCompilerParameters(ccModule.flags.ConlyFlags, ctx, f)
	translateToCMake(cOnlyParameters, f, true, false)

	f.WriteString("\n# CPP FLAGS:\n")
	cppParameters := parseCompilerParameters(ccModule.flags.CppFlags, ctx, f)
	translateToCMake(cppParameters, f, false, true)

	f.WriteString("\n# SYSTEM INCLUDE FLAGS:\n")
	includeParameters := parseCompilerParameters(ccModule.flags.SystemIncludeFlags, ctx, f)
	translateToCMake(includeParameters, f, true, true)

	f.WriteString(fmt.Sprintf("\nadd_executable(%s ${SOURCE_FILES})\n", cleanExecutableName(ccModule.ModuleBase.Name())))
}

func cleanExecutableName(s string) string {
	return strings.Replace(s, "@", "-", -1)
}

func translateToCMake(c compilerParameters, f *os.File, cflags bool, cppflags bool) {
	writeAllIncludeDirectories(c.systemHeaderSearchPath, f, true)
	writeAllIncludeDirectories(c.headerSearchPath, f, false)
	if cflags {
		writeAllFlags(c.flags, f, "CMAKE_C_FLAGS")
	}

	if cppflags {
		writeAllFlags(c.flags, f, "CMAKE_CXX_FLAGS")
	}
	if c.sysroot != "" {
		f.WriteString(fmt.Sprintf("include_directories(SYSTEM \"%s\")\n", buildCMakePath(path.Join(c.sysroot, "usr", "include"))))
	}

}

func buildCMakePath(p string) string {
	if path.IsAbs(p) {
		return p
	}
	return fmt.Sprintf("${ANDROID_ROOT}/%s", p)
}

func writeAllIncludeDirectories(includes []string, f *os.File, isSystem bool) {
	if len(includes) == 0 {
		return
	}

	system := ""
	if isSystem {
		system = "SYSTEM"
	}

	f.WriteString(fmt.Sprintf("include_directories(%s \n", system))

	for _, include := range includes {
		f.WriteString(fmt.Sprintf("    \"%s\"\n", buildCMakePath(include)))
	}
	f.WriteString(")\n\n")

	f.WriteString("file (GLOB_RECURSE TMP_HEADERS\n")
	for _, include := range includes {
		f.WriteString(fmt.Sprintf("    \"%s/**/*.h\"\n", buildCMakePath(include)))
	}
	f.WriteString(")\n")
	f.WriteString("list (APPEND SOURCE_FILES ${TMP_HEADERS})\n\n")
}

func writeAllFlags(flags []string, f *os.File, tag string) {
	for _, flag := range flags {
		f.WriteString(fmt.Sprintf("set(%s \"${%s} %s\")\n", tag, tag, flag))
	}
}

type parameterType int

const (
	headerSearchPath parameterType = iota
	variable
	systemHeaderSearchPath
	flag
	systemRoot
)

type compilerParameters struct {
	headerSearchPath       []string
	systemHeaderSearchPath []string
	flags                  []string
	sysroot                string
}

func makeCompilerParameters() compilerParameters {
	return compilerParameters{
		sysroot: "",
	}
}

func categorizeParameter(parameter string) parameterType {
	if strings.HasPrefix(parameter, "-I") {
		return headerSearchPath
	}
	if strings.HasPrefix(parameter, "$") {
		return variable
	}
	if strings.HasPrefix(parameter, "-isystem") {
		return systemHeaderSearchPath
	}
	if strings.HasPrefix(parameter, "-isysroot") {
		return systemRoot
	}
	if strings.HasPrefix(parameter, "--sysroot") {
		return systemRoot
	}
	return flag
}

func parseCompilerParameters(params []string, ctx android.SingletonContext, f *os.File) compilerParameters {
	var compilerParameters = makeCompilerParameters()

	for i, str := range params {
		f.WriteString(fmt.Sprintf("# Raw param [%d] = '%s'\n", i, str))
	}

	for i := 0; i < len(params); i++ {
		param := params[i]
		if param == "" {
			continue
		}

		switch categorizeParameter(param) {
		case headerSearchPath:
			compilerParameters.headerSearchPath =
				append(compilerParameters.headerSearchPath, strings.TrimPrefix(param, "-I"))
		case variable:
			if evaluated, error := evalVariable(ctx, param); error == nil {
				if outputDebugInfo {
					f.WriteString(fmt.Sprintf("# variable %s = '%s'\n", param, evaluated))
				}

				paramsFromVar := parseCompilerParameters(strings.Split(evaluated, " "), ctx, f)
				concatenateParams(&compilerParameters, paramsFromVar)

			} else {
				if outputDebugInfo {
					f.WriteString(fmt.Sprintf("# variable %s could NOT BE RESOLVED\n", param))
				}
			}
		case systemHeaderSearchPath:
			if i < len(params)-1 {
				compilerParameters.systemHeaderSearchPath =
					append(compilerParameters.systemHeaderSearchPath, params[i+1])
			} else if outputDebugInfo {
				f.WriteString("# Found a header search path marker with no path")
			}
			i = i + 1
		case flag:
			c := cleanupParameter(param)
			f.WriteString(fmt.Sprintf("# FLAG '%s' became %s\n", param, c))
			compilerParameters.flags = append(compilerParameters.flags, c)
		case systemRoot:
			if i < len(params)-1 {
				compilerParameters.sysroot = params[i+1]
			} else if outputDebugInfo {
				f.WriteString("# Found a system root path marker with no path")
			}
			i = i + 1
		}
	}
	return compilerParameters
}

func cleanupParameter(p string) string {

	if !strings.HasPrefix(p, "'") || !strings.HasSuffix(p, "'") || len(p) < 3 {
		return p
	}

	p = p[1 : len(p)-1]
	p = strings.Replace(p, `'\''`, `'`, -1)
	p = strings.Replace(p, `$$`, `$`, -1)

	p = doubleEscape(p)
	return p
}

func escape(s string) string {
	s = strings.Replace(s, `\`, `\\`, -1)
	s = strings.Replace(s, `"`, `\"`, -1)
	return s
}

func doubleEscape(s string) string {
	s = escape(s)
	s = escape(s)
	return s
}

func concatenateParams(c1 *compilerParameters, c2 compilerParameters) {
	c1.headerSearchPath = append(c1.headerSearchPath, c2.headerSearchPath...)
	c1.systemHeaderSearchPath = append(c1.systemHeaderSearchPath, c2.systemHeaderSearchPath...)
	if c2.sysroot != "" {
		c1.sysroot = c2.sysroot
	}
	c1.flags = append(c1.flags, c2.flags...)
}

func evalVariable(ctx android.SingletonContext, str string) (string, error) {
	evaluated, err := ctx.Eval(pctx, str)
	if err == nil {
		return evaluated, nil
	}
	return "", err
}

func getCMakeListsForModule(module *Module, ctx android.SingletonContext) string {
	return filepath.Join(getAndroidSrcRootDirectory(ctx),
		cLionOutputProjectsDirectory,
		path.Dir(ctx.BlueprintFile(module)),
		module.ModuleBase.Name()+"-"+
			module.ModuleBase.Arch().ArchType.Name+"-"+
			module.ModuleBase.Os().Name,
		cMakeListsFilename)
}

func getAndroidSrcRootDirectory(ctx android.SingletonContext) string {
	srcPath, _ := filepath.Abs(android.PathForSource(ctx).String())
	return srcPath
}
