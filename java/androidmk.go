package java

import (
	"fmt"
	"io"
	"strings"

	"android/soong/android"
)

func (library *Library) AndroidMk() android.AndroidMkData {
	return android.AndroidMkData{
		Class:      "JAVA_LIBRARIES",
		OutputFile: android.OptionalPathForPath(library.implementationJarFile),
		Include:    "$(BUILD_SYSTEM)/soong_java_prebuilt.mk",
		Extra: []android.AndroidMkExtraFunc{
			func(w io.Writer, outputFile android.Path) {
				if len(library.logtagsSrcs) > 0 {
					var logtags []string
					for _, l := range library.logtagsSrcs {
						logtags = append(logtags, l.Rel())
					}
					fmt.Fprintln(w, "LOCAL_LOGTAGS_FILES :=", strings.Join(logtags, " "))
				}

				if library.properties.Installable != nil && *library.properties.Installable == false {
					fmt.Fprintln(w, "LOCAL_UNINSTALLABLE_MODULE := true")
				}
				if library.dexJarFile != nil {
					fmt.Fprintln(w, "LOCAL_SOONG_DEX_JAR :=", library.dexJarFile.String())
					if library.deviceProperties.Dex_preopt.Enabled != nil {
						fmt.Fprintln(w, "LOCAL_DEX_PREOPT :=", *library.deviceProperties.Dex_preopt.Enabled)
					}
					if library.deviceProperties.Dex_preopt.App_image != nil {
						fmt.Fprintln(w, "LOCAL_DEX_PREOPT_APP_IMAGE :=", *library.deviceProperties.Dex_preopt.App_image)
					}
					if library.deviceProperties.Dex_preopt.Profile_guided != nil {
						fmt.Fprintln(w, "LOCAL_DEX_PREOPT_GENERATE_PROFILE :=", *library.deviceProperties.Dex_preopt.Profile_guided)
					}
					if library.deviceProperties.Dex_preopt.Profile != nil {
						fmt.Fprintln(w, "LOCAL_DEX_PREOPT_GENERATE_PROFILE := true")
						fmt.Fprintln(w, "LOCAL_DEX_PREOPT_PROFILE_CLASS_LISTING := $(LOCAL_PATH)/"+*library.deviceProperties.Dex_preopt.Profile)
					}
				}
				fmt.Fprintln(w, "LOCAL_SDK_VERSION :=", String(library.deviceProperties.Sdk_version))
				fmt.Fprintln(w, "LOCAL_SOONG_HEADER_JAR :=", library.headerJarFile.String())

				if library.jacocoReportClassesFile != nil {
					fmt.Fprintln(w, "LOCAL_SOONG_JACOCO_REPORT_CLASSES_JAR :=", library.jacocoReportClassesFile.String())
				}

				// Temporary hack: export sources used to compile framework.jar to Make
				// to be used for droiddoc
				// TODO(ccross): remove this once droiddoc is in soong
				if library.Name() == "framework" {
					fmt.Fprintln(w, "SOONG_FRAMEWORK_SRCS :=", strings.Join(library.compiledJavaSrcs.Strings(), " "))
					fmt.Fprintln(w, "SOONG_FRAMEWORK_SRCJARS :=", strings.Join(library.compiledSrcJars.Strings(), " "))
				}
			},
		},
		Custom: func(w io.Writer, name, prefix, moduleDir string, data android.AndroidMkData) {
			android.WriteAndroidMkData(w, data)

			if Bool(library.deviceProperties.Hostdex) && !library.Host() {
				fmt.Fprintln(w, "include $(CLEAR_VARS)")
				fmt.Fprintln(w, "LOCAL_MODULE := "+name+"-hostdex")
				fmt.Fprintln(w, "LOCAL_IS_HOST_MODULE := true")
				fmt.Fprintln(w, "LOCAL_MODULE_CLASS := JAVA_LIBRARIES")
				fmt.Fprintln(w, "LOCAL_PREBUILT_MODULE_FILE :=", library.implementationJarFile.String())
				if library.properties.Installable != nil && *library.properties.Installable == false {
					fmt.Fprintln(w, "LOCAL_UNINSTALLABLE_MODULE := true")
				}
				if library.dexJarFile != nil {
					fmt.Fprintln(w, "LOCAL_SOONG_DEX_JAR :=", library.dexJarFile.String())
				}
				fmt.Fprintln(w, "LOCAL_SOONG_HEADER_JAR :=", library.implementationJarFile.String())
				fmt.Fprintln(w, "LOCAL_REQUIRED_MODULES := "+strings.Join(data.Required, " "))
				fmt.Fprintln(w, "include $(BUILD_SYSTEM)/soong_java_prebuilt.mk")
			}
		},
	}
}

