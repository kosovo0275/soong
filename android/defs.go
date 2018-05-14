package android

import (
	"github.com/google/blueprint"
	_ "github.com/google/blueprint/bootstrap"
)

var (
	pctx = NewPackageContext("android/soong/common")

	cpPreserveSymlinks = pctx.VariableConfigMethod("cpPreserveSymlinks",
		Config.CpPreserveSymlinksFlags)

	Phony = pctx.AndroidStaticRule("Phony",
		blueprint.RuleParams{
			Command:     "# phony $out",
			Description: "phony $out",
		})

	GeneratedFile = pctx.AndroidStaticRule("GeneratedFile",
		blueprint.RuleParams{
			Command:     "# generated $out",
			Description: "generated $out",
			Generator:   true,
		})

	Cp = pctx.AndroidStaticRule("Cp",
		blueprint.RuleParams{
			Command:     "rm -f $out && cp $cpPreserveSymlinks $cpFlags $in $out",
			Description: "cp $out",
		},
		"cpFlags")

	CpExecutable = pctx.AndroidStaticRule("CpExecutable",
		blueprint.RuleParams{
			Command:     "rm -f $out && cp $cpPreserveSymlinks $cpFlags $in $out && chmod +x $out",
			Description: "cp $out",
		},
		"cpFlags")

	Touch = pctx.AndroidStaticRule("Touch",
		blueprint.RuleParams{
			Command:     "touch $out",
			Description: "touch $out",
		})

	Symlink = pctx.AndroidStaticRule("Symlink",
		blueprint.RuleParams{
			Command:     "ln -f -s $fromPath $out",
			Description: "symlink $out",
		},
		"fromPath")

	ErrorRule = pctx.AndroidStaticRule("Error",
		blueprint.RuleParams{
			Command:     `echo "$error" && false`,
			Description: "error building $out",
		},
		"error")

	Cat = pctx.AndroidStaticRule("Cat",
		blueprint.RuleParams{
			Command:     "cat $in > $out",
			Description: "concatenate licenses $out",
		})

	WriteFile = pctx.AndroidStaticRule("WriteFile",
		blueprint.RuleParams{
			Command:     "/data/data/com.termux/files/usr/bin/bash -c 'echo -e $$0 > $out' '$content'",
			Description: "writing file $out",
		},
		"content")

	localPool = blueprint.NewBuiltinPool("local_pool")
)

func init() {
	pctx.Import("github.com/google/blueprint/bootstrap")
}
