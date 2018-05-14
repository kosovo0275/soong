package config

import (
	"android/soong/android"
	"strings"
)

func init() {

	pctx.VariableFunc("TidyDefaultGlobalChecks", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("DEFAULT_GLOBAL_TIDY_CHECKS"); override != "" {
			return override
		}
		return strings.Join([]string{
			"-*",
			"google*",
			"misc-macro-parentheses",
			"performance*",
			"-google-readability*",
			"-google-runtime-references",
		}, ",")
	})

	pctx.VariableFunc("TidyExternalVendorChecks", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("DEFAULT_EXTERNAL_VENDOR_TIDY_CHECKS"); override != "" {
			return override
		}
		return strings.Join([]string{
			"-*",
			"google*",
			"-google-build-using-namespace",
			"-google-default-arguments",
			"-google-explicit-constructor",
			"-google-readability*",
			"-google-runtime-int",
			"-google-runtime-references",
		}, ",")
	})

	pctx.StaticVariable("TidyDefaultHeaderDirs", strings.Join([]string{
		"art/",
		"bionic/",
		"bootable/",
		"build/",
		"cts/",
		"dalvik/",
		"developers/",
		"development/",
		"frameworks/",
		"libcore/",
		"libnativehelper/",
		"system/",
	}, "|"))
}

type PathBasedTidyCheck struct {
	PathPrefix string
	Checks     string
}

const tidyDefault = "${config.TidyDefaultGlobalChecks}"
const tidyExternalVendor = "${config.TidyExternalVendorChecks}"

var DefaultLocalTidyChecks = []PathBasedTidyCheck{
	{"external/", tidyExternalVendor},
	{"external/google", tidyDefault},
	{"external/webrtc", tidyDefault},
	{"frameworks/compile/mclinker/", tidyExternalVendor},
	{"hardware/qcom", tidyExternalVendor},
	{"vendor/", tidyExternalVendor},
	{"vendor/google", tidyDefault},
	{"vendor/google_devices", tidyExternalVendor},
}

var reversedDefaultLocalTidyChecks = reverseTidyChecks(DefaultLocalTidyChecks)

func reverseTidyChecks(in []PathBasedTidyCheck) []PathBasedTidyCheck {
	ret := make([]PathBasedTidyCheck, len(in))
	for i, check := range in {
		ret[len(in)-i-1] = check
	}
	return ret
}

func TidyChecksForDir(dir string) string {
	for _, pathCheck := range reversedDefaultLocalTidyChecks {
		if strings.HasPrefix(dir, pathCheck.PathPrefix) {
			return pathCheck.Checks
		}
	}
	return tidyDefault
}
