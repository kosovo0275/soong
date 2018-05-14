package cc

import (
	"android/soong/android"
)

type LTOProperties struct {
	Lto struct {
		Never *bool `android:"arch_variant"`
		Full  *bool `android:"arch_variant"`
		Thin  *bool `android:"arch_variant"`
	} `android:"arch_variant"`

	FullDep bool `blueprint:"mutated"`
	ThinDep bool `blueprint:"mutated"`

	Use_clang_lld *bool
}

type lto struct {
	Properties LTOProperties
}

func (lto *lto) props() []interface{} {
	return []interface{}{&lto.Properties}
}

func (lto *lto) begin(ctx BaseModuleContext) {
	if ctx.Config().IsEnvTrue("DISABLE_LTO") {
		lto.Properties.Lto.Never = boolPtr(true)
	}
}

func (lto *lto) deps(ctx BaseModuleContext, deps Deps) Deps {
	return deps
}

func (lto *lto) useClangLld(ctx BaseModuleContext) bool {
	if lto.Properties.Use_clang_lld != nil {
		return Bool(lto.Properties.Use_clang_lld)
	}
	return ctx.Config().UseClangLld()
}

func (lto *lto) flags(ctx BaseModuleContext, flags Flags) Flags {
	if lto.LTO() {
		var ltoFlag string
		if Bool(lto.Properties.Lto.Thin) {
			ltoFlag = "-flto=thin"

		} else {
			ltoFlag = "-flto"
		}

		flags.CFlags = append(flags.CFlags, ltoFlag)
		flags.LdFlags = append(flags.LdFlags, ltoFlag)

		if ctx.Config().IsEnvTrue("USE_THINLTO_CACHE") && Bool(lto.Properties.Lto.Thin) && !lto.useClangLld(ctx) {
			cacheDirFormat := "-Wl,-plugin-opt,cache-dir="
			cacheDir := android.PathForOutput(ctx, "thinlto-cache").String()
			flags.LdFlags = append(flags.LdFlags, cacheDirFormat+cacheDir)

			cachePolicyFormat := "-Wl,-plugin-opt,cache-policy="
			policy := "cache_size=10%:cache_size_bytes=10g"
			flags.LdFlags = append(flags.LdFlags, cachePolicyFormat+policy)
		}

		flags.ArGoldPlugin = true

		if !ctx.isPgoCompile() && !lto.useClangLld(ctx) {
			flags.LdFlags = append(flags.LdFlags, "-Wl,-plugin-opt,-inline-threshold=0")
			flags.LdFlags = append(flags.LdFlags, "-Wl,-plugin-opt,-unroll-threshold=0")
		}
	}
	return flags
}

func (lto *lto) LTO() bool {
	if lto == nil || lto.Disabled() {
		return false
	}

	full := Bool(lto.Properties.Lto.Full)
	thin := Bool(lto.Properties.Lto.Thin)
	return full || thin
}

func (lto *lto) Disabled() bool {
	return lto.Properties.Lto.Never != nil && *lto.Properties.Lto.Never
}

func ltoDepsMutator(mctx android.TopDownMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok && m.lto.LTO() {
		full := Bool(m.lto.Properties.Lto.Full)
		thin := Bool(m.lto.Properties.Lto.Thin)
		if full && thin {
			mctx.PropertyErrorf("LTO", "FullLTO and ThinLTO are mutually exclusive")
		}

		mctx.WalkDeps(func(dep android.Module, parent android.Module) bool {
			tag := mctx.OtherModuleDependencyTag(dep)
			switch tag {
			case staticDepTag, staticExportDepTag, lateStaticDepTag, wholeStaticDepTag, objDepTag, reuseObjTag:
				if dep, ok := dep.(*Module); ok && dep.lto != nil &&
					!dep.lto.Disabled() {
					if full && !Bool(dep.lto.Properties.Lto.Full) {
						dep.lto.Properties.FullDep = true
					}
					if thin && !Bool(dep.lto.Properties.Lto.Thin) {
						dep.lto.Properties.ThinDep = true
					}
				}

				return true
			}

			return false
		})
	}
}

func ltoMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok && m.lto != nil {
		variationNames := []string{""}
		if m.lto.Properties.FullDep && !Bool(m.lto.Properties.Lto.Full) {
			variationNames = append(variationNames, "lto-full")
		}
		if m.lto.Properties.ThinDep && !Bool(m.lto.Properties.Lto.Thin) {
			variationNames = append(variationNames, "lto-thin")
		}

		if Bool(m.lto.Properties.Lto.Full) {
			mctx.SetDependencyVariation("lto-full")
		}
		if Bool(m.lto.Properties.Lto.Thin) {
			mctx.SetDependencyVariation("lto-thin")
		}

		if len(variationNames) > 1 {
			modules := mctx.CreateVariations(variationNames...)
			for i, name := range variationNames {
				variation := modules[i].(*Module)
				if name == "" {
					continue
				}

				if name == "lto-full" {
					variation.lto.Properties.Lto.Full = boolPtr(true)
					variation.lto.Properties.Lto.Thin = boolPtr(false)
				}
				if name == "lto-thin" {
					variation.lto.Properties.Lto.Full = boolPtr(false)
					variation.lto.Properties.Lto.Thin = boolPtr(true)
				}
				variation.Properties.PreventInstall = true
				variation.Properties.HideFromMake = true
				variation.lto.Properties.FullDep = false
				variation.lto.Properties.ThinDep = false
			}
		}
	}
}
