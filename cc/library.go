package cc

import (
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/pathtools"

	"android/soong/android"
)

type LibraryProperties struct {
	Static struct {
		Srcs   []string `android:"arch_variant"`
		Cflags []string `android:"arch_variant"`

		Enabled           *bool    `android:"arch_variant"`
		Whole_static_libs []string `android:"arch_variant"`
		Static_libs       []string `android:"arch_variant"`
		Shared_libs       []string `android:"arch_variant"`
	} `android:"arch_variant"`
	Shared struct {
		Srcs   []string `android:"arch_variant"`
		Cflags []string `android:"arch_variant"`

		Enabled           *bool    `android:"arch_variant"`
		Whole_static_libs []string `android:"arch_variant"`
		Static_libs       []string `android:"arch_variant"`
		Shared_libs       []string `android:"arch_variant"`
	} `android:"arch_variant"`

	Version_script              *string `android:"arch_variant"`
	Unexported_symbols_list     *string `android:"arch_variant"`
	Force_symbols_not_weak_list *string `android:"arch_variant"`
	Force_symbols_weak_list     *string `android:"arch_variant"`
	Unique_host_soname          *bool

	Aidl struct {
		Export_aidl_headers *bool
	}

	Proto struct {
		Export_proto_headers *bool
	}
	Target struct {
		Vendor struct {
			Version_script *string `android:"arch_variant"`
		}
	}

	Static_ndk_lib *bool
}

type LibraryMutatedProperties struct {
	VariantName     string `blueprint:"mutated"`
	BuildStatic     bool   `blueprint:"mutated"`
	BuildShared     bool   `blueprint:"mutated"`
	VariantIsShared bool   `blueprint:"mutated"`
	VariantIsStatic bool   `blueprint:"mutated"`
}

type FlagExporterProperties struct {
	Export_include_dirs []string `android:"arch_variant"`

	Target struct {
		Vendor struct {
			Override_export_include_dirs []string
		}
	}
}

func init() {
	android.RegisterModuleType("cc_library_static", LibraryStaticFactory)
	android.RegisterModuleType("cc_library_shared", LibrarySharedFactory)
	android.RegisterModuleType("cc_library", LibraryFactory)
	android.RegisterModuleType("cc_library_host_static", LibraryHostStaticFactory)
	android.RegisterModuleType("cc_library_host_shared", LibraryHostSharedFactory)
	android.RegisterModuleType("cc_library_headers", LibraryHeaderFactory)
}

func LibraryFactory() android.Module {
	module, _ := NewLibrary(android.HostAndDeviceSupported)
	return module.Init()
}

func LibraryStaticFactory() android.Module {
	module, library := NewLibrary(android.HostAndDeviceSupported)
	library.BuildOnlyStatic()
	return module.Init()
}

func LibrarySharedFactory() android.Module {
	module, library := NewLibrary(android.HostAndDeviceSupported)
	library.BuildOnlyShared()
	return module.Init()
}

func LibraryHostStaticFactory() android.Module {
	module, library := NewLibrary(android.HostSupported)
	library.BuildOnlyStatic()
	return module.Init()
}

// Module factory for host shared libraries
func LibraryHostSharedFactory() android.Module {
	module, library := NewLibrary(android.HostSupported)
	library.BuildOnlyShared()
	return module.Init()
}

// Module factory for header-only libraries
func LibraryHeaderFactory() android.Module {
	module, library := NewLibrary(android.HostAndDeviceSupported)
	library.HeaderOnly()
	return module.Init()
}

type flagExporter struct {
	Properties FlagExporterProperties
	flags      []string
	flagsDeps  android.Paths
}

func (f *flagExporter) exportedIncludes(ctx ModuleContext) android.Paths {
	if ctx.useVndk() && f.Properties.Target.Vendor.Override_export_include_dirs != nil {
		return android.PathsForModuleSrc(ctx, f.Properties.Target.Vendor.Override_export_include_dirs)
	} else {
		return android.PathsForModuleSrc(ctx, f.Properties.Export_include_dirs)
	}
}

func (f *flagExporter) exportIncludes(ctx ModuleContext, inc string) {
	includeDirs := f.exportedIncludes(ctx)
	for _, dir := range includeDirs.Strings() {
		f.flags = append(f.flags, inc+dir)
	}
}

func (f *flagExporter) reexportFlags(flags []string) {
	f.flags = append(f.flags, flags...)
}

func (f *flagExporter) reexportDeps(deps android.Paths) {
	f.flagsDeps = append(f.flagsDeps, deps...)
}

