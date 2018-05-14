package android

import (
	"github.com/google/blueprint"
)

type LoadHookContext interface {
	BaseContext
	AppendProperties(...interface{})
	PrependProperties(...interface{})
	CreateModule(blueprint.ModuleFactory, ...interface{})
}

type ArchHookContext interface {
	BaseContext
	AppendProperties(...interface{})
	PrependProperties(...interface{})
}

func AddLoadHook(m blueprint.Module, hook func(LoadHookContext)) {
	h := &m.(Module).base().hooks
	h.load = append(h.load, hook)
}

func AddArchHook(m blueprint.Module, hook func(ArchHookContext)) {
	h := &m.(Module).base().hooks
	h.arch = append(h.arch, hook)
}

func (x *hooks) runLoadHooks(ctx LoadHookContext, m *ModuleBase) {
	if len(x.load) > 0 {
		for _, x := range x.load {
			x(ctx)
			if ctx.Failed() {
				return
			}
		}
	}
}

func (x *hooks) runArchHooks(ctx ArchHookContext, m *ModuleBase) {
	if len(x.arch) > 0 {
		for _, x := range x.arch {
			x(ctx)
			if ctx.Failed() {
				return
			}
		}
	}
}

type InstallHookContext interface {
	ModuleContext
	Path() OutputPath
	Symlink() bool
}

func AddInstallHook(m blueprint.Module, hook func(InstallHookContext)) {
	h := &m.(Module).base().hooks
	h.install = append(h.install, hook)
}

type installHookContext struct {
	ModuleContext
	path    OutputPath
	symlink bool
}

func (x *installHookContext) Path() OutputPath {
	return x.path
}

func (x *installHookContext) Symlink() bool {
	return x.symlink
}

func (x *hooks) runInstallHooks(ctx ModuleContext, path OutputPath, symlink bool) {
	if len(x.install) > 0 {
		mctx := &installHookContext{
			ModuleContext: ctx,
			path:          path,
			symlink:       symlink,
		}
		for _, x := range x.install {
			x(mctx)
			if mctx.Failed() {
				return
			}
		}
	}
}

type hooks struct {
	load    []func(LoadHookContext)
	arch    []func(ArchHookContext)
	install []func(InstallHookContext)
}

func loadHookMutator(ctx TopDownMutatorContext) {
	if m, ok := ctx.Module().(Module); ok {
		var loadHookCtx LoadHookContext = ctx.(*androidTopDownMutatorContext)
		m.base().hooks.runLoadHooks(loadHookCtx, m.base())
	}
}

func archHookMutator(ctx TopDownMutatorContext) {
	if m, ok := ctx.Module().(Module); ok {
		var archHookCtx ArchHookContext = ctx.(*androidTopDownMutatorContext)
		m.base().hooks.runArchHooks(archHookCtx, m.base())
	}
}
