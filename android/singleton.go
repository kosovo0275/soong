package android

import (
	"github.com/google/blueprint"
	"github.com/google/blueprint/pathtools"
)

type SingletonContext interface {
	Config() Config
	ModuleName(module blueprint.Module) string
	ModuleDir(module blueprint.Module) string
	ModuleSubDir(module blueprint.Module) string
	ModuleType(module blueprint.Module) string
	BlueprintFile(module blueprint.Module) string
	ModuleErrorf(module blueprint.Module, format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Failed() bool
	Variable(pctx PackageContext, name, value string)
	Rule(pctx PackageContext, name string, params blueprint.RuleParams, argNames ...string) blueprint.Rule
	Build(pctx PackageContext, params BuildParams)
	RequireNinjaVersion(major, minor, micro int)
	SetNinjaBuildDir(pctx PackageContext, value string)
	Eval(pctx PackageContext, ninjaStr string) (string, error)
	VisitAllModules(visit func(Module))
	VisitAllModulesIf(pred func(Module) bool, visit func(Module))
	VisitDepsDepthFirst(module Module, visit func(Module))
	VisitDepsDepthFirstIf(module Module, pred func(Module) bool, visit func(Module))
	VisitAllModuleVariants(module Module, visit func(Module))
	PrimaryModule(module Module) Module
	FinalModule(module Module) Module
	AddNinjaFileDeps(deps ...string)
	GlobWithDeps(pattern string, excludes []string) ([]string, error)
	Fs() pathtools.FileSystem
}

type singletonAdaptor struct {
	Singleton
}

func (s singletonAdaptor) GenerateBuildActions(ctx blueprint.SingletonContext) {
	s.Singleton.GenerateBuildActions(singletonContextAdaptor{ctx})
}

type Singleton interface {
	GenerateBuildActions(SingletonContext)
}

type singletonContextAdaptor struct {
	blueprint.SingletonContext
}

func (s singletonContextAdaptor) Config() Config {
	return s.SingletonContext.Config().(Config)
}

func (s singletonContextAdaptor) Variable(pctx PackageContext, name, value string) {
	s.SingletonContext.Variable(pctx.PackageContext, name, value)
}

func (s singletonContextAdaptor) Rule(pctx PackageContext, name string, params blueprint.RuleParams, argNames ...string) blueprint.Rule {
	return s.SingletonContext.Rule(pctx.PackageContext, name, params, argNames...)
}

func (s singletonContextAdaptor) Build(pctx PackageContext, params BuildParams) {
	bparams := convertBuildParams(params)
	s.SingletonContext.Build(pctx.PackageContext, bparams)

}

func (s singletonContextAdaptor) SetNinjaBuildDir(pctx PackageContext, value string) {
	s.SingletonContext.SetNinjaBuildDir(pctx.PackageContext, value)
}

func (s singletonContextAdaptor) Eval(pctx PackageContext, ninjaStr string) (string, error) {
	return s.SingletonContext.Eval(pctx.PackageContext, ninjaStr)
}

func visitAdaptor(visit func(Module)) func(blueprint.Module) {
	return func(module blueprint.Module) {
		if aModule, ok := module.(Module); ok {
			visit(aModule)
		}
	}
}

func predAdaptor(pred func(Module) bool) func(blueprint.Module) bool {
	return func(module blueprint.Module) bool {
		if aModule, ok := module.(Module); ok {
			return pred(aModule)
		} else {
			return false
		}
	}
}

func (s singletonContextAdaptor) VisitAllModules(visit func(Module)) {
	s.SingletonContext.VisitAllModules(visitAdaptor(visit))
}

func (s singletonContextAdaptor) VisitAllModulesIf(pred func(Module) bool, visit func(Module)) {
	s.SingletonContext.VisitAllModulesIf(predAdaptor(pred), visitAdaptor(visit))
}

func (s singletonContextAdaptor) VisitDepsDepthFirst(module Module, visit func(Module)) {
	s.SingletonContext.VisitDepsDepthFirst(module, visitAdaptor(visit))
}

func (s singletonContextAdaptor) VisitDepsDepthFirstIf(module Module, pred func(Module) bool, visit func(Module)) {
	s.SingletonContext.VisitDepsDepthFirstIf(module, predAdaptor(pred), visitAdaptor(visit))
}

func (s singletonContextAdaptor) VisitAllModuleVariants(module Module, visit func(Module)) {
	s.SingletonContext.VisitAllModuleVariants(module, visitAdaptor(visit))
}

func (s singletonContextAdaptor) PrimaryModule(module Module) Module {
	return s.SingletonContext.PrimaryModule(module).(Module)
}

func (s singletonContextAdaptor) FinalModule(module Module) Module {
	return s.SingletonContext.FinalModule(module).(Module)
}
