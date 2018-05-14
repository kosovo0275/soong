package cc

import (
	"android/soong/android"
)

type kernelHeadersDecorator struct {
	*libraryDecorator
}

func (stub *kernelHeadersDecorator) link(ctx ModuleContext, flags Flags, deps PathDeps, objs Objects) android.Path {
	if ctx.Device() {
		f := &stub.libraryDecorator.flagExporter
		for _, dir := range ctx.DeviceConfig().DeviceKernelHeaderDirs() {
			f.flags = append(f.flags, "-isystem"+dir)
		}
	}
	return stub.libraryDecorator.linkStatic(ctx, flags, deps, objs)
}

func kernelHeadersFactory() android.Module {
	module, library := NewLibrary(android.HostAndDeviceSupported)
	library.HeaderOnly()

	stub := &kernelHeadersDecorator{
		libraryDecorator: library,
	}

	module.linker = stub

	return module.Init()
}

func init() {
	android.RegisterModuleType("kernel_headers", kernelHeadersFactory)
}
