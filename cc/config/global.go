package config

import (
	"fmt"
	"strings"

	"android/soong/android"
)

var (
	commonGlobalCflags = []string{
		"-DANDROID",
		"-fmessage-length=55",
		"-W",
		"-Wall",
		"-Wno-unused",
		"-Winit-self",
		"-Wpointer-arith",
		"-no-canonical-prefixes",
		"-fno-canonical-system-headers",
		"-DNDEBUG",
		"-UDEBUG",
		"-fno-exceptions",
		"-Wno-multichar",
		"-O2",
		"-fno-strict-aliasing",
	}

	commonGlobalConlyflags = []string{}

	deviceGlobalCflags = []string{
		"-fdiagnostics-color",
		"-ffunction-sections",
		"-fdata-sections",
		"-fno-short-enums",
		"-funwind-tables",
		"-fstack-protector-strong",
		"-Wa,--noexecstack",
		"-Werror=return-type",
		"-Werror=non-virtual-dtor",
		"-Werror=address",
		"-Werror=sequence-point",
	}

	deviceGlobalCppflags = []string{
		"-fvisibility-inlines-hidden",
	}

	deviceGlobalLdflags = []string{
		"-Wl,-z,noexecstack",
		"-Wl,-z,relro",
		"-Wl,-z,now",
		"-Wl,--build-id=md5",
		"-Wl,--warn-shared-textrel",
		"-Wl,--fatal-warnings",
		"-Wl,--no-undefined-version",
	}

	deviceGlobalLldflags = append(ClangFilterUnknownLldflags(deviceGlobalLdflags),
		[]string{
			"-Wl,--pack-dyn-relocs=android",
			"-fuse-ld=lld",
		}...)

	hostGlobalCflags = []string{}

	hostGlobalCppflags = []string{}

	hostGlobalLdflags = []string{}

	hostGlobalLldflags = []string{"-fuse-ld=lld"}

	commonGlobalCppflags = []string{
		"-Wsign-promo",
	}

	noOverrideGlobalCflags = []string{
		"-fPIC",
		"-Wno-return-type",
		"-Wno-old-style-cast",
		"-Wno-unused-variable",
		"-Wno-unused-function",
		"-Wno-unused-parameter",
	}

	IllegalFlags = []string{
		"-w",
	}

	CStdVersion               = "gnu11"
	CppStdVersion             = "gnu++14"
	GccCppStdVersion          = "gnu++11"
	ExperimentalCStdVersion   = "gnu11"
	ExperimentalCppStdVersion = "gnu++1z"

	NdkMaxPrebuiltVersionInt = 27

	ClangDefaultBase         = android.Prefix()
	ClangDefaultVersion      = "7.0.0"
	ClangDefaultShortVersion = "7.0.0"

	WarningAllowedProjects = []string{
		"art/",
		"bionic/",
		"dalvik/",
		"device/",
		"external/",
		"frameworks/",
		"system/core/",
		"vendor/",
	}

	WarningAllowedOldProjects = []string{}
)

var pctx = android.NewPackageContext("android/soong/cc/config")

