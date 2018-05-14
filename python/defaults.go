package python

import (
	"android/soong/android"
)

func init() {
	android.RegisterModuleType("python_defaults", defaultsFactory)
}

type Defaults struct {
	android.ModuleBase
	android.DefaultsModuleBase
}

func (d *Defaults) GenerateAndroidBuildActions(ctx android.ModuleContext) {
}

func (d *Defaults) DepsMutator(ctx android.BottomUpMutatorContext) {
}

func defaultsFactory() android.Module {
	return DefaultsFactory()
}

func DefaultsFactory(props ...interface{}) android.Module {
	module := &Defaults{}

	module.AddProperties(props...)
	module.AddProperties(
		&BaseProperties{},
	)

	android.InitDefaultsModule(module)

	return module
}
