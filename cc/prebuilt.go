package cc

import (
	"android/soong/android"
)

func init() {
	android.RegisterModuleType("cc_prebuilt_library_shared", prebuiltSharedLibraryFactory)
	android.RegisterModuleType("cc_prebuilt_library_static", prebuiltStaticLibraryFactory)
	android.RegisterModuleType("cc_prebuilt_binary", prebuiltBinaryFactory)
}

type prebuiltLinkerInterface interface {
	Name(string) string
	prebuilt() *android.Prebuilt
}

type prebuiltLinker struct {
	android.Prebuilt
	properties struct {
		Srcs []string `android:"arch_variant"`
	}
}

func (p *prebuiltLinker) prebuilt() *android.Prebuilt {
	return &p.Prebuilt
}

func (p *prebuiltLinker) PrebuiltSrcs() []string {
	return p.properties.Srcs
}

type prebuiltLibraryLinker struct {
	*libraryDecorator
	prebuiltLinker
}

var _ prebuiltLinkerInterface = (*prebuiltLibraryLinker)(nil)

func (p *prebuiltLibraryLinker) link(ctx ModuleContext,
	flags Flags, deps PathDeps, objs Objects) android.Path {

	if len(p.properties.Srcs) > 0 {
		p.libraryDecorator.exportIncludes(ctx, "-I")
		p.libraryDecorator.reexportFlags(deps.ReexportedFlags)
		p.libraryDecorator.reexportDeps(deps.ReexportedFlagsDeps)

		return p.Prebuilt.SingleSourcePath(ctx)
	}

	return nil
}

func prebuiltSharedLibraryFactory() android.Module {
	module, _ := NewPrebuiltSharedLibrary(android.HostAndDeviceSupported)
	return module.Init()
}

func NewPrebuiltSharedLibrary(hod android.HostOrDeviceSupported) (*Module, *libraryDecorator) {
	module, library := NewLibrary(hod)
	library.BuildOnlyShared()
	module.compiler = nil

	prebuilt := &prebuiltLibraryLinker{
		libraryDecorator: library,
	}
	module.linker = prebuilt

	module.AddProperties(&prebuilt.properties)

	android.InitPrebuiltModule(module, &prebuilt.properties.Srcs)
	return module, library
}

func prebuiltStaticLibraryFactory() android.Module {
	module, _ := NewPrebuiltStaticLibrary(android.HostAndDeviceSupported)
	return module.Init()
}

func NewPrebuiltStaticLibrary(hod android.HostOrDeviceSupported) (*Module, *libraryDecorator) {
	module, library := NewLibrary(hod)
	library.BuildOnlyStatic()
	module.compiler = nil

	prebuilt := &prebuiltLibraryLinker{
		libraryDecorator: library,
	}
	module.linker = prebuilt

	module.AddProperties(&prebuilt.properties)

	android.InitPrebuiltModule(module, &prebuilt.properties.Srcs)
	return module, library
}

type prebuiltBinaryLinker struct {
	*binaryDecorator
	prebuiltLinker
}

var _ prebuiltLinkerInterface = (*prebuiltBinaryLinker)(nil)

func (p *prebuiltBinaryLinker) link(ctx ModuleContext,
	flags Flags, deps PathDeps, objs Objects) android.Path {

	if len(p.properties.Srcs) > 0 {

		fileName := p.getStem(ctx) + flags.Toolchain.ExecutableSuffix()
		outputFile := android.PathForModuleOut(ctx, fileName)

		ctx.Build(pctx, android.BuildParams{
			Rule:        android.CpExecutable,
			Description: "prebuilt",
			Output:      outputFile,
			Input:       p.Prebuilt.SingleSourcePath(ctx),
		})

		return outputFile
	}

	return nil
}

func prebuiltBinaryFactory() android.Module {
	module, _ := NewPrebuiltBinary(android.HostAndDeviceSupported)
	return module.Init()
}

func NewPrebuiltBinary(hod android.HostOrDeviceSupported) (*Module, *binaryDecorator) {
	module, binary := NewBinary(hod)
	module.compiler = nil

	prebuilt := &prebuiltBinaryLinker{
		binaryDecorator: binary,
	}
	module.linker = prebuilt

	module.AddProperties(&prebuilt.properties)

	android.InitPrebuiltModule(module, &prebuilt.properties.Srcs)
	return module, binary
}
