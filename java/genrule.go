package java

import (
	"android/soong/android"
	"android/soong/genrule"
)

func init() {
	android.RegisterModuleType("java_genrule", genRuleFactory)
	android.RegisterModuleType("java_genrule_host", genRuleFactoryHost)
}

// java_genrule is a genrule that can depend on other java_* objects.
// The cmd may be run multiple times, once for each of the different host/device
// variations.
func genRuleFactory() android.Module {
	module := genrule.NewGenRule()

	android.InitAndroidArchModule(module, android.HostAndDeviceSupported, android.MultilibCommon)

	return module
}

// java_genrule_host is a genrule that can depend on other java_* objects.
// The cmd may be run multiple times, once for each of the different host/device
// variations.
func genRuleFactoryHost() android.Module {
	module := genrule.NewGenRule()

	android.InitAndroidArchModule(module, android.HostSupported, android.MultilibCommon)

	return module
}
