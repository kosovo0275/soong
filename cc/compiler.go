package cc

import (
	"android/soong/android"
	"android/soong/cc/config"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/blueprint/proptools"
)

type BaseCompilerProperties struct {
	Srcs               []string `android:"arch_variant"`
	Exclude_srcs       []string `android:"arch_variant"`
	Cflags             []string `android:"arch_variant"`
	Cppflags           []string `android:"arch_variant"`
	Conlyflags         []string `android:"arch_variant"`
	Asflags            []string `android:"arch_variant"`
	Clang_cflags       []string `android:"arch_variant"`
	Clang_asflags      []string `android:"arch_variant"`
	Yaccflags          []string
	Instruction_set    *string  `android:"arch_variant"`
	Include_dirs       []string `android:"arch_variant,variant_prepend"`
	Local_include_dirs []string `android:"arch_variant,variant_prepend",`
	Generated_sources  []string `android:"arch_variant"`
	Generated_headers  []string `android:"arch_variant"`
	Rtti               *bool
	C_std              *string
	Cpp_std            *string
	Gnu_extensions     *bool

	Aidl struct {
		Include_dirs       []string
		Local_include_dirs []string
		Generate_traces    *bool
	}

	Renderscript struct {
		Include_dirs []string
		Flags        []string
		Target_api   *string
	}

	Debug, Release struct {
		Cflags []string `android:"arch_variant"`
	} `android:"arch_variant"`

	Target struct {
		Vendor struct {
			Srcs         []string
			Exclude_srcs []string
			Cflags       []string
		}
	}

	Proto struct {
		Static *bool `android:"arch_variant"`
	} `android:"arch_variant"`

	OriginalSrcs []string `blueprint:"mutated"`
	Openmp       *bool    `android:"arch_variant"`
}

func NewBaseCompiler() *baseCompiler {
	return &baseCompiler{}
}

type baseCompiler struct {
	Properties    BaseCompilerProperties
	Proto         android.ProtoProperties
	cFlagsDeps    android.Paths
	pathDeps      android.Paths
	flags         builderFlags
	srcs          android.Paths
	srcsBeforeGen android.Paths
}

var _ compiler = (*baseCompiler)(nil)

type CompiledInterface interface {
	Srcs() android.Paths
}

func (compiler *baseCompiler) Srcs() android.Paths {
	return append(android.Paths{}, compiler.srcs...)
}

func (compiler *baseCompiler) appendCflags(flags []string) {
	compiler.Properties.Cflags = append(compiler.Properties.Cflags, flags...)
}

func (compiler *baseCompiler) appendAsflags(flags []string) {
	compiler.Properties.Asflags = append(compiler.Properties.Asflags, flags...)
}

func (compiler *baseCompiler) compilerProps() []interface{} {
	return []interface{}{&compiler.Properties, &compiler.Proto}
}

func (compiler *baseCompiler) compilerInit(ctx BaseModuleContext) {}

func (compiler *baseCompiler) compilerDeps(ctx DepsContext, deps Deps) Deps {
	deps.GeneratedSources = append(deps.GeneratedSources, compiler.Properties.Generated_sources...)
	deps.GeneratedHeaders = append(deps.GeneratedHeaders, compiler.Properties.Generated_headers...)
	android.ExtractSourcesDeps(ctx, compiler.Properties.Srcs)
	android.ExtractSourcesDeps(ctx, compiler.Properties.Exclude_srcs)

	if compiler.hasSrcExt(".proto") {
		deps = protoDeps(ctx, deps, &compiler.Proto, Bool(compiler.Properties.Proto.Static))
	}

	if Bool(compiler.Properties.Openmp) {
		deps.StaticLibs = append(deps.StaticLibs, "libomp")
	}

	return deps
}

func warningsAreAllowed(subdir string) bool {
	subdir += "/"
	for _, prefix := range config.WarningAllowedProjects {
		if strings.HasPrefix(subdir, prefix) {
			return true
		}
	}
	return false
}

func addToModuleList(ctx ModuleContext, list string, module string) {
	getNamedMapForConfig(ctx.Config(), list).Store(module, true)
}

