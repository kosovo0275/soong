package cc

import (
	"android/soong/android"
	"android/soong/genrule"
)

func init() {
	android.RegisterModuleType("cc_genrule", genRuleFactory)
}

func genRuleFactory() android.Module {
	module := genrule.NewGenRule()

	module.Extra = &VendorProperties{}
	module.AddProperties(module.Extra)

	android.InitAndroidArchModule(module, android.HostAndDeviceSupported, android.MultilibBoth)

	return module
}
