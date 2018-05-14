package cc

import (
	"strings"
	"sync"

	"android/soong/android"
)

var (
	vendorPublicLibrarySuffix = ".vendorpublic"

	vendorPublicLibraries     = []string{}
	vendorPublicLibrariesLock sync.Mutex
)

type vendorPublicLibraryProperties struct {
	Symbol_file *string

	Unversioned *bool

	Export_public_headers []string `android:"arch_variant"`
}

type vendorPublicLibraryStubDecorator struct {
	*libraryDecorator

	Properties vendorPublicLibraryProperties

	versionScriptPath android.ModuleGenPath
}

func (stub *vendorPublicLibraryStubDecorator) Name(name string) string {
	return name + vendorPublicLibrarySuffix
}

func (stub *vendorPublicLibraryStubDecorator) compilerInit(ctx BaseModuleContext) {
	stub.baseCompiler.compilerInit(ctx)

	name := ctx.baseModuleName()
	if strings.HasSuffix(name, vendorPublicLibrarySuffix) {
		ctx.PropertyErrorf("name", "Do not append %q manually, just use the base name", vendorPublicLibrarySuffix)
	}

	vendorPublicLibrariesLock.Lock()
	defer vendorPublicLibrariesLock.Unlock()
	for _, lib := range vendorPublicLibraries {
		if lib == name {
			return
		}
	}
	vendorPublicLibraries = append(vendorPublicLibraries, name)
}

func (stub *vendorPublicLibraryStubDecorator) compilerFlags(ctx ModuleContext, flags Flags, deps PathDeps) Flags {
	flags = stub.baseCompiler.compilerFlags(ctx, flags, deps)
	return addStubLibraryCompilerFlags(flags)
}

func (stub *vendorPublicLibraryStubDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) Objects {
	objs, versionScript := compileStubLibrary(ctx, flags, String(stub.Properties.Symbol_file), "current", "")
	stub.versionScriptPath = versionScript
	return objs
}

func (stub *vendorPublicLibraryStubDecorator) linkerDeps(ctx DepsContext, deps Deps) Deps {
	headers := stub.Properties.Export_public_headers
	deps.HeaderLibs = append(deps.HeaderLibs, headers...)
	deps.ReexportHeaderLibHeaders = append(deps.ReexportHeaderLibHeaders, headers...)
	return deps
}

func (stub *vendorPublicLibraryStubDecorator) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	stub.libraryDecorator.libName = strings.TrimSuffix(ctx.ModuleName(), vendorPublicLibrarySuffix)
	return stub.libraryDecorator.linkerFlags(ctx, flags)
}

func (stub *vendorPublicLibraryStubDecorator) link(ctx ModuleContext, flags Flags, deps PathDeps,
	objs Objects) android.Path {
	if !Bool(stub.Properties.Unversioned) {
		linkerScriptFlag := "-Wl,--version-script," + stub.versionScriptPath.String()
		flags.LdFlags = append(flags.LdFlags, linkerScriptFlag)
	}
	return stub.libraryDecorator.link(ctx, flags, deps, objs)
}

func vendorPublicLibraryFactory() android.Module {
	module, library := NewLibrary(android.DeviceSupported)
	library.BuildOnlyShared()
	module.stl = nil
	module.sanitize = nil
	library.StripProperties.Strip.None = BoolPtr(true)

	stub := &vendorPublicLibraryStubDecorator{
		libraryDecorator: library,
	}
	module.compiler = stub
	module.linker = stub
	module.installer = nil

	module.AddProperties(
		&stub.Properties,
		&library.MutatedProperties,
		&library.flagExporter.Properties)

	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibBoth)
	return module
}

func init() {
	android.RegisterModuleType("vendor_public_library", vendorPublicLibraryFactory)
}