func (j *Test) AndroidMk() android.AndroidMkData {
	data := j.Library.AndroidMk()
	data.Extra = append(data.Extra, func(w io.Writer, outputFile android.Path) {
		fmt.Fprintln(w, "LOCAL_MODULE_TAGS := tests")
		if len(j.testProperties.Test_suites) > 0 {
			fmt.Fprintln(w, "LOCAL_COMPATIBILITY_SUITE :=",
				strings.Join(j.testProperties.Test_suites, " "))
		}
	})

	return data
}

func (prebuilt *Import) AndroidMk() android.AndroidMkData {
	return android.AndroidMkData{
		Class:      "JAVA_LIBRARIES",
		OutputFile: android.OptionalPathForPath(prebuilt.combinedClasspathFile),
		Include:    "$(BUILD_SYSTEM)/soong_java_prebuilt.mk",
		Extra: []android.AndroidMkExtraFunc{
			func(w io.Writer, outputFile android.Path) {
				fmt.Fprintln(w, "LOCAL_UNINSTALLABLE_MODULE := ", !Bool(prebuilt.properties.Installable))
				fmt.Fprintln(w, "LOCAL_SOONG_HEADER_JAR :=", prebuilt.combinedClasspathFile.String())
				fmt.Fprintln(w, "LOCAL_SDK_VERSION :=", String(prebuilt.properties.Sdk_version))
			},
		},
	}
}

func (prebuilt *AARImport) AndroidMk() android.AndroidMkData {
	return android.AndroidMkData{
		Class:      "JAVA_LIBRARIES",
		OutputFile: android.OptionalPathForPath(prebuilt.classpathFile),
		Include:    "$(BUILD_SYSTEM)/soong_java_prebuilt.mk",
		Extra: []android.AndroidMkExtraFunc{
			func(w io.Writer, outputFile android.Path) {
				fmt.Fprintln(w, "LOCAL_UNINSTALLABLE_MODULE := true")
				fmt.Fprintln(w, "LOCAL_DEX_PREOPT := false")
				fmt.Fprintln(w, "LOCAL_SOONG_HEADER_JAR :=", prebuilt.classpathFile.String())
				fmt.Fprintln(w, "LOCAL_SOONG_RESOURCE_EXPORT_PACKAGE :=", prebuilt.exportPackage.String())
				fmt.Fprintln(w, "LOCAL_SOONG_EXPORT_PROGUARD_FLAGS :=", prebuilt.proguardFlags.String())
				fmt.Fprintln(w, "LOCAL_SOONG_STATIC_LIBRARY_EXTRA_PACKAGES :=", prebuilt.extraAaptPackagesFile.String())
				fmt.Fprintln(w, "LOCAL_SDK_VERSION :=", String(prebuilt.properties.Sdk_version))
			},
		},
	}
}

func (binary *Binary) AndroidMk() android.AndroidMkData {

	if !binary.isWrapperVariant {
		return android.AndroidMkData{
			Class:      "JAVA_LIBRARIES",
			OutputFile: android.OptionalPathForPath(binary.implementationJarFile),
			Include:    "$(BUILD_SYSTEM)/soong_java_prebuilt.mk",
			Custom: func(w io.Writer, name, prefix, moduleDir string, data android.AndroidMkData) {
				android.WriteAndroidMkData(w, data)

				fmt.Fprintln(w, "jar_installed_module := $(LOCAL_INSTALLED_MODULE)")
			},
		}
	} else {
		return android.AndroidMkData{
			Class:      "EXECUTABLES",
			OutputFile: android.OptionalPathForPath(binary.wrapperFile),
			Extra: []android.AndroidMkExtraFunc{
				func(w io.Writer, outputFile android.Path) {
					fmt.Fprintln(w, "LOCAL_STRIP_MODULE := false")
				},
			},
			Custom: func(w io.Writer, name, prefix, moduleDir string, data android.AndroidMkData) {
				android.WriteAndroidMkData(w, data)

				// Ensure that the wrapper script timestamp is always updated when the jar is updated
				fmt.Fprintln(w, "$(LOCAL_INSTALLED_MODULE): $(jar_installed_module)")
				fmt.Fprintln(w, "jar_installed_module :=")
			},
		}
	}
}

