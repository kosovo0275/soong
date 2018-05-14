package cc

import (
	"strings"

	"github.com/google/blueprint/proptools"

	"android/soong/cc/config"
)

type TidyProperties struct {
	Tidy *bool

	Tidy_flags []string

	Tidy_checks []string
}

type tidyFeature struct {
	Properties TidyProperties
}

func (tidy *tidyFeature) props() []interface{} {
	return []interface{}{&tidy.Properties}
}

func (tidy *tidyFeature) begin(ctx BaseModuleContext) {
}

func (tidy *tidyFeature) deps(ctx DepsContext, deps Deps) Deps {
	return deps
}

func (tidy *tidyFeature) flags(ctx ModuleContext, flags Flags) Flags {
	CheckBadTidyFlags(ctx, "tidy_flags", tidy.Properties.Tidy_flags)
	CheckBadTidyChecks(ctx, "tidy_checks", tidy.Properties.Tidy_checks)

	if tidy.Properties.Tidy != nil && !*tidy.Properties.Tidy {
		return flags
	}

	if tidy.Properties.Tidy == nil && !ctx.Config().ClangTidy() {
		return flags
	}

	if !flags.Clang {
		return flags
	}

	flags.Tidy = true

	esc := proptools.NinjaAndShellEscape

	flags.TidyFlags = append(flags.TidyFlags, esc(tidy.Properties.Tidy_flags)...)
	if len(flags.TidyFlags) == 0 {
		headerFilter := "-header-filter=\"(" + ctx.ModuleDir() + "|${config.TidyDefaultHeaderDirs})\""
		flags.TidyFlags = append(flags.TidyFlags, headerFilter)
	}

	if !ctx.Config().ClangTidy() {
		flags.TidyFlags = append(flags.TidyFlags, "-quiet")
		flags.TidyFlags = append(flags.TidyFlags, "-extra-arg-before=-fno-caret-diagnostics")
	}

	flags.TidyFlags = append(flags.TidyFlags, "-extra-arg-before=-D__clang_analyzer__")

	tidyChecks := "-checks="
	if checks := ctx.Config().TidyChecks(); len(checks) > 0 {
		tidyChecks += checks
	} else {
		tidyChecks += config.TidyChecksForDir(ctx.ModuleDir())
	}
	if len(tidy.Properties.Tidy_checks) > 0 {
		tidyChecks = tidyChecks + "," + strings.Join(esc(tidy.Properties.Tidy_checks), ",")
	}
	flags.TidyFlags = append(flags.TidyFlags, tidyChecks)

	return flags
}
