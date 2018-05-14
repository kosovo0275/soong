package config

import (
	"strings"

	"android/soong/android"
)

var (
	mips64Cflags = []string{
		"-Umips",

		"-Werror=implicit-function-declaration",
	}

	mips64ClangCflags = append(mips64Cflags, []string{
		"-fintegrated-as",
	}...)

	mips64Cppflags = []string{}

	mips64Ldflags = []string{
		"-Wl,--allow-shlib-undefined",
	}

	mips64ArchVariantCflags = map[string][]string{
		"mips64r2": []string{
			"-mips64r2",
			"-msynci",
		},
		"mips64r6": []string{
			"-mips64r6",
			"-msynci",
		},
	}
)

const (
	mips64GccVersion = "4.9"
)

func init() {
	android.RegisterArchVariants(android.Mips64,
		"mips64r2",
		"mips64r6")
	android.RegisterArchFeatures(android.Mips64,
		"rev6",
		"msa")
	android.RegisterArchVariantFeatures(android.Mips64, "mips64r6",
		"rev6")

	pctx.StaticVariable("mips64GccVersion", mips64GccVersion)

	pctx.StaticVariable("Mips64GccRoot",
		"prebuilts/gcc/${HostPrebuiltTag}/mips/mips64el-linux-android-${mips64GccVersion}")

	pctx.StaticVariable("Mips64Cflags", strings.Join(mips64Cflags, " "))
	pctx.StaticVariable("Mips64Ldflags", strings.Join(mips64Ldflags, " "))
	pctx.StaticVariable("Mips64Cppflags", strings.Join(mips64Cppflags, " "))
	pctx.StaticVariable("Mips64IncludeFlags", bionicHeaders("mips"))

	pctx.StaticVariable("Mips64ClangCflags", strings.Join(ClangFilterUnknownCflags(mips64ClangCflags), " "))
	pctx.StaticVariable("Mips64ClangLdflags", strings.Join(ClangFilterUnknownCflags(mips64Ldflags), " "))
	pctx.StaticVariable("Mips64ClangCppflags", strings.Join(ClangFilterUnknownCflags(mips64Cppflags), " "))

	for variant, cflags := range mips64ArchVariantCflags {
		pctx.StaticVariable("Mips64"+variant+"VariantCflags", strings.Join(cflags, " "))
		pctx.StaticVariable("Mips64"+variant+"VariantClangCflags",
			strings.Join(ClangFilterUnknownCflags(cflags), " "))
	}
}

type toolchainMips64 struct {
	toolchain64Bit
	cflags, clangCflags                   string
	toolchainCflags, toolchainClangCflags string
}

func (t *toolchainMips64) Name() string {
	return "mips64"
}

func (t *toolchainMips64) GccRoot() string {
	return "${config.Mips64GccRoot}"
}

func (t *toolchainMips64) GccTriple() string {
	return "mips64el-linux-android"
}

func (t *toolchainMips64) GccVersion() string {
	return mips64GccVersion
}

func (t *toolchainMips64) ToolchainCflags() string {
	return t.toolchainCflags
}

func (t *toolchainMips64) Cflags() string {
	return t.cflags
}

func (t *toolchainMips64) Cppflags() string {
	return "${config.Mips64Cppflags}"
}

func (t *toolchainMips64) Ldflags() string {
	return "${config.Mips64Ldflags}"
}

func (t *toolchainMips64) IncludeFlags() string {
	return "${config.Mips64IncludeFlags}"
}

func (t *toolchainMips64) ClangTriple() string {
	return t.GccTriple()
}

func (t *toolchainMips64) ToolchainClangCflags() string {
	return t.toolchainClangCflags
}

func (t *toolchainMips64) ClangAsflags() string {
	return "-fno-integrated-as"
}

func (t *toolchainMips64) ClangCflags() string {
	return t.clangCflags
}

func (t *toolchainMips64) ClangCppflags() string {
	return "${config.Mips64ClangCppflags}"
}

func (t *toolchainMips64) ClangLdflags() string {
	return "${config.Mips64ClangLdflags}"
}

func (t *toolchainMips64) ClangLldflags() string {

	return "${config.Mips64ClangLdflags}"
}

func (toolchainMips64) SanitizerRuntimeLibraryArch() string {
	return "mips64"
}

func mips64ToolchainFactory(arch android.Arch) Toolchain {
	return &toolchainMips64{
		cflags:               "${config.Mips64Cflags}",
		clangCflags:          "${config.Mips64ClangCflags}",
		toolchainCflags:      "${config.Mips64" + arch.ArchVariant + "VariantCflags}",
		toolchainClangCflags: "${config.Mips64" + arch.ArchVariant + "VariantClangCflags}",
	}
}

func init() {
	registerToolchainFactory(android.Android, android.Mips64, mips64ToolchainFactory)
}
