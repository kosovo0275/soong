package android

import (
	"fmt"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/pathtools"
)

type PackageContext struct {
	blueprint.PackageContext
}

func NewPackageContext(pkgPath string) PackageContext {
	return PackageContext{blueprint.NewPackageContext(pkgPath)}
}

type configErrorWrapper struct {
	pctx   PackageContext
	config Config
	errors []error
}

var _ PathContext = &configErrorWrapper{}
var _ errorfContext = &configErrorWrapper{}
var _ PackageVarContext = &configErrorWrapper{}
var _ PackagePoolContext = &configErrorWrapper{}
var _ PackageRuleContext = &configErrorWrapper{}

func (e *configErrorWrapper) Config() Config {
	return e.config
}
func (e *configErrorWrapper) Errorf(format string, args ...interface{}) {
	e.errors = append(e.errors, fmt.Errorf(format, args...))
}
func (e *configErrorWrapper) AddNinjaFileDeps(deps ...string) {
	e.pctx.AddNinjaFileDeps(deps...)
}

func (e *configErrorWrapper) Fs() pathtools.FileSystem {
	return nil
}

type PackageVarContext interface {
	PathContext
	errorfContext
}

type PackagePoolContext PackageVarContext
type PackageRuleContext PackageVarContext

func (p PackageContext) VariableFunc(name string,
	f func(PackageVarContext) string) blueprint.Variable {

	return p.PackageContext.VariableFunc(name, func(config interface{}) (string, error) {
		ctx := &configErrorWrapper{p, config.(Config), nil}
		ret := f(ctx)
		if len(ctx.errors) > 0 {
			return "", ctx.errors[0]
		}
		return ret, nil
	})
}

func (p PackageContext) PoolFunc(name string,
	f func(PackagePoolContext) blueprint.PoolParams) blueprint.Pool {

	return p.PackageContext.PoolFunc(name, func(config interface{}) (blueprint.PoolParams, error) {
		ctx := &configErrorWrapper{p, config.(Config), nil}
		params := f(ctx)
		if len(ctx.errors) > 0 {
			return params, ctx.errors[0]
		}
		return params, nil
	})
}

func (p PackageContext) RuleFunc(name string,
	f func(PackageRuleContext) blueprint.RuleParams, argNames ...string) blueprint.Rule {

	return p.PackageContext.RuleFunc(name, func(config interface{}) (blueprint.RuleParams, error) {
		ctx := &configErrorWrapper{p, config.(Config), nil}
		params := f(ctx)
		if len(ctx.errors) > 0 {
			return params, ctx.errors[0]
		}
		return params, nil
	}, argNames...)
}

func (p PackageContext) SourcePathVariable(name, path string) blueprint.Variable {
	return p.VariableFunc(name, func(ctx PackageVarContext) string {
		return safePathForSource(ctx, path).String()
	})
}

func (p PackageContext) SourcePathsVariable(name, separator string, paths ...string) blueprint.Variable {
	return p.VariableFunc(name, func(ctx PackageVarContext) string {
		var ret []string
		for _, path := range paths {
			p := safePathForSource(ctx, path)
			ret = append(ret, p.String())
		}
		return strings.Join(ret, separator)
	})
}

func (p PackageContext) SourcePathVariableWithEnvOverride(name, path, env string) blueprint.Variable {
	return p.VariableFunc(name, func(ctx PackageVarContext) string {
		p := safePathForSource(ctx, path)
		return ctx.Config().GetenvWithDefault(env, p.String())
	})
}

func (p PackageContext) HostBinToolVariable(name, path string) blueprint.Variable {
	return p.VariableFunc(name, func(ctx PackageVarContext) string {
		return p.HostBinToolPath(ctx, path).String()
	})
}

func (p PackageContext) HostBinToolPath(ctx PackageVarContext, path string) Path {
	return StringToPath("out/host", ctx.Config().PrebuiltOS(), "bin", path)
}

func (p PackageContext) HostJNIToolVariable(name, path string) blueprint.Variable {
	return p.VariableFunc(name, func(ctx PackageVarContext) string {
		return p.HostJNIToolPath(ctx, path).String()
	})
}

func (p PackageContext) HostJNIToolPath(ctx PackageVarContext, path string) Path {
	ext := ".so"
	return PathForOutput(ctx, "host", ctx.Config().PrebuiltOS(), "lib64", path+ext)
}

func (p PackageContext) HostJavaToolVariable(name, path string) blueprint.Variable {
	return p.VariableFunc(name, func(ctx PackageVarContext) string {
		return p.HostJavaToolPath(ctx, path).String()
	})
}

func (p PackageContext) HostJavaToolPath(ctx PackageVarContext, path string) Path {
	return PathForOutput(ctx, "host", ctx.Config().PrebuiltOS(), "framework", path)
}

func (p PackageContext) IntermediatesPathVariable(name, path string) blueprint.Variable {
	return p.VariableFunc(name, func(ctx PackageVarContext) string {
		return PathForIntermediates(ctx, path).String()
	})
}

func (p PackageContext) PrefixedExistentPathsForSourcesVariable(
	name, prefix string, paths []string) blueprint.Variable {

	return p.VariableFunc(name, func(ctx PackageVarContext) string {
		paths := ExistentPathsForSources(ctx, paths)
		return JoinWithPrefix(paths.Strings(), prefix)
	})
}

func (p PackageContext) AndroidStaticRule(name string, params blueprint.RuleParams,
	argNames ...string) blueprint.Rule {
	return p.AndroidRuleFunc(name, func(PackageRuleContext) blueprint.RuleParams {
		return params
	}, argNames...)
}

func (p PackageContext) AndroidGomaStaticRule(name string, params blueprint.RuleParams,
	argNames ...string) blueprint.Rule {
	return p.StaticRule(name, params, argNames...)
}

func (p PackageContext) AndroidRuleFunc(name string,
	f func(PackageRuleContext) blueprint.RuleParams, argNames ...string) blueprint.Rule {
	return p.RuleFunc(name, func(ctx PackageRuleContext) blueprint.RuleParams {
		params := f(ctx)
		if ctx.Config().UseGoma() && params.Pool == nil {
			params.Pool = localPool
		}
		return params
	}, argNames...)
}