func (f *flagExporter) exportedFlags() []string {
	return f.flags
}

func (f *flagExporter) exportedFlagsDeps() android.Paths {
	return f.flagsDeps
}

type exportedFlagsProducer interface {
	exportedFlags() []string
	exportedFlagsDeps() android.Paths
}

var _ exportedFlagsProducer = (*flagExporter)(nil)

type libraryDecorator struct {
	Properties         LibraryProperties
	MutatedProperties  LibraryMutatedProperties
	reuseObjects       Objects
	reuseExportedFlags []string
	reuseExportedDeps  android.Paths
	tocFile            android.OptionalPath
	flagExporter
	stripper
	relocationPacker
	wholeStaticMissingDeps []string
	objects                Objects
	libName                string
	sanitize               *sanitize
	sabi                   *sabi
	coverageOutputFile     android.OptionalPath
	sAbiOutputFile         android.OptionalPath
	sAbiDiff               android.OptionalPath
	ndkSysrootPath         android.Path
	*baseCompiler
	*baseLinker
	*baseInstaller
}

func (library *libraryDecorator) linkerProps() []interface{} {
	var props []interface{}
	props = append(props, library.baseLinker.linkerProps()...)
	return append(props,
		&library.Properties,
		&library.MutatedProperties,
		&library.flagExporter.Properties,
		&library.stripper.StripProperties,
		&library.relocationPacker.Properties)
}

func (library *libraryDecorator) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	flags = library.baseLinker.linkerFlags(ctx, flags)
	flags.CFlags = append(flags.CFlags, "-fPIC")

	if library.static() {
		flags.CFlags = append(flags.CFlags, library.Properties.Static.Cflags...)
	} else if library.shared() {
		libName := library.getLibName(ctx)
		flags.CFlags = append(flags.CFlags, library.Properties.Shared.Cflags...)
		f := []string{"-Wl,--gc-sections", "-shared", "-Wl,-soname," + libName + flags.Toolchain.ShlibSuffix()}
		flags.LdFlags = append(f, flags.LdFlags...)
	}

	return flags
}

func (library *libraryDecorator) compilerFlags(ctx ModuleContext, flags Flags, deps PathDeps) Flags {
	exportIncludeDirs := library.flagExporter.exportedIncludes(ctx)
	if len(exportIncludeDirs) > 0 {
		f := includeDirsToFlags(exportIncludeDirs)
		flags.GlobalFlags = append(flags.GlobalFlags, f)
		flags.YasmFlags = append(flags.YasmFlags, f)
	}

	return library.baseCompiler.compilerFlags(ctx, flags, deps)
}

func extractExportIncludesFromFlags(flags []string) []string {
	var exportedIncludes []string
	for _, flag := range flags {
		if strings.HasPrefix(flag, "-I") {
			exportedIncludes = append(exportedIncludes, flag)
		}
	}
	return exportedIncludes
}

func (library *libraryDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) Objects {
	if !library.buildShared() && !library.buildStatic() {
		if len(library.baseCompiler.Properties.Srcs) > 0 {
			ctx.PropertyErrorf("srcs", "cc_library_headers must not have any srcs")
		}
		if len(library.Properties.Static.Srcs) > 0 {
			ctx.PropertyErrorf("static.srcs", "cc_library_headers must not have any srcs")
		}
		if len(library.Properties.Shared.Srcs) > 0 {
			ctx.PropertyErrorf("shared.srcs", "cc_library_headers must not have any srcs")
		}
		return Objects{}
	}
	if ctx.createVndkSourceAbiDump() || library.sabi.Properties.CreateSAbiDumps {
		exportIncludeDirs := library.flagExporter.exportedIncludes(ctx)
		var SourceAbiFlags []string
		for _, dir := range exportIncludeDirs.Strings() {
			SourceAbiFlags = append(SourceAbiFlags, "-I"+dir)
		}
		for _, reexportedInclude := range extractExportIncludesFromFlags(library.sabi.Properties.ReexportedIncludeFlags) {
			SourceAbiFlags = append(SourceAbiFlags, reexportedInclude)
		}
		flags.SAbiFlags = SourceAbiFlags
		total_length := len(library.baseCompiler.Properties.Srcs) + len(deps.GeneratedSources) + len(library.Properties.Shared.Srcs) +
			len(library.Properties.Static.Srcs)
		if total_length > 0 {
			flags.SAbiDump = true
		}
	}
	objs := library.baseCompiler.compile(ctx, flags, deps)
	library.reuseObjects = objs
	buildFlags := flagsToBuilderFlags(flags)

	if library.static() {
		srcs := android.PathsForModuleSrc(ctx, library.Properties.Static.Srcs)
		objs = objs.Append(compileObjs(ctx, buildFlags, android.DeviceStaticLibrary, srcs, library.baseCompiler.pathDeps, library.baseCompiler.cFlagsDeps))
	} else if library.shared() {
		srcs := android.PathsForModuleSrc(ctx, library.Properties.Shared.Srcs)
		objs = objs.Append(compileObjs(ctx, buildFlags, android.DeviceSharedLibrary, srcs, library.baseCompiler.pathDeps, library.baseCompiler.cFlagsDeps))
	}

	return objs
}

