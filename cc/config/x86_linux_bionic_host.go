package config

import (
	"strings"

	"android/soong/android"
)

var (
	linuxBionicCflags = ClangFilterUnknownCflags([]string{
		"-fdiagnostics-color",
		"-Wa,--noexecstack",
		"-fPIC",
		"-U_FORTIFY_SOURCE",
		"-D_FORTIFY_SOURCE=2",
		"-fstack-protector-strong",
		"-ffunction-sections",
		"-finline-functions",
		"-finline-limit=300",
		"-fno-short-enums",
		"-funswitch-loops",
		"-funwind-tables",
		"-fno-canonical-system-headers",
		"--gcc-toolchain=${LinuxBionicGccRoot}",
	})

	linuxBionicLdflags = ClangFilterUnknownCflags([]string{
		"-Wl,-z,noexecstack",
		"-Wl,-z,relro",
		"-Wl,-z,now",
		"-Wl,--build-id=md5",
		"-Wl,--warn-shared-textrel",
		"-Wl,--fatal-warnings",
		"-Wl,--hash-style=gnu",
		"-Wl,--no-undefined-version",
		"--gcc-toolchain=${LinuxBionicGccRoot}",
	})

	linuxBionicLldflags = ClangFilterUnknownLldflags(linuxBionicLdflags)
)

func init() {
	pctx.StaticVariable("LinuxBionicCflags", strings.Join(linuxBionicCflags, " "))
	pctx.StaticVariable("LinuxBionicLdflags", strings.Join(linuxBionicLdflags, " "))
	pctx.StaticVariable("LinuxBionicLldflags", strings.Join(linuxBionicLldflags, " "))

	pctx.StaticVariable("LinuxBionicIncludeFlags", bionicHeaders("x86"))

	pctx.StaticVariable("LinuxBionicGccRoot", "${X86_64GccRoot}")
}

type toolchainLinuxBionic struct {
	toolchain64Bit
}

func (t *toolchainLinuxBionic) Name() string {
	return "x86_64"
}

func (t *toolchainLinuxBionic) GccRoot() string {
	return "${config.LinuxBionicGccRoot}"
}

func (t *toolchainLinuxBionic) GccTriple() string {
	return "x86_64-linux-android"
}

func (t *toolchainLinuxBionic) GccVersion() string {
	return "4.9"
}

func (t *toolchainLinuxBionic) Cflags() string {
	return ""
}

func (t *toolchainLinuxBionic) Cppflags() string {
	return ""
}

func (t *toolchainLinuxBionic) Ldflags() string {
	return ""
}

func (t *toolchainLinuxBionic) IncludeFlags() string {
	return "${config.LinuxBionicIncludeFlags}"
}

func (t *toolchainLinuxBionic) ClangTriple() string {

	return "x86_64-linux-android"
}

func (t *toolchainLinuxBionic) ClangCflags() string {
	return "${config.LinuxBionicCflags}"
}

func (t *toolchainLinuxBionic) ClangCppflags() string {
	return ""
}

func (t *toolchainLinuxBionic) ClangLdflags() string {
	return "${config.LinuxBionicLdflags}"
}

func (t *toolchainLinuxBionic) ClangLldflags() string {
	return "${config.LinuxBionicLldflags}"
}

func (t *toolchainLinuxBionic) ToolchainClangCflags() string {
	return "-m64 -march=x86-64" +

		" -U__ANDROID__ -fno-emulated-tls"
}

func (t *toolchainLinuxBionic) ToolchainClangLdflags() string {
	return "-m64"
}

func (t *toolchainLinuxBionic) AvailableLibraries() []string {
	return nil
}

func (t *toolchainLinuxBionic) Bionic() bool {
	return true
}

var toolchainLinuxBionicSingleton Toolchain = &toolchainLinuxBionic{}

func linuxBionicToolchainFactory(arch android.Arch) Toolchain {
	return toolchainLinuxBionicSingleton
}

func init() {
	registerToolchainFactory(android.LinuxBionic, android.X86_64, linuxBionicToolchainFactory)
}
