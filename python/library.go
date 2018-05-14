package python

// This file contains the module types for building Python library.

import (
	"android/soong/android"
)

func init() {
	android.RegisterModuleType("python_library_host", PythonLibraryHostFactory)
	android.RegisterModuleType("python_library", PythonLibraryFactory)
}

func PythonLibraryHostFactory() android.Module {
	module := newModule(android.HostSupportedNoCross, android.MultilibFirst)

	return module.Init()
}

func PythonLibraryFactory() android.Module {
	module := newModule(android.HostAndDeviceSupported, android.MultilibBoth)

	return module.Init()
}
