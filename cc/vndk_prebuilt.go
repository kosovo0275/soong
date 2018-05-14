package cc

import (
	"strings"

	"android/soong/android"
)

var (
	vndkSuffix = ".vndk."
)

type vndkPrebuiltProperties struct {
	Version *string

	Target_arch *string

	Srcs []string `android:"arch_variant"`
}

type vndkPrebuiltLibraryDecorator struct {
	*libraryDecorator
	properties vndkPrebuiltProperties
}

func (p *vndkPrebuiltLibraryDecorator) Name(name string) string {
	return name + p.NameSuffix()
}

func (p *vndkPrebuiltLibraryDecorator) NameSuffix() string {
	if p.arch() != "" {
		return vndkSuffix + p.version() + "." + p.arch()
	}
	return vndkSuffix + p.version()
}

func (p *vndkPrebuiltLibraryDecorator) version() string {
	return String(p.properties.Version)
}

func (p *vndkPrebuiltLibraryDecorator) arch() string {
	return String(p.properties.Target_arch)
}

func (p *vndkPrebuiltLibraryDecorator) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	p.libraryDecorator.libName = strings.TrimSuffix(ctx.ModuleName(), p.NameSuffix())
	return p.libraryDecorator.linkerFlags(ctx, flags)
}

func (p *vndkPrebuiltLibraryDecorator) singleSourcePath(ctx ModuleContext) android.Path {
	if len(p.properties.Srcs) == 0 {
		ctx.PropertyErrorf("srcs", "missing prebuilt source file")
		return nil
	}

	if len(p.properties.Srcs) > 1 {
		ctx.PropertyErrorf("srcs", "multiple prebuilt source files")
		return nil
	}

	return android.PathForModuleSrc(ctx, p.properties.Srcs[0])
}

func (p *vndkPrebuiltLibraryDecorator) link(ctx ModuleContext,
	flags Flags, deps PathDeps, objs Objects) android.Path {
	if len(p.properties.Srcs) > 0 && p.shared() {

		return p.singleSourcePath(ctx)
	}
	return nil
}

func (p *vndkPrebuiltLibraryDecorator) install(ctx ModuleContext, file android.Path) {
	arches := ctx.DeviceConfig().Arches()
	if len(arches) == 0 || arches[0].ArchType.String() != p.arch() {
		return
	}
	if p.shared() {
		if ctx.isVndkSp() {
			p.baseInstaller.subDir = "vndk-sp-" + p.version()
		} else if ctx.isVndk() {
			p.baseInstaller.subDir = "vndk-" + p.version()
		}
		p.baseInstaller.install(ctx, file)
	}
}

func vndkPrebuiltSharedLibrary() *Module {
	module, library := NewLibrary(android.DeviceSupported)
	library.BuildOnlyShared()
	module.stl = nil
	module.sanitize = nil
	library.StripProperties.Strip.None = BoolPtr(true)

	prebuilt := &vndkPrebuiltLibraryDecorator{
		libraryDecorator: library,
	}

	module.compiler = nil
	module.linker = prebuilt
	module.installer = prebuilt

	module.AddProperties(
		&prebuilt.properties,
	)

	return module
}

func vndkPrebuiltSharedFactory() android.Module {
	module := vndkPrebuiltSharedLibrary()
	return module.Init()
}

func init() {
	android.RegisterModuleType("vndk_prebuilt_shared", vndkPrebuiltSharedFactory)
}
