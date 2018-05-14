package cc

import (
	"path/filepath"
	"strings"

	"android/soong/android"
)

var (
	llndkLibrarySuffix = ".llndk"
	llndkHeadersSuffix = ".llndk"
)

type llndkLibraryProperties struct {
	Symbol_file *string

	Export_headers_as_system *bool

	Export_preprocessed_headers []string

	Unversioned *bool

	Vendor_available *bool

	Export_llndk_headers []string `android:"arch_variant"`
}

type llndkStubDecorator struct {
	*libraryDecorator

	Properties llndkLibraryProperties

	exportHeadersTimestamp android.OptionalPath
	versionScriptPath      android.ModuleGenPath
}

func (stub *llndkStubDecorator) compilerFlags(ctx ModuleContext, flags Flags, deps PathDeps) Flags {
	flags = stub.baseCompiler.compilerFlags(ctx, flags, deps)
	return addStubLibraryCompilerFlags(flags)
}

func (stub *llndkStubDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) Objects {
	vndk_ver := ctx.DeviceConfig().VndkVersion()
	if vndk_ver == "current" {
		platform_vndk_ver := ctx.DeviceConfig().PlatformVndkVersion()
		if !inList(platform_vndk_ver, ctx.Config().PlatformVersionCombinedCodenames()) {
			vndk_ver = platform_vndk_ver
		}
	} else if vndk_ver == "" {

		vndk_ver = "current"
	}
	objs, versionScript := compileStubLibrary(ctx, flags, String(stub.Properties.Symbol_file), vndk_ver, "--vndk")
	stub.versionScriptPath = versionScript
	return objs
}

func (stub *llndkStubDecorator) linkerDeps(ctx DepsContext, deps Deps) Deps {
	headers := addSuffix(stub.Properties.Export_llndk_headers, llndkHeadersSuffix)
	deps.HeaderLibs = append(deps.HeaderLibs, headers...)
	deps.ReexportHeaderLibHeaders = append(deps.ReexportHeaderLibHeaders, headers...)
	return deps
}

func (stub *llndkStubDecorator) Name(name string) string {
	return name + llndkLibrarySuffix
}

func (stub *llndkStubDecorator) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	stub.libraryDecorator.libName = strings.TrimSuffix(ctx.ModuleName(),
		llndkLibrarySuffix)
	return stub.libraryDecorator.linkerFlags(ctx, flags)
}

func (stub *llndkStubDecorator) processHeaders(ctx ModuleContext, srcHeaderDir string, outDir android.ModuleGenPath) android.Path {
	srcDir := android.PathForModuleSrc(ctx, srcHeaderDir)
	srcFiles := ctx.GlobFiles(filepath.Join(srcDir.String(), "**/*.h"), nil)

	var installPaths []android.WritablePath
	for _, header := range srcFiles {
		headerDir := filepath.Dir(header.String())
		relHeaderDir, err := filepath.Rel(srcDir.String(), headerDir)
		if err != nil {
			ctx.ModuleErrorf("filepath.Rel(%q, %q) failed: %s",
				srcDir.String(), headerDir, err)
			continue
		}

		installPaths = append(installPaths, outDir.Join(ctx, relHeaderDir, header.Base()))
	}

	return processHeadersWithVersioner(ctx, srcDir, outDir, srcFiles, installPaths)
}

func (stub *llndkStubDecorator) link(ctx ModuleContext, flags Flags, deps PathDeps,
	objs Objects) android.Path {

	if !Bool(stub.Properties.Unversioned) {
		linkerScriptFlag := "-Wl,--version-script," + stub.versionScriptPath.String()
		flags.LdFlags = append(flags.LdFlags, linkerScriptFlag)
	}

	if len(stub.Properties.Export_preprocessed_headers) > 0 {
		genHeaderOutDir := android.PathForModuleGen(ctx, "include")

		var timestampFiles android.Paths
		for _, dir := range stub.Properties.Export_preprocessed_headers {
			timestampFiles = append(timestampFiles, stub.processHeaders(ctx, dir, genHeaderOutDir))
		}

		includePrefix := "-I "
		if Bool(stub.Properties.Export_headers_as_system) {
			includePrefix = "-isystem "
		}

		stub.reexportFlags([]string{includePrefix + " " + genHeaderOutDir.String()})
		stub.reexportDeps(timestampFiles)
	}

	if Bool(stub.Properties.Export_headers_as_system) {
		stub.exportIncludes(ctx, "-isystem")
		stub.libraryDecorator.flagExporter.Properties.Export_include_dirs = []string{}
	}

	return stub.libraryDecorator.link(ctx, flags, deps, objs)
}

func NewLLndkStubLibrary() *Module {
	module, library := NewLibrary(android.DeviceSupported)
	library.BuildOnlyShared()
	module.stl = nil
	module.sanitize = nil
	library.StripProperties.Strip.None = BoolPtr(true)

	stub := &llndkStubDecorator{
		libraryDecorator: library,
	}
	stub.Properties.Vendor_available = BoolPtr(true)
	module.compiler = stub
	module.linker = stub
	module.installer = nil

	module.AddProperties(
		&stub.Properties,
		&library.MutatedProperties,
		&library.flagExporter.Properties)

	return module
}

func llndkLibraryFactory() android.Module {
	module := NewLLndkStubLibrary()
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibBoth)
	return module
}

type llndkHeadersDecorator struct {
	*libraryDecorator
}

func (headers *llndkHeadersDecorator) Name(name string) string {
	return name + llndkHeadersSuffix
}

func llndkHeadersFactory() android.Module {
	module, library := NewLibrary(android.DeviceSupported)
	library.HeaderOnly()

	decorator := &llndkHeadersDecorator{
		libraryDecorator: library,
	}

	module.compiler = nil
	module.linker = decorator
	module.installer = nil

	module.AddProperties(&library.MutatedProperties, &library.flagExporter.Properties)

	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibBoth)

	return module
}

func init() {
	android.RegisterModuleType("llndk_library", llndkLibraryFactory)
	android.RegisterModuleType("llndk_headers", llndkHeadersFactory)
}