type libraryInterface interface {
	getWholeStaticMissingDeps() []string
	static() bool
	objs() Objects
	reuseObjs() (Objects, []string, android.Paths)
	toc() android.OptionalPath
	buildStatic() bool
	buildShared() bool
	setStatic()
	setShared()
}

func (library *libraryDecorator) getLibName(ctx ModuleContext) string {
	name := library.libName
	if name == "" {
		name = ctx.baseModuleName()
	}

	if ctx.isVndkExt() {
		name = ctx.getVndkExtendsModuleName()
	}

	if ctx.Host() && Bool(library.Properties.Unique_host_soname) {
		if !strings.HasSuffix(name, "-host") {
			name = name + "-host"
		}
	}

	return name + library.MutatedProperties.VariantName
}

func (library *libraryDecorator) linkerInit(ctx BaseModuleContext) {
	location := InstallInSystem
	if library.sanitize.inSanitizerDir() {
		location = InstallInSanitizerDir
	}
	library.baseInstaller.location = location
	library.baseLinker.linkerInit(ctx)
	library.relocationPacker.packingInit(ctx)
}

func (library *libraryDecorator) linkerDeps(ctx DepsContext, deps Deps) Deps {
	deps = library.baseLinker.linkerDeps(ctx, deps)

	if library.static() {
		deps.WholeStaticLibs = append(deps.WholeStaticLibs, library.Properties.Static.Whole_static_libs...)
		deps.StaticLibs = append(deps.StaticLibs, library.Properties.Static.Static_libs...)
		deps.SharedLibs = append(deps.SharedLibs, library.Properties.Static.Shared_libs...)
	} else if library.shared() {
		deps.WholeStaticLibs = append(deps.WholeStaticLibs, library.Properties.Shared.Whole_static_libs...)
		deps.StaticLibs = append(deps.StaticLibs, library.Properties.Shared.Static_libs...)
		deps.SharedLibs = append(deps.SharedLibs, library.Properties.Shared.Shared_libs...)
	}
	android.ExtractSourceDeps(ctx, library.Properties.Version_script)
	android.ExtractSourceDeps(ctx, library.Properties.Unexported_symbols_list)
	android.ExtractSourceDeps(ctx, library.Properties.Force_symbols_not_weak_list)
	android.ExtractSourceDeps(ctx, library.Properties.Force_symbols_weak_list)
	android.ExtractSourceDeps(ctx, library.Properties.Target.Vendor.Version_script)

	return deps
}

func (library *libraryDecorator) linkStatic(ctx ModuleContext, flags Flags, deps PathDeps, objs Objects) android.Path {
	library.objects = deps.WholeStaticLibObjs.Copy()
	library.objects = library.objects.Append(objs)
	fileName := ctx.ModuleName() + library.MutatedProperties.VariantName + staticLibraryExtension
	outputFile := android.PathForModuleOut(ctx, fileName)
	builderFlags := flagsToBuilderFlags(flags)

	if Bool(library.baseLinker.Properties.Use_version_lib) && ctx.Host() {
		versionedOutputFile := outputFile
		outputFile = android.PathForModuleOut(ctx, "unversioned", fileName)
		library.injectVersionSymbol(ctx, outputFile, versionedOutputFile)
	}

	TransformObjToStaticLib(ctx, library.objects.objFiles, builderFlags, outputFile, objs.tidyFiles)
	library.coverageOutputFile = TransformCoverageFilesToLib(ctx, library.objects, builderFlags, ctx.ModuleName()+library.MutatedProperties.VariantName)
	library.wholeStaticMissingDeps = ctx.GetMissingDependencies()
	ctx.CheckbuildFile(outputFile)

	return outputFile
}

