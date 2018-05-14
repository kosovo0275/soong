package phony

import (
	"fmt"
	"io"
	"strings"

	"android/soong/android"
)

func init() {
	android.RegisterModuleType("phony", phonyFactory)
}

type phony struct {
	android.ModuleBase
	requiredModuleNames []string
}

func phonyFactory() android.Module {
	module := &phony{}

	android.InitAndroidModule(module)
	return module
}

func (p *phony) DepsMutator(ctx android.BottomUpMutatorContext) {
}

func (p *phony) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	p.requiredModuleNames = ctx.RequiredModuleNames()
	if len(p.requiredModuleNames) == 0 {
		ctx.PropertyErrorf("required", "phony must not have empty required dependencies in order to be useful(and therefore permitted).")
	}
}

func (p *phony) AndroidMk() android.AndroidMkData {
	return android.AndroidMkData{
		Custom: func(w io.Writer, name, prefix, moduleDir string, data android.AndroidMkData) {
			fmt.Fprintln(w, "\ninclude $(CLEAR_VARS)")
			fmt.Fprintln(w, "LOCAL_PATH :=", moduleDir)
			fmt.Fprintln(w, "LOCAL_MODULE :=", name)
			fmt.Fprintln(w, "LOCAL_REQUIRED_MODULES := "+strings.Join(p.requiredModuleNames, " "))
			fmt.Fprintln(w, "include $(BUILD_PHONY_PACKAGE)")
		},
	}
}
