package cc

import (
	"github.com/google/blueprint"

	"android/soong/android"
)

func init() {
	pctx.StaticVariable("relocationPackerCmd", android.TermuxExecutable("relocation_packer"))
}

var relocationPackerRule = pctx.AndroidStaticRule("packRelocations",
	blueprint.RuleParams{
		Command:     "rm -f $out && cp $in $out && $relocationPackerCmd $out",
		CommandDeps: []string{"$relocationPackerCmd"},
	})

type RelocationPackerProperties struct {
	Pack_relocations *bool `android:"arch_variant"`

	PackingRelocations bool `blueprint:"mutated"`

	Use_clang_lld *bool
}

type relocationPacker struct {
	Properties RelocationPackerProperties
}

func (p *relocationPacker) useClangLld(ctx BaseModuleContext) bool {
	if p.Properties.Use_clang_lld != nil {
		return Bool(p.Properties.Use_clang_lld)
	}
	return ctx.Config().UseClangLld()
}

func (p *relocationPacker) packingInit(ctx BaseModuleContext) {
	enabled := true
	if ctx.Config().Getenv("DISABLE_RELOCATION_PACKER") == "true" {
		enabled = false
	}

	if p.useClangLld(ctx) {
		enabled = false
	}
	if ctx.useSdk() {
		enabled = false
	}
	if p.Properties.Pack_relocations != nil && *p.Properties.Pack_relocations == false {
		enabled = false
	}

	p.Properties.PackingRelocations = enabled
}

func (p *relocationPacker) needsPacking(ctx ModuleContext) bool {
	if ctx.Config().EmbeddedInMake() {
		return false
	}
	return p.Properties.PackingRelocations
}

func (p *relocationPacker) pack(ctx ModuleContext, in, out android.ModuleOutPath, flags builderFlags) {
	ctx.Build(pctx, android.BuildParams{
		Rule:        relocationPackerRule,
		Description: "pack relocations",
		Output:      out,
		Input:       in,
	})
}