func (compiler *baseCompiler) compilerFlags(ctx ModuleContext, flags Flags, deps PathDeps) Flags {
	tc := ctx.toolchain()

	compiler.srcsBeforeGen = ctx.ExpandSources(compiler.Properties.Srcs, compiler.Properties.Exclude_srcs)
	compiler.srcsBeforeGen = append(compiler.srcsBeforeGen, deps.GeneratedSources...)

	CheckBadCompilerFlags(ctx, "cflags", compiler.Properties.Cflags)
	CheckBadCompilerFlags(ctx, "cppflags", compiler.Properties.Cppflags)
	CheckBadCompilerFlags(ctx, "conlyflags", compiler.Properties.Conlyflags)
	CheckBadCompilerFlags(ctx, "asflags", compiler.Properties.Asflags)

	esc := proptools.NinjaAndShellEscape

	flags.CFlags = append(flags.CFlags, esc(compiler.Properties.Cflags)...)
	flags.CppFlags = append(flags.CppFlags, esc(compiler.Properties.Cppflags)...)
	flags.ConlyFlags = append(flags.ConlyFlags, esc(compiler.Properties.Conlyflags)...)
	flags.AsFlags = append(flags.AsFlags, esc(compiler.Properties.Asflags)...)
	flags.YasmFlags = append(flags.YasmFlags, esc(compiler.Properties.Asflags)...)
	flags.YaccFlags = append(flags.YaccFlags, esc(compiler.Properties.Yaccflags)...)

	localIncludeDirs := android.PathsForModuleSrc(ctx, compiler.Properties.Local_include_dirs)
	if len(localIncludeDirs) > 0 {
		f := includeDirsToFlags(localIncludeDirs)
		flags.GlobalFlags = append(flags.GlobalFlags, f)
		flags.YasmFlags = append(flags.YasmFlags, f)
	}
	rootIncludeDirs := android.PathsForSource(ctx, compiler.Properties.Include_dirs)
	if len(rootIncludeDirs) > 0 {
		f := includeDirsToFlags(rootIncludeDirs)
		flags.GlobalFlags = append(flags.GlobalFlags, f)
		flags.YasmFlags = append(flags.YasmFlags, f)
	}

	flags.GlobalFlags = append(flags.GlobalFlags, "-I"+android.PathForModuleSrc(ctx).String())
	flags.YasmFlags = append(flags.YasmFlags, "-I"+android.PathForModuleSrc(ctx).String())

	if !(ctx.useSdk() || ctx.useVndk()) || ctx.Host() {
		flags.SystemIncludeFlags = append(flags.SystemIncludeFlags, "${config.CommonGlobalIncludes}", tc.IncludeFlags(), "${config.CommonNativehelperInclude}")
	}

	if ctx.useSdk() {
		version := ctx.sdkVersion()
		if version == "current" {
			version = "__ANDROID_API_FUTURE__"
		}
		flags.GlobalFlags = append(flags.GlobalFlags, "-D__ANDROID_API__="+version)
		legacyIncludes := fmt.Sprintf("prebuilts/ndk/current/platforms/android-%s/arch-%s/usr/include", ctx.sdkVersion(), ctx.Arch().ArchType.String())
		flags.SystemIncludeFlags = append(flags.SystemIncludeFlags, "-isystem "+legacyIncludes)
	} else {
		flags.GlobalFlags = append(flags.GlobalFlags, "-D__ANDROID_API__=__ANDROID_API_FUTURE__")
	}

	if ctx.useVndk() {
		version := ctx.sdkVersion()
		if version == "current" {
			version = "__ANDROID_API_FUTURE__"
		}
		flags.GlobalFlags = append(flags.GlobalFlags, "-D__ANDROID_API__="+version, "-D__ANDROID_VNDK__")
	}

	instructionSet := String(compiler.Properties.Instruction_set)
	if flags.RequiredInstructionSet != "" {
		instructionSet = flags.RequiredInstructionSet
	}
	instructionSetFlags, err := tc.InstructionSetFlags(instructionSet)
	if flags.Clang {
		instructionSetFlags, err = tc.ClangInstructionSetFlags(instructionSet)
	}
	if err != nil {
		ctx.ModuleErrorf("%s", err)
	}

	CheckBadCompilerFlags(ctx, "release.cflags", compiler.Properties.Release.Cflags)
	flags.CFlags = append(flags.CFlags, esc(compiler.Properties.Release.Cflags)...)

	if flags.Clang {
		CheckBadCompilerFlags(ctx, "clang_cflags", compiler.Properties.Clang_cflags)
		CheckBadCompilerFlags(ctx, "clang_asflags", compiler.Properties.Clang_asflags)

		flags.CFlags = config.ClangFilterUnknownCflags(flags.CFlags)
		flags.CFlags = append(flags.CFlags, esc(compiler.Properties.Clang_cflags)...)
		flags.AsFlags = append(flags.AsFlags, esc(compiler.Properties.Clang_asflags)...)
		flags.CppFlags = config.ClangFilterUnknownCflags(flags.CppFlags)
		flags.ConlyFlags = config.ClangFilterUnknownCflags(flags.ConlyFlags)
		flags.LdFlags = config.ClangFilterUnknownCflags(flags.LdFlags)

		target := "-target " + tc.ClangTriple()
		gccPrefix := "-B" + config.ToolPath(tc)

		flags.CFlags = append(flags.CFlags, target, gccPrefix)
		flags.AsFlags = append(flags.AsFlags, target, gccPrefix)
		flags.LdFlags = append(flags.LdFlags, target, gccPrefix)
	}

	hod := "Host"
	if ctx.Os().Class == android.Device {
		hod = "Device"
	}

	flags.GlobalFlags = append(flags.GlobalFlags, instructionSetFlags)
	flags.ConlyFlags = append([]string{"${config.CommonGlobalConlyflags}"}, flags.ConlyFlags...)
	flags.CppFlags = append([]string{fmt.Sprintf("${config.%sGlobalCppflags}", hod)}, flags.CppFlags...)

	if flags.Clang {
		flags.AsFlags = append(flags.AsFlags, tc.ClangAsflags())
		flags.CppFlags = append([]string{"${config.CommonClangGlobalCppflags}"}, flags.CppFlags...)
		flags.GlobalFlags = append(flags.GlobalFlags, tc.ClangCflags(), "${config.CommonClangGlobalCflags}", fmt.Sprintf("${config.%sClangGlobalCflags}", hod))
	} else {
		flags.CppFlags = append([]string{"${config.CommonGlobalCppflags}"}, flags.CppFlags...)
		flags.GlobalFlags = append(flags.GlobalFlags, tc.Cflags(), "${config.CommonGlobalCflags}", fmt.Sprintf("${config.%sGlobalCflags}", hod))
	}

	if ctx.Device() {
		if Bool(compiler.Properties.Rtti) {
			flags.CppFlags = append(flags.CppFlags, "-frtti")
		}
	}

	flags.AsFlags = append(flags.AsFlags, "-D__ASSEMBLY__")

	if flags.Clang {
		flags.CppFlags = append(flags.CppFlags, tc.ClangCppflags())
		flags.GlobalFlags = append(flags.GlobalFlags, tc.ToolchainClangCflags())
	} else {
		flags.CppFlags = append(flags.CppFlags, tc.Cppflags())
		flags.GlobalFlags = append(flags.GlobalFlags, tc.ToolchainCflags())
	}

	flags.YasmFlags = append(flags.YasmFlags, tc.YasmFlags())

	cStd := config.CStdVersion
	if String(compiler.Properties.C_std) == "experimental" {
		cStd = config.ExperimentalCStdVersion
	} else if String(compiler.Properties.C_std) != "" {
		cStd = String(compiler.Properties.C_std)
	}

	cppStd := String(compiler.Properties.Cpp_std)
	switch String(compiler.Properties.Cpp_std) {
	case "":
		cppStd = config.CppStdVersion
	case "experimental":
		cppStd = config.ExperimentalCppStdVersion
	case "c++17", "gnu++17":

		cppStd = strings.Replace(String(compiler.Properties.Cpp_std), "17", "1z", 1)
	}

	if !flags.Clang {
		cppStd = config.GccCppStdVersion
	}

	if compiler.Properties.Gnu_extensions != nil && *compiler.Properties.Gnu_extensions == false {
		cStd = gnuToCReplacer.Replace(cStd)
		cppStd = gnuToCReplacer.Replace(cppStd)
	}

	flags.ConlyFlags = append([]string{"-std=" + cStd}, flags.ConlyFlags...)
	flags.CppFlags = append([]string{"-std=" + cppStd}, flags.CppFlags...)

	if ctx.useVndk() {
		flags.CFlags = append(flags.CFlags, esc(compiler.Properties.Target.Vendor.Cflags)...)
	}

	if compiler.hasSrcExt(".proto") {
		flags = protoFlags(ctx, flags, &compiler.Proto)
	}

	if compiler.hasSrcExt(".y") || compiler.hasSrcExt(".yy") {
		flags.GlobalFlags = append(flags.GlobalFlags, "-I"+android.PathForModuleGen(ctx, "yacc", ctx.ModuleDir()).String())
	}

	if compiler.hasSrcExt(".mc") {
		flags.GlobalFlags = append(flags.GlobalFlags, "-I"+android.PathForModuleGen(ctx, "windmc", ctx.ModuleDir()).String())
	}

	if compiler.hasSrcExt(".aidl") {
		if len(compiler.Properties.Aidl.Local_include_dirs) > 0 {
			localAidlIncludeDirs := android.PathsForModuleSrc(ctx, compiler.Properties.Aidl.Local_include_dirs)
			flags.aidlFlags = append(flags.aidlFlags, includeDirsToFlags(localAidlIncludeDirs))
		}
		if len(compiler.Properties.Aidl.Include_dirs) > 0 {
			rootAidlIncludeDirs := android.PathsForSource(ctx, compiler.Properties.Aidl.Include_dirs)
			flags.aidlFlags = append(flags.aidlFlags, includeDirsToFlags(rootAidlIncludeDirs))
		}

		if Bool(compiler.Properties.Aidl.Generate_traces) {
			flags.aidlFlags = append(flags.aidlFlags, "-t")
		}

		flags.GlobalFlags = append(flags.GlobalFlags, "-I"+android.PathForModuleGen(ctx, "aidl").String())
	}

	if compiler.hasSrcExt(".rs") || compiler.hasSrcExt(".fs") {
		flags = rsFlags(ctx, flags, &compiler.Properties)
	}

	if len(compiler.Properties.Srcs) > 0 {
		module := ctx.ModuleDir() + "/Android.bp:" + ctx.ModuleName()
		if inList("-Wno-error", flags.CFlags) || inList("-Wno-error", flags.CppFlags) {
			addToModuleList(ctx, modulesUsingWnoError, module)
		} else if !inList("-Werror", flags.CFlags) && !inList("-Werror", flags.CppFlags) {
			if warningsAreAllowed(ctx.ModuleDir()) {
				addToModuleList(ctx, modulesAddedWall, module)
				flags.CFlags = append([]string{"-Wall"}, flags.CFlags...)
			} else {
				flags.CFlags = append([]string{"-Wall", "-Werror"}, flags.CFlags...)
			}
		}
	}

	if Bool(compiler.Properties.Openmp) {
		flags.CFlags = append(flags.CFlags, "-fopenmp")
	}

	flags.CFlags = append([]string{"-I${config.ClangBase}/include"}, flags.CFlags...)
	flags.CFlags = append([]string{"-I${config.ClangBase}/include/c++/v1"}, flags.CFlags...)
	if flags.Clang {
		flags.CFlags = append([]string{"-I${config.RSIncludePath}"}, flags.CFlags...)
	} else {
		flags.CFlags = append([]string{"-I${config.ClangBase}/lib/gcc/aarch64-linux-android/7.3.0/include"}, flags.CFlags...)
	}

	return flags
}

