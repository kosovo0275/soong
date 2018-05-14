package config

import (
	"strings"

	"android/soong/android"
)

var (
	x86_64Cflags = []string{

		"-Werror=implicit-function-declaration",
	}

	x86_64Cppflags = []string{}

	x86_64Ldflags = []string{
		"-Wl,--hash-style=gnu",
	}

	x86_64Lldflags = ClangFilterUnknownLldflags(x86_64Ldflags)

	x86_64ArchVariantCflags = map[string][]string{
		"": []string{
			"-march=x86-64",
		},
		"haswell": []string{
			"-march=core-avx2",
		},
		"ivybridge": []string{
			"-march=core-avx-i",
		},
		"sandybridge": []string{
			"-march=corei7",
		},
		"silvermont": []string{
			"-march=slm",
		},
	}

	x86_64ArchFeatureCflags = map[string][]string{
		"ssse3":  []string{"-DUSE_SSSE3", "-mssse3"},
		"sse4":   []string{"-msse4"},
		"sse4_1": []string{"-msse4.1"},
		"sse4_2": []string{"-msse4.2"},
		"popcnt": []string{"-mpopcnt"},
		"avx":    []string{"-mavx"},
		"aes_ni": []string{"-maes"},
	}
)

const (
	x86_64GccVersion = "4.9"
)

func init() {
	android.RegisterArchVariants(android.X86_64,
		"haswell",
		"ivybridge",
		"sandybridge",
		"silvermont")
	android.RegisterArchFeatures(android.X86_64,
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"aes_ni",
		"avx",
		"popcnt")
	android.RegisterArchVariantFeatures(android.X86_64, "",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"popcnt")
	android.RegisterArchVariantFeatures(android.X86_64, "haswell",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"aes_ni",
		"avx",
		"popcnt")
	android.RegisterArchVariantFeatures(android.X86_64, "ivybridge",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"aes_ni",
		"avx",
		"popcnt")
	android.RegisterArchVariantFeatures(android.X86_64, "sandybridge",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"popcnt")
	android.RegisterArchVariantFeatures(android.X86_64, "silvermont",
		"ssse3",
		"sse4",
		"sse4_1",
		"sse4_2",
		"aes_ni",
		"popcnt")

	pctx.StaticVariable("x86_64GccVersion", x86_64GccVersion)

	pctx.StaticVariable("X86_64GccRoot",
		"prebuilts/gcc/${HostPrebuiltTag}/x86/x86_64-linux-android-${x86_64GccVersion}")

	pctx.StaticVariable("X86_64ToolchainCflags", "-m64")
	pctx.StaticVariable("X86_64ToolchainLdflags", "-m64")

	pctx.StaticVariable("X86_64Cflags", strings.Join(x86_64Cflags, " "))
	pctx.StaticVariable("X86_64Ldflags", strings.Join(x86_64Ldflags, " "))
	pctx.StaticVariable("X86_64Lldflags", strings.Join(x86_64Lldflags, " "))
	pctx.StaticVariable("X86_64Cppflags", strings.Join(x86_64Cppflags, " "))
	pctx.StaticVariable("X86_64IncludeFlags", bionicHeaders("x86"))

	pctx.StaticVariable("X86_64ClangCflags", strings.Join(ClangFilterUnknownCflags(x86_64Cflags), " "))
	pctx.StaticVariable("X86_64ClangLdflags", strings.Join(ClangFilterUnknownCflags(x86_64Ldflags), " "))
	pctx.StaticVariable("X86_64ClangLldflags", strings.Join(ClangFilterUnknownCflags(x86_64Lldflags), " "))
	pctx.StaticVariable("X86_64ClangCppflags", strings.Join(ClangFilterUnknownCflags(x86_64Cppflags), " "))

	pctx.StaticVariable("X86_64YasmFlags", "-f elf64 -m amd64")

	for variant, cflags := range x86_64ArchVariantCflags {
		pctx.StaticVariable("X86_64"+variant+"VariantCflags", strings.Join(cflags, " "))
		pctx.StaticVariable("X86_64"+variant+"VariantClangCflags",
			strings.Join(ClangFilterUnknownCflags(cflags), " "))
	}
}

type toolchainX86_64 struct {
	toolchain64Bit
	toolchainCflags, toolchainClangCflags string
}

func (t *toolchainX86_64) Name() string {
	return "x86_64"
}

func (t *toolchainX86_64) GccRoot() string {
	return "${config.X86_64GccRoot}"
}

func (t *toolchainX86_64) GccTriple() string {
	return "x86_64-linux-android"
}

func (t *toolchainX86_64) GccVersion() string {
	return x86_64GccVersion
}

func (t *toolchainX86_64) ToolchainLdflags() string {
	return "${config.X86_64ToolchainLdflags}"
}

func (t *toolchainX86_64) ToolchainCflags() string {
	return t.toolchainCflags
}

func (t *toolchainX86_64) Cflags() string {
	return "${config.X86_64Cflags}"
}

func (t *toolchainX86_64) Cppflags() string {
	return "${config.X86_64Cppflags}"
}

func (t *toolchainX86_64) Ldflags() string {
	return "${config.X86_64Ldflags}"
}

func (t *toolchainX86_64) IncludeFlags() string {
	return "${config.X86_64IncludeFlags}"
}

func (t *toolchainX86_64) ClangTriple() string {
	return t.GccTriple()
}

func (t *toolchainX86_64) ToolchainClangLdflags() string {
	return "${config.X86_64ToolchainLdflags}"
}

func (t *toolchainX86_64) ToolchainClangCflags() string {
	return t.toolchainClangCflags
}

func (t *toolchainX86_64) ClangCflags() string {
	return "${config.X86_64ClangCflags}"
}

func (t *toolchainX86_64) ClangCppflags() string {
	return "${config.X86_64ClangCppflags}"
}

func (t *toolchainX86_64) ClangLdflags() string {
	return "${config.X86_64Ldflags}"
}

func (t *toolchainX86_64) ClangLldflags() string {
	return "${config.X86_64Lldflags}"
}

func (t *toolchainX86_64) YasmFlags() string {
	return "${config.X86_64YasmFlags}"
}

func (toolchainX86_64) SanitizerRuntimeLibraryArch() string {
	return "x86_64"
}

func x86_64ToolchainFactory(arch android.Arch) Toolchain {
	toolchainCflags := []string{
		"${config.X86_64ToolchainCflags}",
		"${config.X86_64" + arch.ArchVariant + "VariantCflags}",
	}

	toolchainClangCflags := []string{
		"${config.X86_64ToolchainCflags}",
		"${config.X86_64" + arch.ArchVariant + "VariantClangCflags}",
	}

	for _, feature := range arch.ArchFeatures {
		toolchainCflags = append(toolchainCflags, x86_64ArchFeatureCflags[feature]...)
		toolchainClangCflags = append(toolchainClangCflags, x86_64ArchFeatureCflags[feature]...)
	}

	return &toolchainX86_64{
		toolchainCflags:      strings.Join(toolchainCflags, " "),
		toolchainClangCflags: strings.Join(toolchainClangCflags, " "),
	}
}

func init() {
	registerToolchainFactory(android.Android, android.X86_64, x86_64ToolchainFactory)
}
