package cc

import (
	"fmt"
	"strings"

	"android/soong/android"
	"android/soong/cc/config"
)

func init() {
	android.RegisterModuleType("ndk_prebuilt_object", ndkPrebuiltObjectFactory)
	android.RegisterModuleType("ndk_prebuilt_static_stl", ndkPrebuiltStaticStlFactory)
	android.RegisterModuleType("ndk_prebuilt_shared_stl", ndkPrebuiltSharedStlFactory)
}

func getNdkLibDir(ctx android.ModuleContext, toolchain config.Toolchain, version string) android.SourcePath {
	suffix := ""

	if toolchain.Is64Bit() && ctx.Arch().ArchType != android.Arm64 {
		suffix = "64"
	}
	return android.PathForSource(ctx, fmt.Sprintf("prebuilts/ndk/current/platforms/android-%s/arch-%s/usr/lib%s",
		version, toolchain.Name(), suffix))
}

func ndkPrebuiltModuleToPath(ctx android.ModuleContext, toolchain config.Toolchain,
	ext string, version string) android.Path {

	name := strings.Split(strings.TrimPrefix(ctx.ModuleName(), "ndk_"), ".")[0]
	dir := getNdkLibDir(ctx, toolchain, version)
	return dir.Join(ctx, name+ext)
}

type ndkPrebuiltObjectLinker struct {
	objectLinker
}

func (*ndkPrebuiltObjectLinker) linkerDeps(ctx DepsContext, deps Deps) Deps {

	return deps
}

func ndkPrebuiltObjectFactory() android.Module {
	module := newBaseModule(android.DeviceSupported, android.MultilibBoth)
	module.linker = &ndkPrebuiltObjectLinker{
		objectLinker: objectLinker{
			baseLinker: NewBaseLinker(),
		},
	}
	module.Properties.HideFromMake = true
	return module.Init()
}

func (c *ndkPrebuiltObjectLinker) link(ctx ModuleContext, flags Flags,
	deps PathDeps, objs Objects) android.Path {

	if !strings.HasPrefix(ctx.ModuleName(), "ndk_crt") {
		ctx.ModuleErrorf("NDK prebuilt objects must have an ndk_crt prefixed name")
	}

	return ndkPrebuiltModuleToPath(ctx, flags.Toolchain, objectExtension, ctx.sdkVersion())
}

type ndkPrebuiltStlLinker struct {
	*libraryDecorator
}

func (ndk *ndkPrebuiltStlLinker) linkerProps() []interface{} {
	return append(ndk.libraryDecorator.linkerProps(), &ndk.Properties, &ndk.flagExporter.Properties)
}

func (*ndkPrebuiltStlLinker) linkerDeps(ctx DepsContext, deps Deps) Deps {

	return deps
}

func ndkPrebuiltSharedStlFactory() android.Module {
	module, library := NewLibrary(android.DeviceSupported)
	library.BuildOnlyShared()
	module.compiler = nil
	module.linker = &ndkPrebuiltStlLinker{
		libraryDecorator: library,
	}
	module.installer = nil
	minVersionString := "minimum"
	noStlString := "none"
	module.Properties.Sdk_version = &minVersionString
	module.stl.Properties.Stl = &noStlString
	return module.Init()
}

func ndkPrebuiltStaticStlFactory() android.Module {
	module, library := NewLibrary(android.DeviceSupported)
	library.BuildOnlyStatic()
	module.compiler = nil
	module.linker = &ndkPrebuiltStlLinker{
		libraryDecorator: library,
	}
	module.installer = nil
	module.Properties.HideFromMake = true
	return module.Init()
}

func getNdkStlLibDir(ctx android.ModuleContext) android.SourcePath {
	libDir := "prebuilts/ndk/current/sources/cxx-stl/llvm-libc++/libs"
	return android.PathForSource(ctx, libDir).Join(ctx, ctx.Arch().Abi[0])
}

func (ndk *ndkPrebuiltStlLinker) link(ctx ModuleContext, flags Flags,
	deps PathDeps, objs Objects) android.Path {
	if !strings.HasPrefix(ctx.ModuleName(), "ndk_lib") {
		ctx.ModuleErrorf("NDK prebuilt libraries must have an ndk_lib prefixed name")
	}

	ndk.exportIncludes(ctx, "-isystem")

	libName := strings.TrimPrefix(ctx.ModuleName(), "ndk_")
	libExt := flags.Toolchain.ShlibSuffix()
	if ndk.static() {
		libExt = staticLibraryExtension
	}

	libDir := getNdkStlLibDir(ctx)
	return libDir.Join(ctx, libName+libExt)
}