func (library *libraryDecorator) linkShared(ctx ModuleContext, flags Flags, deps PathDeps, objs Objects) android.Path {
	var linkerDeps android.Paths
	linkerDeps = append(linkerDeps, flags.LdFlagsDeps...)

	versionScript := ctx.ExpandOptionalSource(library.Properties.Version_script, "version_script")
	unexportedSymbols := ctx.ExpandOptionalSource(library.Properties.Unexported_symbols_list, "unexported_symbols_list")
	forceNotWeakSymbols := ctx.ExpandOptionalSource(library.Properties.Force_symbols_not_weak_list, "force_symbols_not_weak_list")
	forceWeakSymbols := ctx.ExpandOptionalSource(library.Properties.Force_symbols_weak_list, "force_symbols_weak_list")
	if ctx.useVndk() && library.Properties.Target.Vendor.Version_script != nil {
		versionScript = ctx.ExpandOptionalSource(library.Properties.Target.Vendor.Version_script, "target.vendor.version_script")
	}
	if versionScript.Valid() {
		flags.LdFlags = append(flags.LdFlags, "-Wl,--version-script,"+versionScript.String())
		linkerDeps = append(linkerDeps, versionScript.Path())
		if library.sanitize.isSanitizerEnabled(cfi) {
			cfiExportsMap := android.PathForSource(ctx, cfiExportsMapPath)
			flags.LdFlags = append(flags.LdFlags, "-Wl,--version-script,"+cfiExportsMap.String())
			linkerDeps = append(linkerDeps, cfiExportsMap)
		}
	}
	if unexportedSymbols.Valid() {
		ctx.PropertyErrorf("unexported_symbols_list", "Only supported on Darwin")
	}
	if forceNotWeakSymbols.Valid() {
		ctx.PropertyErrorf("force_symbols_not_weak_list", "Only supported on Darwin")
	}
	if forceWeakSymbols.Valid() {
		ctx.PropertyErrorf("force_symbols_weak_list", "Only supported on Darwin")
	}

	fileName := library.getLibName(ctx) + flags.Toolchain.ShlibSuffix()
	outputFile := android.PathForModuleOut(ctx, fileName)
	ret := outputFile

	builderFlags := flagsToBuilderFlags(flags)

	tocPath := outputFile.RelPathString()
	tocPath = pathtools.ReplaceExtension(tocPath, flags.Toolchain.ShlibSuffix()[1:]+".toc")
	tocFile := android.PathForOutput(ctx, tocPath)
	library.tocFile = android.OptionalPathForPath(tocFile)
	TransformSharedObjectToToc(ctx, outputFile, tocFile, builderFlags)

	if library.relocationPacker.needsPacking(ctx) {
		packedOutputFile := outputFile
		outputFile = android.PathForModuleOut(ctx, "unpacked", fileName)
		library.relocationPacker.pack(ctx, outputFile, packedOutputFile, builderFlags)
	}

	if library.stripper.needsStrip(ctx) {
		strippedOutputFile := outputFile
		outputFile = android.PathForModuleOut(ctx, "unstripped", fileName)
		library.stripper.strip(ctx, outputFile, strippedOutputFile, builderFlags)
	}

	if Bool(library.baseLinker.Properties.Use_version_lib) && ctx.Host() {
		versionedOutputFile := outputFile
		outputFile = android.PathForModuleOut(ctx, "unversioned", fileName)
		library.injectVersionSymbol(ctx, outputFile, versionedOutputFile)
	}

	sharedLibs := deps.SharedLibs
	sharedLibs = append(sharedLibs, deps.LateSharedLibs...)

	linkerDeps = append(linkerDeps, deps.SharedLibsDeps...)
	linkerDeps = append(linkerDeps, deps.LateSharedLibsDeps...)
	linkerDeps = append(linkerDeps, objs.tidyFiles...)

	TransformObjToDynamicBinary(ctx, objs.objFiles, sharedLibs, deps.StaticLibs, deps.LateStaticLibs, deps.WholeStaticLibs, linkerDeps, false, builderFlags, outputFile)

	objs.coverageFiles = append(objs.coverageFiles, deps.StaticLibObjs.coverageFiles...)
	objs.coverageFiles = append(objs.coverageFiles, deps.WholeStaticLibObjs.coverageFiles...)

	objs.sAbiDumpFiles = append(objs.sAbiDumpFiles, deps.StaticLibObjs.sAbiDumpFiles...)
	objs.sAbiDumpFiles = append(objs.sAbiDumpFiles, deps.WholeStaticLibObjs.sAbiDumpFiles...)

	library.coverageOutputFile = TransformCoverageFilesToLib(ctx, objs, builderFlags, library.getLibName(ctx))
	library.linkSAbiDumpFiles(ctx, objs, fileName, ret)

	return ret
}

