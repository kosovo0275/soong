package cc

import (
	"android/soong/android"
)

type StripProperties struct {
	Strip struct {
		None         *bool
		Keep_symbols *bool
	}
}

type stripper struct {
	StripProperties StripProperties
}

func (stripper *stripper) needsStrip(ctx ModuleContext) bool {
	return !ctx.Config().EmbeddedInMake() && !Bool(stripper.StripProperties.Strip.None)
}

func (stripper *stripper) strip(ctx ModuleContext, in, out android.ModuleOutPath, flags builderFlags) {
	flags.stripKeepSymbols = Bool(stripper.StripProperties.Strip.Keep_symbols)
	flags.stripAddGnuDebuglink = true
	TransformStrip(ctx, in, out, flags)
}
