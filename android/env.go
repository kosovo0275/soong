package android

import (
	"os"
	"strings"

	"android/soong/env"
)

var OriginalEnv map[string]string

func init() {
	OriginalEnv = make(map[string]string)
	for _, env := range os.Environ() {
		idx := strings.IndexRune(env, '=')
		if idx != -1 {
			OriginalEnv[env[:idx]] = env[idx+1:]
		}
	}
	//	os.Clearenv()
}

func EnvSingleton() Singleton {
	return &envSingleton{}
}

type envSingleton struct{}

func (c *envSingleton) GenerateBuildActions(ctx SingletonContext) {
	envDeps := ctx.Config().EnvDeps()

	envFile := PathForOutput(ctx, ".soong.environment")
	if ctx.Failed() {
		return
	}

	err := env.WriteEnvFile(envFile.String(), envDeps)
	if err != nil {
		ctx.Errorf(err.Error())
	}

	ctx.AddNinjaFileDeps(envFile.String())
}
