package android

import (
	"fmt"
	"io"
)

func init() {
	RegisterModuleType("prebuilt_etc", PrebuiltEtcFactory)
}

type prebuiltEtcProperties struct {
	Src     *string `android:"arch_variant"`
	Sub_dir *string `android:"arch_variant"`
}

type PrebuiltEtc struct {
	ModuleBase
	properties             prebuiltEtcProperties
	sourceFilePath         Path
	installDirPath         OutputPath
	additionalDependencies *Paths
}

func (p *PrebuiltEtc) DepsMutator(ctx BottomUpMutatorContext) {
	if p.properties.Src == nil {
		ctx.PropertyErrorf("src", "missing prebuilt source file")
	}

	ExtractSourceDeps(ctx, p.properties.Src)
}

func (p *PrebuiltEtc) SourceFilePath(ctx ModuleContext) Path {
	return ctx.ExpandSource(String(p.properties.Src), "src")
}

func (p *PrebuiltEtc) SetAdditionalDependencies(paths Paths) {
	p.additionalDependencies = &paths
}

func (p *PrebuiltEtc) GenerateAndroidBuildActions(ctx ModuleContext) {
	p.sourceFilePath = ctx.ExpandSource(String(p.properties.Src), "src")
	p.installDirPath = PathForModuleInstall(ctx, "etc", String(p.properties.Sub_dir))
}

func (p *PrebuiltEtc) AndroidMk() AndroidMkData {
	return AndroidMkData{
		Custom: func(w io.Writer, name, prefix, moduleDir string, data AndroidMkData) {
			fmt.Fprintln(w, "\ninclude $(CLEAR_VARS)")
			fmt.Fprintln(w, "LOCAL_PATH :=", moduleDir)
			fmt.Fprintln(w, "LOCAL_MODULE :=", name)
			fmt.Fprintln(w, "LOCAL_MODULE_CLASS := ETC")
			fmt.Fprintln(w, "LOCAL_MODULE_TAGS := optional")
			fmt.Fprintln(w, "LOCAL_PREBUILT_MODULE_FILE :=", p.sourceFilePath.String())
			fmt.Fprintln(w, "LOCAL_MODULE_PATH :=", "$(OUT_DIR)/"+p.installDirPath.RelPathString())
			if p.additionalDependencies != nil {
				fmt.Fprint(w, "LOCAL_ADDITIONAL_DEPENDENCIES :=")
				for _, path := range *p.additionalDependencies {
					fmt.Fprint(w, " "+path.String())
				}
				fmt.Fprintln(w, "")
			}
			fmt.Fprintln(w, "include $(BUILD_PREBUILT)")
		},
	}
}

func InitPrebuiltEtcModule(p *PrebuiltEtc) {
	p.AddProperties(&p.properties)
}

func PrebuiltEtcFactory() Module {
	module := &PrebuiltEtc{}
	InitPrebuiltEtcModule(module)
	InitAndroidArchModule(module, HostAndDeviceSupported, MultilibCommon)
	return module
}
