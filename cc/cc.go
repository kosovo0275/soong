package cc

import (
	"strconv"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"

	"android/soong/android"
	"android/soong/cc/config"
	"android/soong/genrule"
)

func init() {
	android.RegisterModuleType("cc_defaults", defaultsFactory)

	android.PreDepsMutators(func(ctx android.RegisterMutatorsContext) {
		ctx.BottomUp("image", vendorMutator).Parallel()
		ctx.BottomUp("link", linkageMutator).Parallel()
		ctx.BottomUp("vndk", vndkMutator).Parallel()
		ctx.BottomUp("ndk_api", ndkApiMutator).Parallel()
		ctx.BottomUp("test_per_src", testPerSrcMutator).Parallel()
		ctx.BottomUp("begin", beginMutator).Parallel()
	})

	android.PostDepsMutators(func(ctx android.RegisterMutatorsContext) {
		ctx.TopDown("asan_deps", sanitizerDepsMutator(asan))
		ctx.BottomUp("asan", sanitizerMutator(asan)).Parallel()

		ctx.TopDown("cfi_deps", sanitizerDepsMutator(cfi))
		ctx.BottomUp("cfi", sanitizerMutator(cfi)).Parallel()

		ctx.TopDown("tsan_deps", sanitizerDepsMutator(tsan))
		ctx.BottomUp("tsan", sanitizerMutator(tsan)).Parallel()

		ctx.TopDown("sanitize_runtime_deps", sanitizerRuntimeDepsMutator())

		ctx.BottomUp("coverage", coverageLinkingMutator).Parallel()
		ctx.TopDown("vndk_deps", sabiDepsMutator)

		ctx.TopDown("lto_deps", ltoDepsMutator)
		ctx.BottomUp("lto", ltoMutator).Parallel()
	})

	pctx.Import("android/soong/cc/config")
}

type Deps struct {
	SharedLibs, LateSharedLibs                  []string
	StaticLibs, LateStaticLibs, WholeStaticLibs []string
	HeaderLibs                                  []string
	RuntimeLibs                                 []string

	ReexportSharedLibHeaders, ReexportStaticLibHeaders, ReexportHeaderLibHeaders []string

	ObjFiles []string

	GeneratedSources []string
	GeneratedHeaders []string

	ReexportGeneratedHeaders []string

	LinkerScript string
}

type PathDeps struct {
	SharedLibs, LateSharedLibs                  android.Paths
	SharedLibsDeps, LateSharedLibsDeps          android.Paths
	StaticLibs, LateStaticLibs, WholeStaticLibs android.Paths

	Objs               Objects
	StaticLibObjs      Objects
	WholeStaticLibObjs Objects

	GeneratedSources android.Paths
	GeneratedHeaders android.Paths

	Flags, ReexportedFlags []string
	ReexportedFlagsDeps    android.Paths

	LinkerScript android.OptionalPath
}

type Flags struct {
	GlobalFlags     []string
	ArFlags         []string
	AsFlags         []string
	CFlags          []string
	ToolingCFlags   []string
	ConlyFlags      []string
	CppFlags        []string
	ToolingCppFlags []string
	YaccFlags       []string
	protoFlags      []string
	protoOutParams  []string
	aidlFlags       []string
	rsFlags         []string
	LdFlags         []string
	libFlags        []string
	TidyFlags       []string
	SAbiFlags       []string
	YasmFlags       []string

	SystemIncludeFlags []string

	Toolchain config.Toolchain
	Clang     bool
	Tidy      bool
	Coverage  bool
	SAbiDump  bool
	ProtoRoot bool

	RequiredInstructionSet string
	DynamicLinker          string

	CFlagsDeps  android.Paths
	LdFlagsDeps android.Paths

	GroupStaticLibs bool
	ArGoldPlugin    bool
}

type ObjectLinkerProperties struct {
	Objs []string `android:"arch_variant"`

	Prefix_symbols *string
}

type BaseProperties struct {
	Clang *bool `android:"arch_variant"`

	Sdk_version *string

	AndroidMkSharedLibs  []string `blueprint:"mutated"`
	AndroidMkRuntimeLibs []string `blueprint:"mutated"`
	HideFromMake         bool     `blueprint:"mutated"`
	PreventInstall       bool     `blueprint:"mutated"`

	UseVndk bool `blueprint:"mutated"`

	Logtags []string
}

type VendorProperties struct {
	Vendor_available *bool
	Double_loadable  *bool
}

type UnusedProperties struct {
	Tags []string
}

type ModuleContextIntf interface {
	static() bool
	staticBinary() bool
	clang() bool
	toolchain() config.Toolchain
	useSdk() bool
	sdkVersion() string
	useVndk() bool
	isVndk() bool
	isVndkSp() bool
	isVndkExt() bool
	createVndkSourceAbiDump() bool
	selectedStl() string
	baseModuleName() string
	getVndkExtendsModuleName() string
	isPgoCompile() bool
}

type ModuleContext interface {
	android.ModuleContext
	ModuleContextIntf
}

type BaseModuleContext interface {
	android.BaseContext
	ModuleContextIntf
}

type DepsContext interface {
	android.BottomUpMutatorContext
	ModuleContextIntf
}

type feature interface {
	begin(ctx BaseModuleContext)
	deps(ctx DepsContext, deps Deps) Deps
	flags(ctx ModuleContext, flags Flags) Flags
	props() []interface{}
}

