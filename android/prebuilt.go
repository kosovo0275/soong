package android

import (
	"fmt"

	"github.com/google/blueprint"
)

type prebuiltDependencyTag struct {
	blueprint.BaseDependencyTag
}

var prebuiltDepTag prebuiltDependencyTag

type PrebuiltProperties struct {
	Prefer       *bool `android:"arch_variant"`
	SourceExists bool  `blueprint:"mutated"`
	UsePrebuilt  bool  `blueprint:"mutated"`
}

type Prebuilt struct {
	properties PrebuiltProperties
	module     Module
	srcs       *[]string
}

func (p *Prebuilt) Name(name string) string {
	return "prebuilt_" + name
}

func (p *Prebuilt) SingleSourcePath(ctx ModuleContext) Path {
	if len(*p.srcs) == 0 {
		ctx.PropertyErrorf("srcs", "missing prebuilt source file")
		return nil
	}

	if len(*p.srcs) > 1 {
		ctx.PropertyErrorf("srcs", "multiple prebuilt source files")
		return nil
	}

	return ctx.ExpandSource((*p.srcs)[0], "")
}

func InitPrebuiltModule(module PrebuiltInterface, srcs *[]string) {
	p := module.Prebuilt()
	module.AddProperties(&p.properties)
	p.srcs = srcs
}

type PrebuiltInterface interface {
	Module
	Prebuilt() *Prebuilt
}

func RegisterPrebuiltsPreArchMutators(ctx RegisterMutatorsContext) {
	ctx.BottomUp("prebuilts", prebuiltMutator).Parallel()
}

func RegisterPrebuiltsPostDepsMutators(ctx RegisterMutatorsContext) {
	ctx.TopDown("prebuilt_select", PrebuiltSelectModuleMutator).Parallel()
	ctx.BottomUp("prebuilt_postdeps", PrebuiltPostDepsMutator).Parallel()
}

func prebuiltMutator(ctx BottomUpMutatorContext) {
	if m, ok := ctx.Module().(PrebuiltInterface); ok && m.Prebuilt() != nil {
		p := m.Prebuilt()
		name := m.base().BaseModuleName()
		if ctx.OtherModuleExists(name) {
			ctx.AddReverseDependency(ctx.Module(), prebuiltDepTag, name)
			p.properties.SourceExists = true
		} else {
			ctx.Rename(name)
		}
	}
}

func PrebuiltSelectModuleMutator(ctx TopDownMutatorContext) {
	if m, ok := ctx.Module().(PrebuiltInterface); ok && m.Prebuilt() != nil {
		p := m.Prebuilt()
		if p.srcs == nil {
			panic(fmt.Errorf("prebuilt module did not have InitPrebuiltModule called on it"))
		}
		if !p.properties.SourceExists {
			p.properties.UsePrebuilt = p.usePrebuilt(ctx, nil)
		}
	} else if s, ok := ctx.Module().(Module); ok {
		ctx.VisitDirectDepsWithTag(prebuiltDepTag, func(m Module) {
			p := m.(PrebuiltInterface).Prebuilt()
			if p.usePrebuilt(ctx, s) {
				p.properties.UsePrebuilt = true
				s.SkipInstall()
			}
		})
	}
}

func PrebuiltPostDepsMutator(ctx BottomUpMutatorContext) {
	if m, ok := ctx.Module().(PrebuiltInterface); ok && m.Prebuilt() != nil {
		p := m.Prebuilt()
		name := m.base().BaseModuleName()
		if p.properties.UsePrebuilt {
			if p.properties.SourceExists {
				ctx.ReplaceDependencies(name)
			}
		} else {
			m.SkipInstall()
		}
		if len(*p.srcs) > 0 {
			ExtractSourceDeps(ctx, &(*p.srcs)[0])
		}
	}
}

func (p *Prebuilt) usePrebuilt(ctx TopDownMutatorContext, source Module) bool {
	if len(*p.srcs) == 0 {
		return false
	}

	if Bool(p.properties.Prefer) {
		return true
	}

	return source == nil || !source.Enabled()
}