func (app *AndroidApp) AndroidMk() android.AndroidMkData {
	return android.AndroidMkData{
		Class:      "APPS",
		OutputFile: android.OptionalPathForPath(app.outputFile),
		Include:    "$(BUILD_SYSTEM)/soong_app_prebuilt.mk",
		Extra: []android.AndroidMkExtraFunc{
			func(w io.Writer, outputFile android.Path) {
				fmt.Fprintln(w, "LOCAL_SOONG_RESOURCE_EXPORT_PACKAGE :=", app.exportPackage.String())
				if app.dexJarFile != nil {
					fmt.Fprintln(w, "LOCAL_SOONG_DEX_JAR :=", app.dexJarFile.String())
				}
				if app.implementationJarFile != nil {
					fmt.Fprintln(w, "LOCAL_SOONG_CLASSES_JAR :=", app.implementationJarFile)
				}
				if app.headerJarFile != nil {
					fmt.Fprintln(w, "LOCAL_SOONG_HEADER_JAR :=", app.headerJarFile.String())
				}
				if app.jacocoReportClassesFile != nil {
					fmt.Fprintln(w, "LOCAL_SOONG_JACOCO_REPORT_CLASSES_JAR :=", app.jacocoReportClassesFile.String())
				}
				if app.proguardDictionary != nil {
					fmt.Fprintln(w, "LOCAL_SOONG_PROGUARD_DICT :=", app.proguardDictionary.String())
				}

				if app.Name() == "framework-res" {
					fmt.Fprintln(w, "LOCAL_MODULE_PATH := $(TARGET_OUT_JAVA_LIBRARIES)")
					// Make base_rules.mk not put framework-res in a subdirectory called
					// framework_res.
					fmt.Fprintln(w, "LOCAL_NO_STANDARD_LIBRARIES := true")
				}

				if len(app.rroDirs) > 0 {
					// Reverse the order, Soong stores rroDirs in aapt2 order (low to high priority), but Make
					// expects it in LOCAL_RESOURCE_DIRS order (high to low priority).
					fmt.Fprintln(w, "LOCAL_SOONG_RRO_DIRS :=", strings.Join(android.ReversePaths(app.rroDirs).Strings(), " "))
				}

				if Bool(app.appProperties.Export_package_resources) {
					fmt.Fprintln(w, "LOCAL_EXPORT_PACKAGE_RESOURCES := true")
				}

				fmt.Fprintln(w, "LOCAL_FULL_MANIFEST_FILE :=", app.manifestPath.String())

				if Bool(app.appProperties.Privileged) {
					fmt.Fprintln(w, "LOCAL_PRIVILEGED_MODULE := true")
				}

				fmt.Fprintln(w, "LOCAL_CERTIFICATE :=", app.certificate.pem.String())
			},
		},
	}
}

