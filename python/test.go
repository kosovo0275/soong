package python

import (
	"android/soong/android"
)

// This file contains the module types for building Python test.

func init() {
	android.RegisterModuleType("python_test_host", PythonTestHostFactory)
	android.RegisterModuleType("python_test", PythonTestFactory)
}

type testDecorator struct {
	*binaryDecorator
}

func (test *testDecorator) install(ctx android.ModuleContext, file android.Path) {
	test.binaryDecorator.pythonInstaller.dir = "nativetest"
	test.binaryDecorator.pythonInstaller.dir64 = "nativetest64"

	test.binaryDecorator.pythonInstaller.relative = ctx.ModuleName()

	test.binaryDecorator.pythonInstaller.install(ctx, file)
}

func NewTest(hod android.HostOrDeviceSupported) *Module {
	module, binary := NewBinary(hod)

	binary.pythonInstaller = NewPythonInstaller("nativetest", "nativetest64")

	test := &testDecorator{binaryDecorator: binary}

	module.bootstrapper = test
	module.installer = test

	return module
}

func PythonTestHostFactory() android.Module {
	module := NewTest(android.HostSupportedNoCross)

	return module.Init()
}

func PythonTestFactory() android.Module {
	module := NewTest(android.HostAndDeviceSupported)
	module.multilib = android.MultilibBoth

	return module.Init()
}