func (library *libraryDecorator) linkSAbiDumpFiles(ctx ModuleContext, objs Objects, fileName string, soFile android.Path) {
	//Also take into account object re-use.
	if len(objs.sAbiDumpFiles) > 0 && ctx.createVndkSourceAbiDump() {
		vndkVersion := ctx.DeviceConfig().PlatformVndkVersion()
		if ver := ctx.DeviceConfig().VndkVersion(); ver != "" && ver != "current" {
			vndkVersion = ver
		}

		refSourceDumpFile := android.PathForVndkRefAbiDump(ctx, vndkVersion, fileName, vndkVsNdk(ctx), true)
		exportIncludeDirs := library.flagExporter.exportedIncludes(ctx)
		var SourceAbiFlags []string
		for _, dir := range exportIncludeDirs.Strings() {
			SourceAbiFlags = append(SourceAbiFlags, "-I"+dir)
		}
		for _, reexportedInclude := range extractExportIncludesFromFlags(library.sabi.Properties.ReexportedIncludeFlags) {
			SourceAbiFlags = append(SourceAbiFlags, reexportedInclude)
		}
		exportedHeaderFlags := strings.Join(SourceAbiFlags, " ")
		library.sAbiOutputFile = TransformDumpToLinkedDump(ctx, objs.sAbiDumpFiles, soFile, fileName, exportedHeaderFlags)
		if refSourceDumpFile.Valid() {
			unzippedRefDump := UnzipRefDump(ctx, refSourceDumpFile.Path(), fileName)
			library.sAbiDiff = SourceAbiDiff(ctx, library.sAbiOutputFile.Path(), unzippedRefDump, fileName, exportedHeaderFlags, ctx.isVndkExt())
		}
	}
}

func vndkVsNdk(ctx ModuleContext) bool {
	if inList(ctx.baseModuleName(), llndkLibraries) {
		return false
	}
	return true
}

func (library *libraryDecorator) link(ctx ModuleContext, flags Flags, deps PathDeps, objs Objects) android.Path {
	objs = deps.Objs.Copy().Append(objs)
	var out android.Path
	if library.static() || library.header() {
		out = library.linkStatic(ctx, flags, deps, objs)
	} else {
		out = library.linkShared(ctx, flags, deps, objs)
	}

	library.exportIncludes(ctx, "-I")
	library.reexportFlags(deps.ReexportedFlags)
	library.reexportDeps(deps.ReexportedFlagsDeps)

	if Bool(library.Properties.Aidl.Export_aidl_headers) {
		if library.baseCompiler.hasSrcExt(".aidl") {
			flags := []string{
				"-I" + android.PathForModuleGen(ctx, "aidl").String(),
			}
			library.reexportFlags(flags)
			library.reuseExportedFlags = append(library.reuseExportedFlags, flags...)
			library.reexportDeps(library.baseCompiler.pathDeps) // TODO: restrict to aidl deps
			library.reuseExportedDeps = append(library.reuseExportedDeps, library.baseCompiler.pathDeps...)
		}
	}

	if Bool(library.Properties.Proto.Export_proto_headers) {
		if library.baseCompiler.hasSrcExt(".proto") {
			includes := []string{}
			if flags.ProtoRoot {
				includes = append(includes, "-I"+android.ProtoSubDir(ctx).String())
			}
			includes = append(includes, "-I"+android.ProtoDir(ctx).String())
			library.reexportFlags(includes)
			library.reuseExportedFlags = append(library.reuseExportedFlags, includes...)
			library.reexportDeps(library.baseCompiler.pathDeps) // TODO: restrict to proto deps
			library.reuseExportedDeps = append(library.reuseExportedDeps, library.baseCompiler.pathDeps...)
		}
	}

	return out
}

func (library *libraryDecorator) buildStatic() bool {
	return library.MutatedProperties.BuildStatic && BoolDefault(library.Properties.Static.Enabled, true)
}

func (library *libraryDecorator) buildShared() bool {
	return library.MutatedProperties.BuildShared && BoolDefault(library.Properties.Shared.Enabled, true)
}

func (library *libraryDecorator) getWholeStaticMissingDeps() []string {
	return append([]string(nil), library.wholeStaticMissingDeps...)
}

