package cc

import (
	"strings"
	"sync"

	"android/soong/android"
	"android/soong/cc/config"
)

var (
	lsdumpPaths []string
	sabiLock    sync.Mutex
)

type SAbiProperties struct {
	CreateSAbiDumps        bool `blueprint:"mutated"`
	ReexportedIncludeFlags []string
}

type sabi struct {
	Properties SAbiProperties
}

func (sabimod *sabi) props() []interface{} {
	return []interface{}{&sabimod.Properties}
}

func (sabimod *sabi) begin(ctx BaseModuleContext) {}

func (sabimod *sabi) deps(ctx BaseModuleContext, deps Deps) Deps {
	return deps
}

func inListWithPrefixSearch(flag string, filter []string) bool {

	for _, f := range filter {
		if (f == flag) || (strings.HasSuffix(f, "*") && strings.HasPrefix(flag, strings.TrimSuffix(f, "*"))) {
			return true
		}
	}
	return false
}

func filterOutWithPrefix(list []string, filter []string) (remainder []string) {

	for _, l := range list {
		if !inListWithPrefixSearch(l, filter) {
			remainder = append(remainder, l)
		}
	}
	return
}

func (sabimod *sabi) flags(ctx ModuleContext, flags Flags) Flags {

	flags.ToolingCFlags = filterOutWithPrefix(flags.CFlags, config.ClangLibToolingUnknownCflags)

	if ctx.Arch().CpuVariant == "exynos-m2" {
		flags.ToolingCFlags = append(flags.ToolingCFlags, "-mcpu=cortex-a53")
	}

	return flags
}

func sabiDepsMutator(mctx android.TopDownMutatorContext) {
	if c, ok := mctx.Module().(*Module); ok &&
		((c.isVndk() && c.useVndk()) || inList(c.Name(), llndkLibraries) ||
			(c.sabi != nil && c.sabi.Properties.CreateSAbiDumps)) {
		mctx.VisitDirectDeps(func(m android.Module) {
			tag := mctx.OtherModuleDependencyTag(m)
			switch tag {
			case staticDepTag, staticExportDepTag, lateStaticDepTag, wholeStaticDepTag:

				cc, _ := m.(*Module)
				if cc == nil {
					return
				}
				cc.sabi.Properties.CreateSAbiDumps = true
			}
		})
	}
}