type compiler interface {
	compilerInit(ctx BaseModuleContext)
	compilerDeps(ctx DepsContext, deps Deps) Deps
	compilerFlags(ctx ModuleContext, flags Flags, deps PathDeps) Flags
	compilerProps() []interface{}

	appendCflags([]string)
	appendAsflags([]string)
	compile(ctx ModuleContext, flags Flags, deps PathDeps) Objects
}

type linker interface {
	linkerInit(ctx BaseModuleContext)
	linkerDeps(ctx DepsContext, deps Deps) Deps
	linkerFlags(ctx ModuleContext, flags Flags) Flags
	linkerProps() []interface{}

	link(ctx ModuleContext, flags Flags, deps PathDeps, objs Objects) android.Path
	appendLdflags([]string)
}

type installer interface {
	installerProps() []interface{}
	install(ctx ModuleContext, path android.Path)
	inData() bool
	inSanitizerDir() bool
	hostToolPath() android.OptionalPath
}

type dependencyTag struct {
	blueprint.BaseDependencyTag
	name    string
	library bool

	reexportFlags bool
}

var (
	sharedDepTag          = dependencyTag{name: "shared", library: true}
	sharedExportDepTag    = dependencyTag{name: "shared", library: true, reexportFlags: true}
	lateSharedDepTag      = dependencyTag{name: "late shared", library: true}
	staticDepTag          = dependencyTag{name: "static", library: true}
	staticExportDepTag    = dependencyTag{name: "static", library: true, reexportFlags: true}
	lateStaticDepTag      = dependencyTag{name: "late static", library: true}
	wholeStaticDepTag     = dependencyTag{name: "whole static", library: true, reexportFlags: true}
	headerDepTag          = dependencyTag{name: "header", library: true}
	headerExportDepTag    = dependencyTag{name: "header", library: true, reexportFlags: true}
	genSourceDepTag       = dependencyTag{name: "gen source"}
	genHeaderDepTag       = dependencyTag{name: "gen header"}
	genHeaderExportDepTag = dependencyTag{name: "gen header", reexportFlags: true}
	objDepTag             = dependencyTag{name: "obj"}
	linkerScriptDepTag    = dependencyTag{name: "linker script"}
	reuseObjTag           = dependencyTag{name: "reuse objects"}
	ndkStubDepTag         = dependencyTag{name: "ndk stub", library: true}
	ndkLateStubDepTag     = dependencyTag{name: "ndk late stub", library: true}
	vndkExtDepTag         = dependencyTag{name: "vndk extends", library: true}
	runtimeDepTag         = dependencyTag{name: "runtime lib"}
)

type Module struct {
	android.ModuleBase
	android.DefaultableModuleBase

	Properties       BaseProperties
	VendorProperties VendorProperties
	unused           UnusedProperties

	hod      android.HostOrDeviceSupported
	multilib android.Multilib

	features  []feature
	compiler  compiler
	linker    linker
	installer installer
	stl       *stl
	sanitize  *sanitize
	coverage  *coverage
	sabi      *sabi
	vndkdep   *vndkdep
	lto       *lto
	pgo       *pgo

	androidMkSharedLibDeps []string

	outputFile android.OptionalPath

	cachedToolchain config.Toolchain

	subAndroidMkOnce map[subAndroidMkProvider]bool

	flags Flags

	depsInLinkOrder android.Paths

	staticVariant *Module
}

func (c *Module) Init() android.Module {
	c.AddProperties(&c.Properties, &c.VendorProperties, &c.unused)
	if c.compiler != nil {
		c.AddProperties(c.compiler.compilerProps()...)
	}
	if c.linker != nil {
		c.AddProperties(c.linker.linkerProps()...)
	}
	if c.installer != nil {
		c.AddProperties(c.installer.installerProps()...)
	}
	if c.stl != nil {
		c.AddProperties(c.stl.props()...)
	}
	if c.sanitize != nil {
		c.AddProperties(c.sanitize.props()...)
	}
	if c.coverage != nil {
		c.AddProperties(c.coverage.props()...)
	}
	if c.sabi != nil {
		c.AddProperties(c.sabi.props()...)
	}
	if c.vndkdep != nil {
		c.AddProperties(c.vndkdep.props()...)
	}
	if c.lto != nil {
		c.AddProperties(c.lto.props()...)
	}
	if c.pgo != nil {
		c.AddProperties(c.pgo.props()...)
	}
	for _, feature := range c.features {
		c.AddProperties(feature.props()...)
	}

	android.InitAndroidArchModule(c, c.hod, c.multilib)

	android.InitDefaultableModule(c)

	return c
}

func (c *Module) isDependencyRoot() bool {
	if root, ok := c.linker.(interface {
		isDependencyRoot() bool
	}); ok {
		return root.isDependencyRoot()
	}
	return false
}

func (c *Module) useVndk() bool {
	return c.Properties.UseVndk
}

func (c *Module) isVndk() bool {
	if vndkdep := c.vndkdep; vndkdep != nil {
		return vndkdep.isVndk()
	}
	return false
}

func (c *Module) isPgoCompile() bool {
	if pgo := c.pgo; pgo != nil {
		return pgo.Properties.PgoCompile
	}
	return false
}

func (c *Module) isVndkSp() bool {
	if vndkdep := c.vndkdep; vndkdep != nil {
		return vndkdep.isVndkSp()
	}
	return false
}

