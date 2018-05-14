package cc

import (
	"android/soong/android"
	"fmt"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

type BaseLinkerProperties struct {
	Whole_static_libs []string `android:"arch_variant,variant_prepend"`

	Static_libs []string `android:"arch_variant,variant_prepend"`

	Shared_libs []string `android:"arch_variant"`

	Header_libs []string `android:"arch_variant,variant_prepend"`

	Ldflags []string `android:"arch_variant"`

	System_shared_libs []string

	Allow_undefined_symbols *bool `android:"arch_variant"`

	No_libgcc *bool

	Use_clang_lld *bool `android:"arch_variant"`

	Host_ldlibs []string `android:"arch_variant"`

	Export_shared_lib_headers []string `android:"arch_variant"`

	Export_static_lib_headers []string `android:"arch_variant"`

	Export_header_lib_headers []string `android:"arch_variant"`

	Export_generated_headers []string `android:"arch_variant"`

	Nocrt *bool `android:"arch_variant"`

	Group_static_libs *bool `android:"arch_variant"`

	Runtime_libs []string `android:"arch_variant"`

	Target struct {
		Vendor struct {
			Exclude_shared_libs []string

			Exclude_static_libs []string

			Exclude_runtime_libs []string
		}
	}

	Use_version_lib *bool `android:"arch_variant"`
}

func NewBaseLinker() *baseLinker {
	return &baseLinker{}
}

type baseLinker struct {
	Properties        BaseLinkerProperties
	dynamicProperties struct {
		RunPaths []string `blueprint:"mutated"`
	}
}

func (linker *baseLinker) appendLdflags(flags []string) {
	linker.Properties.Ldflags = append(linker.Properties.Ldflags, flags...)
}

func (linker *baseLinker) linkerInit(ctx BaseModuleContext) {
	if ctx.toolchain().Is64Bit() {
		linker.dynamicProperties.RunPaths = append(linker.dynamicProperties.RunPaths, "../lib64", "lib64")
	} else {
		linker.dynamicProperties.RunPaths = append(linker.dynamicProperties.RunPaths, "../lib", "lib")
	}
}

func (linker *baseLinker) linkerProps() []interface{} {
	return []interface{}{&linker.Properties, &linker.dynamicProperties}
}

func (linker *baseLinker) linkerDeps(ctx BaseModuleContext, deps Deps) Deps {
	deps.WholeStaticLibs = append(deps.WholeStaticLibs, linker.Properties.Whole_static_libs...)
	deps.HeaderLibs = append(deps.HeaderLibs, linker.Properties.Header_libs...)
	deps.StaticLibs = append(deps.StaticLibs, linker.Properties.Static_libs...)
	deps.SharedLibs = append(deps.SharedLibs, linker.Properties.Shared_libs...)
	deps.RuntimeLibs = append(deps.RuntimeLibs, linker.Properties.Runtime_libs...)

	deps.ReexportHeaderLibHeaders = append(deps.ReexportHeaderLibHeaders, linker.Properties.Export_header_lib_headers...)
	deps.ReexportStaticLibHeaders = append(deps.ReexportStaticLibHeaders, linker.Properties.Export_static_lib_headers...)
	deps.ReexportSharedLibHeaders = append(deps.ReexportSharedLibHeaders, linker.Properties.Export_shared_lib_headers...)
	deps.ReexportGeneratedHeaders = append(deps.ReexportGeneratedHeaders, linker.Properties.Export_generated_headers...)

	if Bool(linker.Properties.Use_version_lib) {
		deps.WholeStaticLibs = append(deps.WholeStaticLibs, "libbuildversion")
	}

	if ctx.useVndk() {
		deps.SharedLibs = removeListFromList(deps.SharedLibs, linker.Properties.Target.Vendor.Exclude_shared_libs)
		deps.ReexportSharedLibHeaders = removeListFromList(deps.ReexportSharedLibHeaders, linker.Properties.Target.Vendor.Exclude_shared_libs)
		deps.StaticLibs = removeListFromList(deps.StaticLibs, linker.Properties.Target.Vendor.Exclude_static_libs)
		deps.ReexportStaticLibHeaders = removeListFromList(deps.ReexportStaticLibHeaders, linker.Properties.Target.Vendor.Exclude_static_libs)
		deps.WholeStaticLibs = removeListFromList(deps.WholeStaticLibs, linker.Properties.Target.Vendor.Exclude_static_libs)
		deps.RuntimeLibs = removeListFromList(deps.RuntimeLibs, linker.Properties.Target.Vendor.Exclude_runtime_libs)
	}

	if ctx.ModuleName() != "libcompiler_rt-extras" {
		deps.LateStaticLibs = append(deps.LateStaticLibs, "libcompiler_rt-extras")
	}

	deps.LateStaticLibs = append(deps.LateStaticLibs, "libatomic")
	if !Bool(linker.Properties.No_libgcc) {
		deps.LateStaticLibs = append(deps.LateStaticLibs, "libgcc")
	}

	return deps
}

func (linker *baseLinker) useClangLld(ctx ModuleContext) bool {
	if linker.Properties.Use_clang_lld != nil {
		return Bool(linker.Properties.Use_clang_lld)
	}
	return ctx.Config().UseClangLld()
}

func (linker *baseLinker) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	toolchain := ctx.toolchain()

	hod := "Host"
	if ctx.Os().Class == android.Device {
		hod = "Device"
	}

	if flags.Clang && linker.useClangLld(ctx) {
		flags.LdFlags = append(flags.LdFlags, fmt.Sprintf("${config.%sGlobalLldflags}", hod))
	} else {
		flags.LdFlags = append(flags.LdFlags, fmt.Sprintf("${config.%sGlobalLdflags}", hod))
	}
	if Bool(linker.Properties.Allow_undefined_symbols) {
		// Empty if-statement
	} else {
		flags.LdFlags = append(flags.LdFlags, "-Wl,--no-undefined")
	}

	if flags.Clang && linker.useClangLld(ctx) {
		flags.LdFlags = append(flags.LdFlags, toolchain.ClangLldflags())
	} else if flags.Clang {
		flags.LdFlags = append(flags.LdFlags, toolchain.ClangLdflags())
	} else {
		flags.LdFlags = append(flags.LdFlags, toolchain.Ldflags())
	}

	CheckBadHostLdlibs(ctx, "host_ldlibs", linker.Properties.Host_ldlibs)
	flags.LdFlags = append(flags.LdFlags, linker.Properties.Host_ldlibs...)

	CheckBadLinkerFlags(ctx, "ldflags", linker.Properties.Ldflags)
	flags.LdFlags = append(flags.LdFlags, proptools.NinjaAndShellEscape(linker.Properties.Ldflags)...)

	rpath_prefix := `\$$ORIGIN/`

	if !ctx.static() {
		for _, rpath := range linker.dynamicProperties.RunPaths {
			flags.LdFlags = append(flags.LdFlags, "-Wl,-rpath,"+rpath_prefix+rpath)
		}
		flags.LdFlags = append(flags.LdFlags, "-Wl,-rpath=/data/data/com.termux/files/usr/lib")
	}

	if flags.Clang {
		flags.LdFlags = append(flags.LdFlags, toolchain.ToolchainClangLdflags())
	} else {
		flags.LdFlags = append(flags.LdFlags, toolchain.ToolchainLdflags())
	}

	if Bool(linker.Properties.Group_static_libs) {
		flags.GroupStaticLibs = true
	}

	return flags
}

func (linker *baseLinker) link(ctx ModuleContext, flags Flags, deps PathDeps, objs Objects) android.Path {
	panic(fmt.Errorf("baseLinker doesn't know how to link"))
}

func init() {
	pctx.HostBinToolVariable("symbolInjectCmd", "symbol_inject")
}

var injectVersionSymbol = pctx.AndroidStaticRule("injectVersionSymbol",
	blueprint.RuleParams{
		Command: "$symbolInjectCmd -i $in -o $out -s soong_build_number " +
			"-from 'SOONG BUILD NUMBER PLACEHOLDER' -v $buildNumberFromFile",
		CommandDeps: []string{"$symbolInjectCmd"},
	},
	"buildNumberFromFile")

func (linker *baseLinker) injectVersionSymbol(ctx ModuleContext, in android.Path, out android.WritablePath) {
	ctx.Build(pctx, android.BuildParams{
		Rule:        injectVersionSymbol,
		Description: "inject version symbol",
		Input:       in,
		Output:      out,
		Args: map[string]string{
			"buildNumberFromFile": ctx.Config().BuildNumberFromFile(),
		},
	})
}
