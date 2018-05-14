package config

import (
	"sort"
	"strings"
)

var ClangUnknownCflags = sorted([]string{
	"-finline-functions",
	"-finline-limit=64",
	"-fno-canonical-system-headers",
	"-Wno-clobbered",
	"-fno-devirtualize",
	"-fno-tree-sra",
	"-fprefetch-loop-arrays",
	"-funswitch-loops",
	"-Werror=unused-but-set-parameter",
	"-Werror=unused-but-set-variable",
	"-Wmaybe-uninitialized",
	"-Wno-error=clobbered",
	"-Wno-error=maybe-uninitialized",
	"-Wno-error=unused-but-set-parameter",
	"-Wno-error=unused-but-set-variable",
	"-Wno-extended-offsetof",
	"-Wno-free-nonheap-object",
	"-Wno-literal-suffix",
	"-Wno-maybe-uninitialized",
	"-Wno-old-style-declaration",
	"-Wno-psabi",
	"-Wno-unused-but-set-parameter",
	"-Wno-unused-but-set-variable",
	"-Wno-unused-local-typedefs",
	"-Wunused-but-set-parameter",
	"-Wunused-but-set-variable",
	"-fdiagnostics-color",
	"-fgcse-after-reload",
	"-frerun-cse-after-loop",
	"-frename-registers",
	"-fno-strict-volatile-bitfields",
	"-fno-align-jumps",
	"-mthumb-interwork",
	"-fno-builtin-sin",
	"-fno-caller-saves",
	"-fno-early-inlining",
	"-fno-move-loop-invariants",
	"-fno-partial-inlining",
	"-fno-tree-copy-prop",
	"-fno-tree-loop-optimize",
	"-msynci",
	"-mno-synci",
	"-mno-fused-madd",
	"-finline-limit=300",
	"-fno-inline-functions-called-once",
	"-mfpmath=sse",
	"-mbionic",
	"--enable-stdcall-fixup",
})

var ClangUnknownLldflags = sorted([]string{
	"-fuse-ld=gold",
	"-Wl,--icf=safe",
	"-Wl,--fix-cortex-a8",
	"-Wl,--no-fix-cortex-a8",
	"-Wl,-m,aarch64_elf64_le_vec",
})

var ClangLibToolingUnknownCflags = []string{
	"-flto*",
	"-fsanitize*",
}

func init() {
	pctx.StaticVariable("ClangExtraCflags", strings.Join([]string{
		"-D__compiler_offsetof=__builtin_offsetof",
		"-Werror=int-conversion",
		"-Wno-reserved-id-macro",
		"-Wno-format-pedantic",
		"-Wno-unused-command-line-argument",
		"-fcolor-diagnostics",
		"-Wno-expansion-to-defined",
		"-Wno-zero-as-null-pointer-constant",
		"-Wno-deprecated-register",
		"-Wno-sign-compare",
	}, " "))

	pctx.StaticVariable("ClangExtraCppflags", strings.Join([]string{
		"-Wno-inconsistent-missing-override",
		"-Wno-null-dereference",
		"-D_LIBCPP_ENABLE_THREAD_SAFETY_ANNOTATIONS",
		"-Wno-thread-safety-negative",
		"-Wno-gnu-include-next",
	}, " "))

	pctx.StaticVariable("ClangExtraTargetCflags", strings.Join([]string{}, " "))

	pctx.StaticVariable("ClangExtraNoOverrideCflags", strings.Join([]string{
		"-Werror=address-of-temporary",
		"-Werror=return-type",
		"-Wno-tautological-constant-compare",
		"-Wno-tautological-type-limit-compare",
		"-Wno-tautological-unsigned-enum-zero-compare",
		"-Wno-tautological-unsigned-zero-compare",
		"-Wno-null-pointer-arithmetic",
		"-Wno-enum-compare",
		"-Wno-enum-compare-switch",
		"-Wno-c++98-compat-extra-semi",
	}, " "))
}

func ClangFilterUnknownCflags(cflags []string) []string {
	ret := make([]string, 0, len(cflags))
	for _, f := range cflags {
		if !inListSorted(f, ClangUnknownCflags) {
			ret = append(ret, f)
		}
	}

	return ret
}

func ClangFilterUnknownLldflags(lldflags []string) []string {
	ret := make([]string, 0, len(lldflags))
	for _, f := range lldflags {
		if !inListSorted(f, ClangUnknownLldflags) {
			ret = append(ret, f)
		}
	}

	return ret
}

func inListSorted(s string, list []string) bool {
	for _, l := range list {
		if s == l {
			return true
		} else if s < l {
			return false
		}
	}
	return false
}

func sorted(list []string) []string {
	sort.Strings(list)
	return list
}