func (c *Module) isVndkExt() bool {
	if vndkdep := c.vndkdep; vndkdep != nil {
		return vndkdep.isVndkExt()
	}
	return false
}

func (c *Module) getVndkExtendsModuleName() string {
	if vndkdep := c.vndkdep; vndkdep != nil {
		return vndkdep.getVndkExtendsModuleName()
	}
	return ""
}

func (c *Module) hasVendorVariant() bool {
	return c.isVndk() || Bool(c.VendorProperties.Vendor_available)
}

type baseModuleContext struct {
	android.BaseContext
	moduleContextImpl
}

type depsContext struct {
	android.BottomUpMutatorContext
	moduleContextImpl
}

type moduleContext struct {
	android.ModuleContext
	moduleContextImpl
}

func (ctx *moduleContext) SocSpecific() bool {
	return ctx.ModuleContext.SocSpecific() ||
		(ctx.mod.hasVendorVariant() && ctx.mod.useVndk() && !ctx.mod.isVndk())
}

type moduleContextImpl struct {
	mod *Module
	ctx BaseModuleContext
}

func (ctx *moduleContextImpl) clang() bool {
	return ctx.mod.clang(ctx.ctx)
}

func (ctx *moduleContextImpl) toolchain() config.Toolchain {
	return ctx.mod.toolchain(ctx.ctx)
}

func (ctx *moduleContextImpl) static() bool {
	return ctx.mod.static()
}

func (ctx *moduleContextImpl) staticBinary() bool {
	if static, ok := ctx.mod.linker.(interface {
		staticBinary() bool
	}); ok {
		return static.staticBinary()
	}
	return false
}

func (ctx *moduleContextImpl) useSdk() bool {
	if !ctx.useVndk() {
		return String(ctx.mod.Properties.Sdk_version) != ""
	}
	return false
}

func (ctx *moduleContextImpl) sdkVersion() string {
	//	if ctx.ctx.Device() {
	return String(ctx.mod.Properties.Sdk_version)
	//	}
	//	return ""
}

func (ctx *moduleContextImpl) useVndk() bool {
	return ctx.mod.useVndk()
}

func (ctx *moduleContextImpl) isVndk() bool {
	return ctx.mod.isVndk()
}

func (ctx *moduleContextImpl) isPgoCompile() bool {
	return ctx.mod.isPgoCompile()
}

func (ctx *moduleContextImpl) isVndkSp() bool {
	return ctx.mod.isVndkSp()
}

func (ctx *moduleContextImpl) isVndkExt() bool {
	return ctx.mod.isVndkExt()
}

func (ctx *moduleContextImpl) createVndkSourceAbiDump() bool {
	skipAbiChecks := ctx.ctx.Config().IsEnvTrue("SKIP_ABI_CHECKS")
	isUnsanitizedVariant := true
	sanitize := ctx.mod.sanitize
	if sanitize != nil {
		isUnsanitizedVariant = sanitize.isUnsanitizedVariant()
	}
	vendorAvailable := Bool(ctx.mod.VendorProperties.Vendor_available)
	return !skipAbiChecks && isUnsanitizedVariant && ctx.ctx.Device() && ((ctx.useVndk() && ctx.isVndk() && vendorAvailable) || inList(ctx.baseModuleName(), llndkLibraries))
}

func (ctx *moduleContextImpl) selectedStl() string {
	if stl := ctx.mod.stl; stl != nil {
		return stl.Properties.SelectedStl
	}
	return ""
}

func (ctx *moduleContextImpl) baseModuleName() string {
	return ctx.mod.ModuleBase.BaseModuleName()
}

func (ctx *moduleContextImpl) getVndkExtendsModuleName() string {
	return ctx.mod.getVndkExtendsModuleName()
}

func newBaseModule(hod android.HostOrDeviceSupported, multilib android.Multilib) *Module {
	return &Module{
		hod:      hod,
		multilib: multilib,
	}
}

func newModule(hod android.HostOrDeviceSupported, multilib android.Multilib) *Module {
	module := newBaseModule(hod, multilib)
	module.features = []feature{
		&tidyFeature{},
	}
	module.stl = &stl{}
	module.sanitize = &sanitize{}
	module.coverage = &coverage{}
	module.sabi = &sabi{}
	module.vndkdep = &vndkdep{}
	module.lto = &lto{}
	module.pgo = &pgo{}
	return module
}

func (c *Module) Prebuilt() *android.Prebuilt {
	if p, ok := c.linker.(prebuiltLinkerInterface); ok {
		return p.prebuilt()
	}
	return nil
}

func (c *Module) Name() string {
	name := c.ModuleBase.Name()
	if p, ok := c.linker.(interface {
		Name(string) string
	}); ok {
		name = p.Name(name)
	}
	return name
}

func orderDeps(directStaticDeps []android.Path, directSharedDeps []android.Path, allTransitiveDeps map[android.Path][]android.Path) (orderedAllDeps []android.Path, orderedDeclaredDeps []android.Path) {
	for _, dep := range directStaticDeps {
		orderedAllDeps = append(orderedAllDeps, dep)
		orderedAllDeps = append(orderedAllDeps, allTransitiveDeps[dep]...)
	}
	for _, dep := range directSharedDeps {
		orderedAllDeps = append(orderedAllDeps, dep)
		orderedAllDeps = append(orderedAllDeps, allTransitiveDeps[dep]...)
	}

	orderedAllDeps = android.LastUniquePaths(orderedAllDeps)

	_, orderedDeclaredDeps = android.FilterPathList(orderedAllDeps, directStaticDeps)

	return orderedAllDeps, orderedDeclaredDeps
}

