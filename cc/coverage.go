package cc

import (
	"android/soong/android"
)

type CoverageProperties struct {
	Native_coverage *bool

	CoverageEnabled bool `blueprint:"mutated"`
}

type coverage struct {
	Properties CoverageProperties

	linkCoverage bool
}

func (cov *coverage) props() []interface{} {
	return []interface{}{&cov.Properties}
}

func (cov *coverage) begin(ctx BaseModuleContext) {}

func (cov *coverage) deps(ctx BaseModuleContext, deps Deps) Deps {
	return deps
}

func (cov *coverage) flags(ctx ModuleContext, flags Flags) Flags {
	if !ctx.DeviceConfig().NativeCoverageEnabled() {
		return flags
	}

	if cov.Properties.CoverageEnabled {
		flags.Coverage = true
		flags.GlobalFlags = append(flags.GlobalFlags, "--coverage", "-O0")
		cov.linkCoverage = true
	}

	if !cov.linkCoverage {
		if ctx.static() && !ctx.staticBinary() {

			ctx.VisitDirectDepsWithTag(wholeStaticDepTag, func(m android.Module) {
				if cc, ok := m.(*Module); ok && cc.coverage != nil {
					if cc.coverage.linkCoverage {
						cov.linkCoverage = true
					}
				}
			})
		} else {

			ctx.VisitDirectDeps(func(m android.Module) {
				cc, ok := m.(*Module)
				if !ok || cc.coverage == nil {
					return
				}

				if static, ok := cc.linker.(libraryInterface); !ok || !static.static() {
					return
				}

				if cc.coverage.linkCoverage {
					cov.linkCoverage = true
				}
			})
		}
	}

	if cov.linkCoverage {
		flags.LdFlags = append(flags.LdFlags, "--coverage")
	}

	return flags
}

func coverageLinkingMutator(mctx android.BottomUpMutatorContext) {
	if c, ok := mctx.Module().(*Module); ok && c.coverage != nil {
		var enabled bool

		if !mctx.DeviceConfig().NativeCoverageEnabled() {

		} else if mctx.Host() {

		} else if c.coverage.Properties.Native_coverage != nil {
			enabled = *c.coverage.Properties.Native_coverage
		} else {
			enabled = mctx.DeviceConfig().CoverageEnabledForPath(mctx.ModuleDir())
		}

		if enabled {

			m := mctx.CreateLocalVariations("cov")
			m[0].(*Module).coverage.Properties.CoverageEnabled = true
		}
	}
}
