package xml

import (
	"android/soong/android"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

// prebuilt_etc_xml installs an xml file under <partition>/etc/<subdir>.
// It also optionally validates the xml file against the schema.

var (
	pctx = android.NewPackageContext("android/soong/xml")

	xmllintDtd = pctx.AndroidStaticRule("xmllint-dtd",
		blueprint.RuleParams{
			Command:     `$XmlLintCmd --dtdvalid $dtd $in > /dev/null && touch -a $out`,
			CommandDeps: []string{"$XmlLintCmd"},
			Restat:      true,
		},
		"dtd")

	xmllintXsd = pctx.AndroidStaticRule("xmllint-xsd",
		blueprint.RuleParams{
			Command:     `$XmlLintCmd --schema $xsd $in > /dev/null && touch -a $out`,
			CommandDeps: []string{"$XmlLintCmd"},
			Restat:      true,
		},
		"xsd")

	xmllintMinimal = pctx.AndroidStaticRule("xmllint-minimal",
		blueprint.RuleParams{
			Command:     `$XmlLintCmd $in > /dev/null && touch -a $out`,
			CommandDeps: []string{"$XmlLintCmd"},
			Restat:      true,
		})
)

func init() {
	android.RegisterModuleType("prebuilt_etc_xml", PrebuiltEtcXmlFactory)
	pctx.HostBinToolVariable("XmlLintCmd", "xmllint")
}

type prebuiltEtcXmlProperties struct {
	// Optional DTD that will be used to validate the xml file.
	Schema *string
}

type prebuiltEtcXml struct {
	android.PrebuiltEtc

	properties prebuiltEtcXmlProperties
}

func (p *prebuiltEtcXml) timestampFilePath(ctx android.ModuleContext) android.WritablePath {
	return android.PathForModuleOut(ctx, p.PrebuiltEtc.SourceFilePath(ctx).Base()+"-timestamp")
}

func (p *prebuiltEtcXml) DepsMutator(ctx android.BottomUpMutatorContext) {
	p.PrebuiltEtc.DepsMutator(ctx)

	// To support ":modulename" in schema
	android.ExtractSourceDeps(ctx, p.properties.Schema)
}

func (p *prebuiltEtcXml) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	p.PrebuiltEtc.GenerateAndroidBuildActions(ctx)

	if p.properties.Schema != nil {
		schema := ctx.ExpandSource(proptools.String(p.properties.Schema), "schema")

		switch schema.Ext() {
		case ".dtd":
			ctx.Build(pctx, android.BuildParams{
				Rule:        xmllintDtd,
				Description: "xmllint-dtd",
				Input:       p.PrebuiltEtc.SourceFilePath(ctx),
				Output:      p.timestampFilePath(ctx),
				Implicit:    schema,
				Args: map[string]string{
					"dtd": schema.String(),
				},
			})
			break
		case ".xsd":
			ctx.Build(pctx, android.BuildParams{
				Rule:        xmllintXsd,
				Description: "xmllint-xsd",
				Input:       p.PrebuiltEtc.SourceFilePath(ctx),
				Output:      p.timestampFilePath(ctx),
				Implicit:    schema,
				Args: map[string]string{
					"xsd": schema.String(),
				},
			})
			break
		default:
			ctx.PropertyErrorf("schema", "not supported extension: %q", schema.Ext())
		}
	} else {
		// when schema is not specified, just check if the xml is well-formed
		ctx.Build(pctx, android.BuildParams{
			Rule:        xmllintMinimal,
			Description: "xmllint-minimal",
			Input:       p.PrebuiltEtc.SourceFilePath(ctx),
			Output:      p.timestampFilePath(ctx),
		})
	}

	p.SetAdditionalDependencies([]android.Path{p.timestampFilePath(ctx)})
}

func PrebuiltEtcXmlFactory() android.Module {
	module := &prebuiltEtcXml{}
	module.AddProperties(&module.properties)

	android.InitPrebuiltEtcModule(&module.PrebuiltEtc)
	// This module is device-only
	android.InitAndroidArchModule(module, android.HostAndDeviceSupported, android.MultilibCommon)
	return module
}