func orderStaticModuleDeps(module *Module, staticDeps []*Module, sharedDeps []*Module) (results []android.Path) {
	allTransitiveDeps := make(map[android.Path][]android.Path, len(staticDeps))
	staticDepFiles := []android.Path{}
	for _, dep := range staticDeps {
		allTransitiveDeps[dep.outputFile.Path()] = dep.depsInLinkOrder
		staticDepFiles = append(staticDepFiles, dep.outputFile.Path())
	}
	sharedDepFiles := []android.Path{}
	for _, sharedDep := range sharedDeps {
		staticAnalogue := sharedDep.staticVariant
		if staticAnalogue != nil {
			allTransitiveDeps[staticAnalogue.outputFile.Path()] = staticAnalogue.depsInLinkOrder
			sharedDepFiles = append(sharedDepFiles, staticAnalogue.outputFile.Path())
		}
	}

	module.depsInLinkOrder, results = orderDeps(staticDepFiles, sharedDepFiles, allTransitiveDeps)

	return results
}

func (c *Module) GenerateAndroidBuildActions(actx android.ModuleContext) {
	ctx := &moduleContext{
		ModuleContext: actx,
		moduleContextImpl: moduleContextImpl{
			mod: c,
		},
	}
	ctx.ctx = ctx

	deps := c.depsToPaths(ctx)
	if ctx.Failed() {
		return
	}

	flags := Flags{
		Toolchain: c.toolchain(ctx),
		Clang:     c.clang(ctx),
	}
	if c.compiler != nil {
		flags = c.compiler.compilerFlags(ctx, flags, deps)
	}
	if c.linker != nil {
		flags = c.linker.linkerFlags(ctx, flags)
	}
	if c.stl != nil {
		flags = c.stl.flags(ctx, flags)
	}
	if c.sanitize != nil {
		flags = c.sanitize.flags(ctx, flags)
	}
	if c.coverage != nil {
		flags = c.coverage.flags(ctx, flags)
	}
	if c.lto != nil {
		flags = c.lto.flags(ctx, flags)
	}
	if c.pgo != nil {
		flags = c.pgo.flags(ctx, flags)
	}
	for _, feature := range c.features {
		flags = feature.flags(ctx, flags)
	}
	if ctx.Failed() {
		return
	}

	flags.CFlags, _ = filterList(flags.CFlags, config.IllegalFlags)
	flags.CppFlags, _ = filterList(flags.CppFlags, config.IllegalFlags)
	flags.ConlyFlags, _ = filterList(flags.ConlyFlags, config.IllegalFlags)

	flags.GlobalFlags = append(flags.GlobalFlags, deps.Flags...)
	c.flags = flags
	if c.sabi != nil {
		flags = c.sabi.flags(ctx, flags)
	}
	ctx.Variable(pctx, "cflags", strings.Join(flags.CFlags, " "))
	ctx.Variable(pctx, "cppflags", strings.Join(flags.CppFlags, " "))
	ctx.Variable(pctx, "asflags", strings.Join(flags.AsFlags, " "))
	flags.CFlags = []string{"$cflags"}
	flags.CppFlags = []string{"$cppflags"}
	flags.AsFlags = []string{"$asflags"}

	var objs Objects
	if c.compiler != nil {
		objs = c.compiler.compile(ctx, flags, deps)
		if ctx.Failed() {
			return
		}
	}

	if c.linker != nil {
		outputFile := c.linker.link(ctx, flags, deps, objs)
		if ctx.Failed() {
			return
		}
		c.outputFile = android.OptionalPathForPath(outputFile)
	}

	if c.installer != nil && !c.Properties.PreventInstall && c.outputFile.Valid() {
		c.installer.install(ctx, c.outputFile.Path())
		if ctx.Failed() {
			return
		}
	}
}

func (c *Module) toolchain(ctx BaseModuleContext) config.Toolchain {
	if c.cachedToolchain == nil {
		c.cachedToolchain = config.FindToolchain(ctx.Os(), ctx.Arch())
	}
	return c.cachedToolchain
}

func (c *Module) begin(ctx BaseModuleContext) {
	if c.compiler != nil {
		c.compiler.compilerInit(ctx)
	}
	if c.linker != nil {
		c.linker.linkerInit(ctx)
	}
	if c.stl != nil {
		c.stl.begin(ctx)
	}
	if c.sanitize != nil {
		c.sanitize.begin(ctx)
	}
	if c.coverage != nil {
		c.coverage.begin(ctx)
	}
	if c.sabi != nil {
		c.sabi.begin(ctx)
	}
	if c.vndkdep != nil {
		c.vndkdep.begin(ctx)
	}
	if c.lto != nil {
		c.lto.begin(ctx)
	}
	if c.pgo != nil {
		c.pgo.begin(ctx)
	}
	for _, feature := range c.features {
		feature.begin(ctx)
	}
	if ctx.useSdk() {
		version, err := normalizeNdkApiLevel(ctx, ctx.sdkVersion(), ctx.Arch())
		if err != nil {
			ctx.PropertyErrorf("sdk_version", err.Error())
		}
		c.Properties.Sdk_version = StringPtr(version)
	}
}

