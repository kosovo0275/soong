package cc

import (
	"android/soong/android"
)

func init() {
	android.RegisterModuleType("toolchain_library", toolchainLibraryFactory)
}

type toolchainLibraryDecorator struct {
	*libraryDecorator
}

func (*toolchainLibraryDecorator) linkerDeps(ctx DepsContext, deps Deps) Deps {

	return deps
}

func toolchainLibraryFactory() android.Module {
	module, library := NewLibrary(android.HostAndDeviceSupported)
	library.BuildOnlyStatic()
	toolchainLibrary := &toolchainLibraryDecorator{
		libraryDecorator: library,
	}
	module.compiler = toolchainLibrary
	module.linker = toolchainLibrary
	module.Properties.Clang = BoolPtr(false)
	module.stl = nil
	module.sanitize = nil
	module.installer = nil
	return module.Init()
}

func (library *toolchainLibraryDecorator) compile(ctx ModuleContext, flags Flags,
	deps PathDeps) Objects {
	return Objects{}
}

func (library *toolchainLibraryDecorator) link(ctx ModuleContext,
	flags Flags, deps PathDeps, objs Objects) android.Path {

	libName := ctx.ModuleName() + staticLibraryExtension
	outputFile := android.PathForModuleOut(ctx, libName)

	if flags.Clang {
		ctx.ModuleErrorf("toolchain_library must use GCC, not Clang")
	}

	CopyGccLib(ctx, libName, flagsToBuilderFlags(flags), outputFile)

	ctx.CheckbuildFile(outputFile)

	return outputFile
}