func init() {
	pctx.StaticVariable("CommonGlobalCflags", strings.Join(commonGlobalCflags, " "))
	pctx.StaticVariable("CommonGlobalConlyflags", strings.Join(commonGlobalConlyflags, " "))
	pctx.StaticVariable("DeviceGlobalCflags", strings.Join(deviceGlobalCflags, " "))
	pctx.StaticVariable("DeviceGlobalCppflags", strings.Join(deviceGlobalCppflags, " "))
	pctx.StaticVariable("DeviceGlobalLdflags", strings.Join(deviceGlobalLdflags, " "))
	pctx.StaticVariable("DeviceGlobalLldflags", strings.Join(deviceGlobalLldflags, " "))
	pctx.StaticVariable("HostGlobalCflags", strings.Join(hostGlobalCflags, " "))
	pctx.StaticVariable("HostGlobalCppflags", strings.Join(hostGlobalCppflags, " "))
	pctx.StaticVariable("HostGlobalLdflags", strings.Join(hostGlobalLdflags, " "))
	pctx.StaticVariable("HostGlobalLldflags", strings.Join(hostGlobalLldflags, " "))
	pctx.StaticVariable("NoOverrideGlobalCflags", strings.Join(noOverrideGlobalCflags, " "))
	pctx.StaticVariable("CommonGlobalCppflags", strings.Join(commonGlobalCppflags, " "))
	pctx.StaticVariable("CommonClangGlobalCflags", strings.Join(append(ClangFilterUnknownCflags(commonGlobalCflags), "${ClangExtraCflags}"), " "))
	pctx.StaticVariable("DeviceClangGlobalCflags", strings.Join(append(ClangFilterUnknownCflags(deviceGlobalCflags), "${ClangExtraTargetCflags}"), " "))
	pctx.StaticVariable("HostClangGlobalCflags", strings.Join(ClangFilterUnknownCflags(hostGlobalCflags), " "))
	pctx.StaticVariable("NoOverrideClangGlobalCflags", strings.Join(append(ClangFilterUnknownCflags(noOverrideGlobalCflags), "${ClangExtraNoOverrideCflags}"), " "))
	pctx.StaticVariable("CommonClangGlobalCppflags", strings.Join(append(ClangFilterUnknownCflags(commonGlobalCppflags), "${ClangExtraCppflags}"), " "))
	pctx.PrefixedExistentPathsForSourcesVariable("CommonGlobalIncludes", "-I",
		[]string{
			"system/core/include",
			"system/media/audio/include",
			"hardware/libhardware/include",
			"hardware/libhardware_legacy/include",
			"hardware/ril/include",
			"libnativehelper/include",
			"frameworks/native/include",
			"frameworks/native/opengl/include",
			"frameworks/av/include",
		})
	pctx.PrefixedExistentPathsForSourcesVariable("CommonNativehelperInclude", "-I", []string{"libnativehelper/include_deprecated"})

	pctx.StaticVariable("ClangDefaultBase", ClangDefaultBase)
	pctx.VariableFunc("ClangBase", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("LLVM_PREBUILTS_BASE"); override != "" {
			return override
		}
		return "${ClangDefaultBase}"
	})
	pctx.VariableFunc("ClangVersion", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("LLVM_PREBUILTS_VERSION"); override != "" {
			return override
		}
		return ClangDefaultVersion
	})
	pctx.StaticVariable("ClangPath", android.Prefix())
	pctx.StaticVariable("ClangBin", "${ClangPath}/bin")

	pctx.VariableFunc("ClangShortVersion", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("LLVM_RELEASE_VERSION"); override != "" {
			return override
		}
		return ClangDefaultShortVersion
	})
	pctx.StaticVariable("ClangAsanLibDir", "${ClangPath}/lib/clang/${ClangShortVersion}/lib/linux")
	pctx.StaticVariable("LLVMGoldPlugin", "${ClangPath}/lib/LLVMgold.so")
	pctx.StaticVariable("RSClangBase", android.Prefix())
	pctx.StaticVariable("RSClangVersion", ClangDefaultVersion)
	pctx.StaticVariable("RSReleaseVersion", ClangDefaultShortVersion)
	pctx.StaticVariable("RSLLVMPrebuiltsPath", "${RSClangBase}/bin")
	pctx.StaticVariable("RSIncludePath", "${RSClangBase}/lib/clang/${RSReleaseVersion}/include")

	pctx.PrefixedExistentPathsForSourcesVariable("RsGlobalIncludes", "-I",
		[]string{
			"external/clang/lib/Headers",
			"frameworks/rs/script_api/include",
		})

	pctx.VariableFunc("CcWrapper", func(ctx android.PackageVarContext) string {
		if override := ctx.Config().Getenv("CC_WRAPPER"); override != "" {
			return override + " "
		}
		return ""
	})
}

var HostPrebuiltTag = pctx.VariableConfigMethod("HostPrebuiltTag", android.Config.PrebuiltOS)

func bionicHeaders(kernelArch string) string {
	return strings.Join([]string{
		/*
			"-isystem bionic/libc/include",
			"-isystem bionic/libc/kernel/uapi",
			"-isystem bionic/libc/kernel/uapi/asm-" + kernelArch,
			"-isystem bionic/libc/kernel/android/scsi",
			"-isystem bionic/libc/kernel/android/uapi",
		*/
	}, " ")
}

func replaceFirst(slice []string, from, to string) {
	if slice[0] != from {
		panic(fmt.Errorf("Expected %q, found %q", from, to))
	}
	slice[0] = to
}