func (c *Module) deps(ctx DepsContext) Deps {
	deps := Deps{}

	if c.compiler != nil {
		deps = c.compiler.compilerDeps(ctx, deps)
	}
	if c.pgo != nil {
		deps = c.pgo.deps(ctx, deps)
	}
	if c.linker != nil {
		deps = c.linker.linkerDeps(ctx, deps)
	}
	if c.stl != nil {
		deps = c.stl.deps(ctx, deps)
	}
	if c.sanitize != nil {
		deps = c.sanitize.deps(ctx, deps)
	}
	if c.coverage != nil {
		deps = c.coverage.deps(ctx, deps)
	}
	if c.sabi != nil {
		deps = c.sabi.deps(ctx, deps)
	}
	if c.vndkdep != nil {
		deps = c.vndkdep.deps(ctx, deps)
	}
	if c.lto != nil {
		deps = c.lto.deps(ctx, deps)
	}
	for _, feature := range c.features {
		deps = feature.deps(ctx, deps)
	}

	deps.WholeStaticLibs = android.LastUniqueStrings(deps.WholeStaticLibs)
	deps.StaticLibs = android.LastUniqueStrings(deps.StaticLibs)
	deps.LateStaticLibs = android.LastUniqueStrings(deps.LateStaticLibs)
	deps.SharedLibs = android.LastUniqueStrings(deps.SharedLibs)
	deps.LateSharedLibs = android.LastUniqueStrings(deps.LateSharedLibs)
	deps.HeaderLibs = android.LastUniqueStrings(deps.HeaderLibs)
	deps.RuntimeLibs = android.LastUniqueStrings(deps.RuntimeLibs)

	for _, lib := range deps.ReexportSharedLibHeaders {
		if !inList(lib, deps.SharedLibs) {
			ctx.PropertyErrorf("export_shared_lib_headers", "Shared library not in shared_libs: '%s'", lib)
		}
	}

	for _, lib := range deps.ReexportStaticLibHeaders {
		if !inList(lib, deps.StaticLibs) {
			ctx.PropertyErrorf("export_static_lib_headers", "Static library not in static_libs: '%s'", lib)
		}
	}

	for _, lib := range deps.ReexportHeaderLibHeaders {
		if !inList(lib, deps.HeaderLibs) {
			ctx.PropertyErrorf("export_header_lib_headers", "Header library not in header_libs: '%s'", lib)
		}
	}

	for _, gen := range deps.ReexportGeneratedHeaders {
		if !inList(gen, deps.GeneratedHeaders) {
			ctx.PropertyErrorf("export_generated_headers", "Generated header module not in generated_headers: '%s'", gen)
		}
	}

	return deps
}

func (c *Module) beginMutator(actx android.BottomUpMutatorContext) {
	ctx := &baseModuleContext{
		BaseContext: actx,
		moduleContextImpl: moduleContextImpl{
			mod: c,
		},
	}
	ctx.ctx = ctx

	c.begin(ctx)
}

func (c *Module) DepsMutator(actx android.BottomUpMutatorContext) {
	if !c.Enabled() {
		return
	}

	ctx := &depsContext{
		BottomUpMutatorContext: actx,
		moduleContextImpl: moduleContextImpl{
			mod: c,
		},
	}
	ctx.ctx = ctx

	deps := c.deps(ctx)

	variantNdkLibs := []string{}
	variantLateNdkLibs := []string{}
	if ctx.Os() == android.Android {
		version := ctx.sdkVersion()

		rewriteNdkLibs := func(list []string) (nonvariantLibs []string, variantLibs []string) {
			variantLibs = []string{}
			nonvariantLibs = []string{}
			for _, entry := range list {
				if ctx.useSdk() && inList(entry, ndkPrebuiltSharedLibraries) {
					if !inList(entry, ndkMigratedLibs) {
						nonvariantLibs = append(nonvariantLibs, entry+".ndk."+version)
					} else {
						variantLibs = append(variantLibs, entry+ndkLibrarySuffix)
					}
				} else if ctx.useVndk() && inList(entry, llndkLibraries) {
					nonvariantLibs = append(nonvariantLibs, entry+llndkLibrarySuffix)
				} else if (ctx.Platform() || ctx.ProductSpecific()) && inList(entry, vendorPublicLibraries) {
					vendorPublicLib := entry + vendorPublicLibrarySuffix
					if actx.OtherModuleExists(vendorPublicLib) {
						nonvariantLibs = append(nonvariantLibs, vendorPublicLib)
					} else {
						nonvariantLibs = append(nonvariantLibs, entry)
					}
				} else {
					nonvariantLibs = append(nonvariantLibs, entry)
				}
			}
			return nonvariantLibs, variantLibs
		}

		deps.SharedLibs, variantNdkLibs = rewriteNdkLibs(deps.SharedLibs)
		deps.LateSharedLibs, variantLateNdkLibs = rewriteNdkLibs(deps.LateSharedLibs)
		deps.ReexportSharedLibHeaders, _ = rewriteNdkLibs(deps.ReexportSharedLibHeaders)
	}

	for _, lib := range deps.HeaderLibs {
		depTag := headerDepTag
		if inList(lib, deps.ReexportHeaderLibHeaders) {
			depTag = headerExportDepTag
		}
		actx.AddVariationDependencies(nil, depTag, lib)
	}

	actx.AddVariationDependencies([]blueprint.Variation{{"link", "static"}}, wholeStaticDepTag,
		deps.WholeStaticLibs...)

	for _, lib := range deps.StaticLibs {
		depTag := staticDepTag
		if inList(lib, deps.ReexportStaticLibHeaders) {
			depTag = staticExportDepTag
		}
		actx.AddVariationDependencies([]blueprint.Variation{{"link", "static"}}, depTag, lib)
	}

	actx.AddVariationDependencies([]blueprint.Variation{{"link", "static"}}, lateStaticDepTag,
		deps.LateStaticLibs...)

	for _, lib := range deps.SharedLibs {
		depTag := sharedDepTag
		if inList(lib, deps.ReexportSharedLibHeaders) {
			depTag = sharedExportDepTag
		}
		actx.AddVariationDependencies([]blueprint.Variation{{"link", "shared"}}, depTag, lib)
	}

	actx.AddVariationDependencies([]blueprint.Variation{{"link", "shared"}}, lateSharedDepTag,
		deps.LateSharedLibs...)

	actx.AddVariationDependencies([]blueprint.Variation{{"link", "shared"}}, runtimeDepTag,
		deps.RuntimeLibs...)

	actx.AddDependency(c, genSourceDepTag, deps.GeneratedSources...)

	for _, gen := range deps.GeneratedHeaders {
		depTag := genHeaderDepTag
		if inList(gen, deps.ReexportGeneratedHeaders) {
			depTag = genHeaderExportDepTag
		}
		actx.AddDependency(c, depTag, gen)
	}

	actx.AddDependency(c, objDepTag, deps.ObjFiles...)

	if deps.LinkerScript != "" {
		actx.AddDependency(c, linkerScriptDepTag, deps.LinkerScript)
	}

	version := ctx.sdkVersion()
	actx.AddVariationDependencies([]blueprint.Variation{
		{"ndk_api", version}, {"link", "shared"}}, ndkStubDepTag, variantNdkLibs...)
	actx.AddVariationDependencies([]blueprint.Variation{
		{"ndk_api", version}, {"link", "shared"}}, ndkLateStubDepTag, variantLateNdkLibs...)

	if vndkdep := c.vndkdep; vndkdep != nil {
		if vndkdep.isVndkExt() {
			baseModuleMode := vendorMode
			if actx.DeviceConfig().VndkVersion() == "" {
				baseModuleMode = coreMode
			}
			actx.AddVariationDependencies([]blueprint.Variation{
				{"image", baseModuleMode}, {"link", "shared"}}, vndkExtDepTag,
				vndkdep.getVndkExtendsModuleName())
		}
	}
}