func (compiler *baseCompiler) hasSrcExt(ext string) bool {
	for _, src := range compiler.srcsBeforeGen {
		if src.Ext() == ext {
			return true
		}
	}
	for _, src := range compiler.Properties.Srcs {
		if filepath.Ext(src) == ext {
			return true
		}
	}
	for _, src := range compiler.Properties.OriginalSrcs {
		if filepath.Ext(src) == ext {
			return true
		}
	}

	return false
}

var gnuToCReplacer = strings.NewReplacer("gnu", "c")

func ndkPathDeps(ctx ModuleContext) android.Paths {
	return nil
}

func (compiler *baseCompiler) compile(ctx ModuleContext, flags Flags, deps PathDeps) Objects {
	pathDeps := deps.GeneratedHeaders
	pathDeps = append(pathDeps, ndkPathDeps(ctx)...)
	buildFlags := flagsToBuilderFlags(flags)
	srcs := append(android.Paths(nil), compiler.srcsBeforeGen...)
	srcs, genDeps := genSources(ctx, srcs, buildFlags)
	pathDeps = append(pathDeps, genDeps...)
	compiler.pathDeps = pathDeps
	compiler.cFlagsDeps = flags.CFlagsDeps
	compiler.srcs = srcs
	objs := compileObjs(ctx, buildFlags, "", srcs, pathDeps, compiler.cFlagsDeps)

	if ctx.Failed() {
		return Objects{}
	}

	return objs
}

func compileObjs(ctx android.ModuleContext, flags builderFlags, subdir string, srcFiles, pathDeps android.Paths, cFlagsDeps android.Paths) Objects {
	return TransformSourceToObj(ctx, subdir, srcFiles, flags, pathDeps, cFlagsDeps)
}