func (library *libraryDecorator) objs() Objects {
	return library.objects
}

func (library *libraryDecorator) reuseObjs() (Objects, []string, android.Paths) {
	return library.reuseObjects, library.reuseExportedFlags, library.reuseExportedDeps
}

func (library *libraryDecorator) toc() android.OptionalPath {
	return library.tocFile
}

func (library *libraryDecorator) install(ctx ModuleContext, file android.Path) {
	if library.shared() {
		if ctx.Device() && ctx.useVndk() {
			if ctx.isVndkSp() {
				library.baseInstaller.subDir = "vndk-sp"
			} else if ctx.isVndk() {
				library.baseInstaller.subDir = "vndk"
			}

			// Append a version to vndk or vndk-sp directories on the system partition.
			if ctx.isVndk() && !ctx.isVndkExt() {
				vndkVersion := ctx.DeviceConfig().PlatformVndkVersion()
				if vndkVersion != "current" && vndkVersion != "" {
					library.baseInstaller.subDir += "-" + vndkVersion
				}
			}
		}
		library.baseInstaller.install(ctx, file)
	}
}

func (library *libraryDecorator) static() bool {
	return library.MutatedProperties.VariantIsStatic
}

func (library *libraryDecorator) shared() bool {
	return library.MutatedProperties.VariantIsShared
}

func (library *libraryDecorator) header() bool {
	return !library.static() && !library.shared()
}

func (library *libraryDecorator) setStatic() {
	library.MutatedProperties.VariantIsStatic = true
	library.MutatedProperties.VariantIsShared = false
}

func (library *libraryDecorator) setShared() {
	library.MutatedProperties.VariantIsStatic = false
	library.MutatedProperties.VariantIsShared = true
}

func (library *libraryDecorator) BuildOnlyStatic() {
	library.MutatedProperties.BuildShared = false
}

func (library *libraryDecorator) BuildOnlyShared() {
	library.MutatedProperties.BuildStatic = false
}

func (library *libraryDecorator) HeaderOnly() {
	library.MutatedProperties.BuildShared = false
	library.MutatedProperties.BuildStatic = false
}

func NewLibrary(hod android.HostOrDeviceSupported) (*Module, *libraryDecorator) {
	module := newModule(hod, android.MultilibBoth)

	library := &libraryDecorator{
		MutatedProperties: LibraryMutatedProperties{
			BuildShared: true,
			BuildStatic: true,
		},
		baseCompiler:  NewBaseCompiler(),
		baseLinker:    NewBaseLinker(),
		baseInstaller: NewBaseInstaller("lib", "lib64", InstallInSystem),
		sanitize:      module.sanitize,
		sabi:          module.sabi,
	}

	module.compiler = library
	module.linker = library
	module.installer = library

	return module, library
}

// connects a shared library to a static library in order to reuse its .o files to avoid
// compiling source files twice.
func reuseStaticLibrary(mctx android.BottomUpMutatorContext, static, shared *Module) {
	if staticCompiler, ok := static.compiler.(*libraryDecorator); ok {
		sharedCompiler := shared.compiler.(*libraryDecorator)
		if len(staticCompiler.Properties.Static.Cflags) == 0 &&
			len(sharedCompiler.Properties.Shared.Cflags) == 0 {

			mctx.AddInterVariantDependency(reuseObjTag, shared, static)
			sharedCompiler.baseCompiler.Properties.OriginalSrcs = sharedCompiler.baseCompiler.Properties.Srcs
			sharedCompiler.baseCompiler.Properties.Srcs = nil
			sharedCompiler.baseCompiler.Properties.Generated_sources = nil
		}
	}
}

func linkageMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok && m.linker != nil {
		if library, ok := m.linker.(libraryInterface); ok {
			var modules []blueprint.Module
			if library.buildStatic() && library.buildShared() {
				modules = mctx.CreateLocalVariations("static", "shared")
				static := modules[0].(*Module)
				shared := modules[1].(*Module)

				static.linker.(libraryInterface).setStatic()
				shared.linker.(libraryInterface).setShared()

				reuseStaticLibrary(mctx, static, shared)

			} else if library.buildStatic() {
				modules = mctx.CreateLocalVariations("static")
				modules[0].(*Module).linker.(libraryInterface).setStatic()
			} else if library.buildShared() {
				modules = mctx.CreateLocalVariations("shared")
				modules[0].(*Module).linker.(libraryInterface).setShared()
			}
		}
	}
}