func beginMutator(ctx android.BottomUpMutatorContext) {
	if c, ok := ctx.Module().(*Module); ok && c.Enabled() {
		c.beginMutator(ctx)
	}
}

func (c *Module) clang(ctx BaseModuleContext) bool {
	clang := Bool(c.Properties.Clang)

	if c.Properties.Clang == nil {
		clang = true
	}

	if !c.toolchain(ctx).ClangSupported() {
		clang = false
	}

	return clang
}

func checkLinkType(ctx android.ModuleContext, from *Module, to *Module, tag dependencyTag) {
	return
}

func checkDoubleLoadableLibries(ctx android.ModuleContext, from *Module, to *Module) {
	if _, ok := from.linker.(*libraryDecorator); !ok {
		return
	}

	if inList(ctx.ModuleName(), llndkLibraries) || (from.useVndk() && Bool(from.VendorProperties.Double_loadable)) {
		_, depIsLlndk := to.linker.(*llndkStubDecorator)
		depIsVndkSp := false
		if to.vndkdep != nil && to.vndkdep.isVndkSp() {
			depIsVndkSp = true
		}
		depIsVndk := false
		if to.vndkdep != nil && to.vndkdep.isVndk() {
			depIsVndk = true
		}
		depIsDoubleLoadable := Bool(to.VendorProperties.Double_loadable)
		if !depIsLlndk && !depIsVndkSp && !depIsDoubleLoadable && depIsVndk {
			ctx.ModuleErrorf("links VNDK library %q that isn't double_loadable.", ctx.OtherModuleName(to))
		}
	}
}

