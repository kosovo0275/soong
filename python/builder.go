package python

// This file contains Ninja build actions for building Python program.

import (
	"strings"

	"android/soong/android"

	"github.com/google/blueprint"
	_ "github.com/google/blueprint/bootstrap"
)

var (
	pctx = android.NewPackageContext("android/soong/python")

	zip = pctx.AndroidStaticRule("zip",
		blueprint.RuleParams{
			Command:     `$parCmd -o $out $args`,
			CommandDeps: []string{"$parCmd"},
		},
		"args")

	combineZip = pctx.AndroidStaticRule("combineZip",
		blueprint.RuleParams{
			Command:     `$mergeParCmd $out $in`,
			CommandDeps: []string{"$mergeParCmd"},
		},
	)

	hostPar = pctx.AndroidStaticRule("hostPar",
		blueprint.RuleParams{
			Command: `sed -e 's/%interpreter%/$interp/g' -e 's/%main%/$main/g' $template > $stub && ` +
				`$mergeParCmd -p -pm $stub $mergedZip $srcsZips && echo '#!/data/data/com.termux/files/usr/bin/python2.7' | cat - $mergedZip > $out && ` +
				`chmod +x $out && (rm -f $stub; rm -f $mergedZip)`,
			CommandDeps: []string{"$mergeParCmd"},
		},
		"interp", "main", "template", "stub", "mergedZip", "srcsZips")

	embeddedPar = pctx.AndroidStaticRule("embeddedPar",
		blueprint.RuleParams{
			Command: `echo '$main' > $entryPoint &&` +
				`$mergeParCmd -p -e $entryPoint $mergedZip $srcsZips && cat $launcher | cat - $mergedZip > $out && ` +
				`chmod +x $out && (rm -f $entryPoint; rm -f $mergedZip)`,
			CommandDeps: []string{"$mergeParCmd"},
		},
		"main", "entryPoint", "mergedZip", "srcsZips", "launcher")
)

func init() {
	pctx.Import("github.com/google/blueprint/bootstrap")
	pctx.Import("android/soong/common")

	pctx.HostBinToolVariable("parCmd", "soong_zip")
	pctx.HostBinToolVariable("mergeParCmd", "merge_zips")
}

func registerBuildActionForParFile(ctx android.ModuleContext, embeddedLauncher bool,
	launcherPath android.Path, interpreter, main, binName string,
	srcsZips android.Paths) android.Path {

	// .intermediate output path for merged zip file.
	mergedZip := android.PathForModuleOut(ctx, binName+".mergedzip")

	// .intermediate output path for bin executable.
	binFile := android.PathForModuleOut(ctx, binName)

	// implicit dependency for parFile build action.
	implicits := srcsZips

	if !embeddedLauncher {
		// the path of stub_template_host.txt from source tree.
		template := android.PathForSource(ctx, stubTemplateHost)
		implicits = append(implicits, template)

		// intermediate output path for __main__.py
		stub := android.PathForModuleOut(ctx, mainFileName).String()

		ctx.Build(pctx, android.BuildParams{
			Rule:        hostPar,
			Description: "host python archive",
			Output:      binFile,
			Implicits:   implicits,
			Args: map[string]string{
				"interp": strings.Replace(interpreter, "/", `\/`, -1),
				// we need remove "runfiles/" suffix since stub script starts
				// searching for main file in each sub-dir of "runfiles" directory tree.
				"main": strings.Replace(strings.TrimPrefix(main, runFiles+"/"),
					"/", `\/`, -1),
				"template":  template.String(),
				"stub":      stub,
				"mergedZip": mergedZip.String(),
				"srcsZips":  strings.Join(srcsZips.Strings(), " "),
			},
		})
	} else {
		// added launcherPath to the implicits Ninja dependencies.
		implicits = append(implicits, launcherPath)

		// .intermediate output path for entry_point.txt
		entryPoint := android.PathForModuleOut(ctx, entryPointFile).String()

		ctx.Build(pctx, android.BuildParams{
			Rule:        embeddedPar,
			Description: "embedded python archive",
			Output:      binFile,
			Implicits:   implicits,
			Args: map[string]string{
				"main":       main,
				"entryPoint": entryPoint,
				"mergedZip":  mergedZip.String(),
				"srcsZips":   strings.Join(srcsZips.Strings(), " "),
				"launcher":   launcherPath.String(),
			},
		})
	}

	return binFile
}
