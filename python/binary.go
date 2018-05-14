package python

// This file contains the module types for building Python binary.

import (
	"fmt"

	"android/soong/android"
)

func init() {
	android.RegisterModuleType("python_binary_host", PythonBinaryHostFactory)
}

type BinaryProperties struct {
	// the name of the source file that is the main entry point of the program.
	// this file must also be listed in srcs.
	// If left unspecified, module name is used instead.
	// If name doesn’t match any filename in srcs, main must be specified.
	Main *string `android:"arch_variant"`

	// set the name of the output binary.
	Stem *string `android:"arch_variant"`

	// append to the name of the output binary.
	Suffix *string `android:"arch_variant"`

	// list of compatibility suites (for example "cts", "vts") that the module should be
	// installed into.
	Test_suites []string `android:"arch_variant"`
}

type binaryDecorator struct {
	binaryProperties BinaryProperties

	*pythonInstaller
}

type IntermPathProvider interface {
	IntermPathForModuleOut() android.OptionalPath
}

var (
	stubTemplateHost = "build/soong/python/scripts/stub_template_host.txt"
)

func NewBinary(hod android.HostOrDeviceSupported) (*Module, *binaryDecorator) {
	module := newModule(hod, android.MultilibFirst)
	decorator := &binaryDecorator{pythonInstaller: NewPythonInstaller("bin", "")}

	module.bootstrapper = decorator
	module.installer = decorator

	return module, decorator
}

func PythonBinaryHostFactory() android.Module {
	module, _ := NewBinary(android.HostSupportedNoCross)

	return module.Init()
}

func (binary *binaryDecorator) bootstrapperProps() []interface{} {
	return []interface{}{&binary.binaryProperties}
}

func (binary *binaryDecorator) bootstrap(ctx android.ModuleContext, actualVersion string,
	embeddedLauncher bool, srcsPathMappings []pathMapping, srcsZip android.Path,
	depsSrcsZips android.Paths) android.OptionalPath {

	main := binary.getPyMainFile(ctx, srcsPathMappings)

	var launcherPath android.Path
	if embeddedLauncher {
		ctx.VisitDirectDepsWithTag(launcherTag, func(m android.Module) {
			if provider, ok := m.(IntermPathProvider); ok {
				if launcherPath != nil {
					panic(fmt.Errorf("launcher path was found before: %q",
						launcherPath))
				}
				launcherPath = provider.IntermPathForModuleOut().Path()
			}
		})
	}

	binFile := registerBuildActionForParFile(ctx, embeddedLauncher, launcherPath,
		binary.getHostInterpreterName(ctx, actualVersion),
		main, binary.getStem(ctx), append(android.Paths{srcsZip}, depsSrcsZips...))

	return android.OptionalPathForPath(binFile)
}

// get host interpreter name.
func (binary *binaryDecorator) getHostInterpreterName(ctx android.ModuleContext,
	actualVersion string) string {
	var interp string
	switch actualVersion {
	case pyVersion2:
		interp = "python2.7"
	case pyVersion3:
		interp = "python3"
	default:
		panic(fmt.Errorf("unknown Python actualVersion: %q for module: %q.",
			actualVersion, ctx.ModuleName()))
	}

	return interp
}

// find main program path within runfiles tree.
func (binary *binaryDecorator) getPyMainFile(ctx android.ModuleContext,
	srcsPathMappings []pathMapping) string {
	var main string
	if String(binary.binaryProperties.Main) == "" {
		main = ctx.ModuleName() + pyExt
	} else {
		main = String(binary.binaryProperties.Main)
	}

	for _, path := range srcsPathMappings {
		if main == path.src.Rel() {
			return path.dest
		}
	}
	ctx.PropertyErrorf("main", "%q is not listed in srcs.", main)

	return ""
}

func (binary *binaryDecorator) getStem(ctx android.ModuleContext) string {
	stem := ctx.ModuleName()
	if String(binary.binaryProperties.Stem) != "" {
		stem = String(binary.binaryProperties.Stem)
	}

	return stem + String(binary.binaryProperties.Suffix)
}