func (c *Module) depsToPaths(ctx android.ModuleContext) PathDeps {
	var depPaths PathDeps

	directStaticDeps := []*Module{}
	directSharedDeps := []*Module{}

	ctx.VisitDirectDeps(func(dep android.Module) {
		depName := ctx.OtherModuleName(dep)
		depTag := ctx.OtherModuleDependencyTag(dep)

		ccDep, _ := dep.(*Module)
		if ccDep == nil {

			switch depTag {
			case android.DefaultsDepTag, android.SourceDepTag:

			case genSourceDepTag:
				if genRule, ok := dep.(genrule.SourceFileGenerator); ok {
					depPaths.GeneratedSources = append(depPaths.GeneratedSources,
						genRule.GeneratedSourceFiles()...)
				} else {
					ctx.ModuleErrorf("module %q is not a gensrcs or genrule", depName)
				}

				fallthrough
			case genHeaderDepTag, genHeaderExportDepTag:
				if genRule, ok := dep.(genrule.SourceFileGenerator); ok {
					depPaths.GeneratedHeaders = append(depPaths.GeneratedHeaders,
						genRule.GeneratedDeps()...)
					flags := includeDirsToFlags(genRule.GeneratedHeaderDirs())
					depPaths.Flags = append(depPaths.Flags, flags)
					if depTag == genHeaderExportDepTag {
						depPaths.ReexportedFlags = append(depPaths.ReexportedFlags, flags)
						depPaths.ReexportedFlagsDeps = append(depPaths.ReexportedFlagsDeps,
							genRule.GeneratedDeps()...)

						c.sabi.Properties.ReexportedIncludeFlags = append(c.sabi.Properties.ReexportedIncludeFlags, flags)

					}
				} else {
					ctx.ModuleErrorf("module %q is not a genrule", depName)
				}
			case linkerScriptDepTag:
				if genRule, ok := dep.(genrule.SourceFileGenerator); ok {
					files := genRule.GeneratedSourceFiles()
					if len(files) == 1 {
						depPaths.LinkerScript = android.OptionalPathForPath(files[0])
					} else if len(files) > 1 {
						ctx.ModuleErrorf("module %q can only generate a single file if used for a linker script", depName)
					}
				} else {
					ctx.ModuleErrorf("module %q is not a genrule", depName)
				}
			default:
				ctx.ModuleErrorf("depends on non-cc module %q", depName)
			}
			return
		}

		if dep.Target().Os != ctx.Os() {
			ctx.ModuleErrorf("OS mismatch between %q and %q", ctx.ModuleName(), depName)
			return
		}
		if dep.Target().Arch.ArchType != ctx.Arch().ArchType {
			ctx.ModuleErrorf("Arch mismatch between %q and %q", ctx.ModuleName(), depName)
			return
		}

		if depTag == reuseObjTag {
			if l, ok := ccDep.compiler.(libraryInterface); ok {
				c.staticVariant = ccDep
				objs, flags, deps := l.reuseObjs()
				depPaths.Objs = depPaths.Objs.Append(objs)
				depPaths.ReexportedFlags = append(depPaths.ReexportedFlags, flags...)
				depPaths.ReexportedFlagsDeps = append(depPaths.ReexportedFlagsDeps, deps...)
				return
			}
		}
		if t, ok := depTag.(dependencyTag); ok && t.library {
			if i, ok := ccDep.linker.(exportedFlagsProducer); ok {
				flags := i.exportedFlags()
				deps := i.exportedFlagsDeps()
				depPaths.Flags = append(depPaths.Flags, flags...)
				depPaths.GeneratedHeaders = append(depPaths.GeneratedHeaders, deps...)

				if t.reexportFlags {
					depPaths.ReexportedFlags = append(depPaths.ReexportedFlags, flags...)
					depPaths.ReexportedFlagsDeps = append(depPaths.ReexportedFlagsDeps, deps...)

					c.sabi.Properties.ReexportedIncludeFlags = append(c.sabi.Properties.ReexportedIncludeFlags, flags...)
				}
			}

			checkLinkType(ctx, c, ccDep, t)
			checkDoubleLoadableLibries(ctx, c, ccDep)
		}

		var ptr *android.Paths
		var depPtr *android.Paths

		linkFile := ccDep.outputFile
		depFile := android.OptionalPath{}

		switch depTag {
		case ndkStubDepTag, sharedDepTag, sharedExportDepTag:
			ptr = &depPaths.SharedLibs
			depPtr = &depPaths.SharedLibsDeps
			depFile = ccDep.linker.(libraryInterface).toc()
			directSharedDeps = append(directSharedDeps, ccDep)
		case lateSharedDepTag, ndkLateStubDepTag:
			ptr = &depPaths.LateSharedLibs
			depPtr = &depPaths.LateSharedLibsDeps
			depFile = ccDep.linker.(libraryInterface).toc()
		case staticDepTag, staticExportDepTag:
			ptr = nil
			directStaticDeps = append(directStaticDeps, ccDep)
		case lateStaticDepTag:
			ptr = &depPaths.LateStaticLibs
		case wholeStaticDepTag:
			ptr = &depPaths.WholeStaticLibs
			staticLib, ok := ccDep.linker.(libraryInterface)
			if !ok || !staticLib.static() {
				ctx.ModuleErrorf("module %q not a static library", depName)
				return
			}

			if missingDeps := staticLib.getWholeStaticMissingDeps(); missingDeps != nil {
				postfix := " (required by " + ctx.OtherModuleName(dep) + ")"
				for i := range missingDeps {
					missingDeps[i] += postfix
				}
				ctx.AddMissingDependencies(missingDeps)
			}
			depPaths.WholeStaticLibObjs = depPaths.WholeStaticLibObjs.Append(staticLib.objs())
		case headerDepTag:
			// Nothing
		case objDepTag:
			depPaths.Objs.objFiles = append(depPaths.Objs.objFiles, linkFile.Path())
		}

		switch depTag {
		case staticDepTag, staticExportDepTag, lateStaticDepTag:
			staticLib, ok := ccDep.linker.(libraryInterface)
			if !ok || !staticLib.static() {
				ctx.ModuleErrorf("module %q not a static library", depName)
				return
			}

			depPaths.StaticLibObjs.coverageFiles = append(depPaths.StaticLibObjs.coverageFiles, staticLib.objs().coverageFiles...)
			depPaths.StaticLibObjs.sAbiDumpFiles = append(depPaths.StaticLibObjs.sAbiDumpFiles, staticLib.objs().sAbiDumpFiles...)

		}

		if ptr != nil {
			if !linkFile.Valid() {
				ctx.ModuleErrorf("module %q missing output file", depName)
				return
			}
			*ptr = append(*ptr, linkFile.Path())
		}

		if depPtr != nil {
			dep := depFile
			if !dep.Valid() {
				dep = linkFile
			}
			*depPtr = append(*depPtr, dep.Path())
		}

		makeLibName := func(depName string) string {
			libName := strings.TrimSuffix(depName, llndkLibrarySuffix)
			libName = strings.TrimSuffix(libName, vendorPublicLibrarySuffix)
			libName = strings.TrimPrefix(libName, "prebuilt_")
			isLLndk := inList(libName, llndkLibraries)
			isVendorPublicLib := inList(libName, vendorPublicLibraries)
			bothVendorAndCoreVariantsExist := ccDep.hasVendorVariant() || isLLndk
			if c.useVndk() && bothVendorAndCoreVariantsExist {

				return libName + vendorSuffix
			} else if (ctx.Platform() || ctx.ProductSpecific()) && isVendorPublicLib {
				return libName + vendorPublicLibrarySuffix
			} else {
				return libName
			}
		}

		switch depTag {
		case sharedDepTag, sharedExportDepTag, lateSharedDepTag:
			c.Properties.AndroidMkSharedLibs = append(c.Properties.AndroidMkSharedLibs, makeLibName(depName))
		case runtimeDepTag:
			c.Properties.AndroidMkRuntimeLibs = append(c.Properties.AndroidMkRuntimeLibs, makeLibName(depName))
		}
	})

	depPaths.StaticLibs = append(depPaths.StaticLibs, orderStaticModuleDeps(c, directStaticDeps, directSharedDeps)...)

	depPaths.Flags = android.FirstUniqueStrings(depPaths.Flags)
	depPaths.GeneratedHeaders = android.FirstUniquePaths(depPaths.GeneratedHeaders)
	depPaths.ReexportedFlags = android.FirstUniqueStrings(depPaths.ReexportedFlags)
	depPaths.ReexportedFlagsDeps = android.FirstUniquePaths(depPaths.ReexportedFlagsDeps)

	if c.sabi != nil {
		c.sabi.Properties.ReexportedIncludeFlags = android.FirstUniqueStrings(c.sabi.Properties.ReexportedIncludeFlags)
	}

	return depPaths
}

