package android

import (
	"io"
	"strings"
	"text/template"
)

func init() {
	RegisterModuleType("filegroup", FileGroupFactory)
}

type fileGroupProperties struct {
	Srcs               []string
	Exclude_srcs       []string
	Path               *string
	Export_to_make_var *string
}

type fileGroup struct {
	ModuleBase
	properties fileGroupProperties
	srcs       Paths
}

var _ SourceFileProducer = (*fileGroup)(nil)

func FileGroupFactory() Module {
	module := &fileGroup{}
	module.AddProperties(&module.properties)
	InitAndroidModule(module)
	return module
}

func (fg *fileGroup) DepsMutator(ctx BottomUpMutatorContext) {
	ExtractSourcesDeps(ctx, fg.properties.Srcs)
	ExtractSourcesDeps(ctx, fg.properties.Exclude_srcs)
}

func (fg *fileGroup) GenerateAndroidBuildActions(ctx ModuleContext) {
	fg.srcs = ctx.ExpandSourcesSubDir(fg.properties.Srcs, fg.properties.Exclude_srcs, String(fg.properties.Path))
}

func (fg *fileGroup) Srcs() Paths {
	return append(Paths{}, fg.srcs...)
}

var androidMkTemplate = template.Must(template.New("filegroup").Parse(`
ifdef {{.makeVar}}
  $(error variable {{.makeVar}} set by soong module is already set in make)
endif
{{.makeVar}} := {{.value}}
.KATI_READONLY := {{.makeVar}}
`))

func (fg *fileGroup) AndroidMk() AndroidMkData {
	return AndroidMkData{
		Custom: func(w io.Writer, name, prefix, moduleDir string, data AndroidMkData) {
			if makeVar := String(fg.properties.Export_to_make_var); makeVar != "" {
				androidMkTemplate.Execute(w, map[string]string{
					"makeVar": makeVar,
					"value":   strings.Join(fg.srcs.Strings(), " "),
				})
			}
		},
	}
}
