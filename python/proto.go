package python

import (
	"android/soong/android"
	"strings"

	"github.com/google/blueprint"
)

func init() {
	pctx.HostBinToolVariable("protocCmd", "aprotoc")
}

var (
	proto = pctx.AndroidStaticRule("protoc",
		blueprint.RuleParams{
			Command: `rm -rf $out.tmp && mkdir -p $out.tmp && ` +
				`$protocCmd --python_out=$out.tmp -I $protoBase $protoFlags $in && ` +
				`$parCmd -o $out -P $pkgPath -C $out.tmp -D $out.tmp && rm -rf $out.tmp`,
			CommandDeps: []string{
				"$protocCmd",
				"$parCmd",
			},
		}, "protoBase", "protoFlags", "pkgPath")
)

func genProto(ctx android.ModuleContext, p *android.ProtoProperties,
	protoFile android.Path, protoFlags []string, pkgPath string) android.Path {
	srcJarFile := android.PathForModuleGen(ctx, protoFile.Base()+".srcszip")

	protoRoot := android.ProtoCanonicalPathFromRoot(ctx, p)

	var protoBase string
	if protoRoot {
		protoBase = "."
	} else {
		protoBase = strings.TrimSuffix(protoFile.String(), protoFile.Rel())
	}

	ctx.Build(pctx, android.BuildParams{
		Rule:        proto,
		Description: "protoc " + protoFile.Rel(),
		Output:      srcJarFile,
		Input:       protoFile,
		Args: map[string]string{
			"protoBase":  protoBase,
			"protoFlags": strings.Join(protoFlags, " "),
			"pkgPath":    pkgPath,
		},
	})

	return srcJarFile
}