func (c *Module) InstallInData() bool {
	if c.installer == nil {
		return false
	}
	return c.installer.inData()
}

func (c *Module) InstallInSanitizerDir() bool {
	if c.installer == nil {
		return false
	}
	if c.sanitize != nil && c.sanitize.inSanitizerDir() {
		return true
	}
	return c.installer.inSanitizerDir()
}

func (c *Module) HostToolPath() android.OptionalPath {
	if c.installer == nil {
		return android.OptionalPath{}
	}
	return c.installer.hostToolPath()
}

func (c *Module) IntermPathForModuleOut() android.OptionalPath {
	return c.outputFile
}

func (c *Module) Srcs() android.Paths {
	if c.outputFile.Valid() {
		return android.Paths{c.outputFile.Path()}
	}
	return android.Paths{}
}

func (c *Module) static() bool {
	if static, ok := c.linker.(interface {
		static() bool
	}); ok {
		return static.static()
	}
	return false
}

type Defaults struct {
	android.ModuleBase
	android.DefaultsModuleBase
}

func (*Defaults) GenerateAndroidBuildActions(ctx android.ModuleContext) {
}

func (d *Defaults) DepsMutator(ctx android.BottomUpMutatorContext) {
}

func defaultsFactory() android.Module {
	return DefaultsFactory()
}

func DefaultsFactory(props ...interface{}) android.Module {
	module := &Defaults{}

	module.AddProperties(props...)
	module.AddProperties(
		&BaseProperties{},
		&VendorProperties{},
		&BaseCompilerProperties{},
		&BaseLinkerProperties{},
		&LibraryProperties{},
		&FlagExporterProperties{},
		&BinaryLinkerProperties{},
		&TestProperties{},
		&TestBinaryProperties{},
		&UnusedProperties{},
		&StlProperties{},
		&SanitizeProperties{},
		&StripProperties{},
		&InstallerProperties{},
		&TidyProperties{},
		&CoverageProperties{},
		&SAbiProperties{},
		&VndkProperties{},
		&LTOProperties{},
		&PgoProperties{},
		&android.ProtoProperties{},
	)

	android.InitDefaultsModule(module)

	return module
}

const (
	coreMode   = "core"
	vendorMode = "vendor"
)

func squashVendorSrcs(m *Module) {
	if lib, ok := m.compiler.(*libraryDecorator); ok {
		lib.baseCompiler.Properties.Srcs = append(lib.baseCompiler.Properties.Srcs, lib.baseCompiler.Properties.Target.Vendor.Srcs...)
		lib.baseCompiler.Properties.Exclude_srcs = append(lib.baseCompiler.Properties.Exclude_srcs, lib.baseCompiler.Properties.Target.Vendor.Exclude_srcs...)
	}
}

func vendorMutator(mctx android.BottomUpMutatorContext) {
	return
}

func getCurrentNdkPrebuiltVersion(ctx DepsContext) string {
	if ctx.Config().PlatformSdkVersionInt() > config.NdkMaxPrebuiltVersionInt {
		return strconv.Itoa(config.NdkMaxPrebuiltVersionInt)
	}
	return ctx.Config().PlatformSdkVersion()
}

var Bool = proptools.Bool
var BoolDefault = proptools.BoolDefault
var BoolPtr = proptools.BoolPtr
var String = proptools.String
var StringPtr = proptools.StringPtr