func (a *AndroidLibrary) AndroidMk() android.AndroidMkData {
	data := a.Library.AndroidMk()

	data.Extra = append(data.Extra, func(w io.Writer, outputFile android.Path) {
		if a.proguardDictionary != nil {
			fmt.Fprintln(w, "LOCAL_SOONG_PROGUARD_DICT :=", a.proguardDictionary.String())
		}

		if a.Name() == "framework-res" {
			fmt.Fprintln(w, "LOCAL_MODULE_PATH := $(TARGET_OUT_JAVA_LIBRARIES)")
			// Make base_rules.mk not put framework-res in a subdirectory called
			// framework_res.
			fmt.Fprintln(w, "LOCAL_NO_STANDARD_LIBRARIES := true")
		}

		fmt.Fprintln(w, "LOCAL_SOONG_RESOURCE_EXPORT_PACKAGE :=", a.exportPackage.String())
		fmt.Fprintln(w, "LOCAL_SOONG_STATIC_LIBRARY_EXTRA_PACKAGES :=", a.extraAaptPackagesFile.String())
		fmt.Fprintln(w, "LOCAL_FULL_MANIFEST_FILE :=", a.manifestPath.String())
		fmt.Fprintln(w, "LOCAL_SOONG_EXPORT_PROGUARD_FLAGS :=",
			strings.Join(a.exportedProguardFlagFiles.Strings(), " "))
		fmt.Fprintln(w, "LOCAL_UNINSTALLABLE_MODULE := true")
		fmt.Fprintln(w, "LOCAL_DEX_PREOPT := false")
	})

	return data
}

func (jd *Javadoc) AndroidMk() android.AndroidMkData {
	return android.AndroidMkData{
		Class:      "JAVA_LIBRARIES",
		OutputFile: android.OptionalPathForPath(jd.stubsSrcJar),
		Include:    "$(BUILD_SYSTEM)/soong_java_prebuilt.mk",
		Extra: []android.AndroidMkExtraFunc{
			func(w io.Writer, outputFile android.Path) {
				if BoolDefault(jd.properties.Installable, true) {
					fmt.Fprintln(w, "LOCAL_DROIDDOC_DOC_ZIP := ", jd.docZip.String())
				}
				if jd.stubsSrcJar != nil {
					fmt.Fprintln(w, "LOCAL_DROIDDOC_STUBS_SRCJAR := ", jd.stubsSrcJar.String())
				}
			},
		},
	}
}

func (ddoc *Droiddoc) AndroidMk() android.AndroidMkData {
	return android.AndroidMkData{
		Class:      "JAVA_LIBRARIES",
		OutputFile: android.OptionalPathForPath(ddoc.stubsSrcJar),
		Include:    "$(BUILD_SYSTEM)/soong_java_prebuilt.mk",
		Extra: []android.AndroidMkExtraFunc{
			func(w io.Writer, outputFile android.Path) {
				if BoolDefault(ddoc.Javadoc.properties.Installable, true) {
					fmt.Fprintln(w, "LOCAL_DROIDDOC_DOC_ZIP := ", ddoc.Javadoc.docZip.String())
				}
				if ddoc.Javadoc.stubsSrcJar != nil {
					fmt.Fprintln(w, "LOCAL_DROIDDOC_STUBS_SRCJAR := ", ddoc.Javadoc.stubsSrcJar.String())
				}
				apiFilePrefix := "INTERNAL_PLATFORM_"
				if String(ddoc.properties.Api_tag_name) != "" {
					apiFilePrefix += String(ddoc.properties.Api_tag_name) + "_"
				}
				if String(ddoc.properties.Api_filename) != "" {
					fmt.Fprintln(w, apiFilePrefix+"API_FILE := ", ddoc.apiFile.String())
				}
				if String(ddoc.properties.Private_api_filename) != "" {
					fmt.Fprintln(w, apiFilePrefix+"PRIVATE_API_FILE := ", ddoc.privateApiFile.String())
				}
				if String(ddoc.properties.Private_dex_api_filename) != "" {
					fmt.Fprintln(w, apiFilePrefix+"PRIVATE_DEX_API_FILE := ", ddoc.privateDexApiFile.String())
				}
				if String(ddoc.properties.Removed_api_filename) != "" {
					fmt.Fprintln(w, apiFilePrefix+"REMOVED_API_FILE := ", ddoc.removedApiFile.String())
				}
				if String(ddoc.properties.Removed_dex_api_filename) != "" {
					fmt.Fprintln(w, apiFilePrefix+"REMOVED_DEX_API_FILE := ", ddoc.removedDexApiFile.String())
				}
				if String(ddoc.properties.Exact_api_filename) != "" {
					fmt.Fprintln(w, apiFilePrefix+"EXACT_API_FILE := ", ddoc.exactApiFile.String())
				}
			},
		},
	}
}
