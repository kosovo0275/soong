package android

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/blueprint"
)

func init() {
	RegisterSingletonType("writedocs", DocsSingleton)
}

func DocsSingleton() Singleton {
	return &docsSingleton{}
}

type docsSingleton struct{}

func primaryBuilderPath(ctx SingletonContext) Path {
	primaryBuilder, err := filepath.Rel(ctx.Config().BuildDir(), os.Args[0])
	if err != nil {
		ctx.Errorf("path to primary builder %q is not in build dir %q",
			os.Args[0], ctx.Config().BuildDir())
	}

	return PathForOutput(ctx, primaryBuilder)
}

func (c *docsSingleton) GenerateBuildActions(ctx SingletonContext) {
	docsFile := PathForOutput(ctx, "docs", "soong_build.html")
	primaryBuilder := primaryBuilderPath(ctx)
	soongDocs := ctx.Rule(pctx, "soongDocs", blueprint.RuleParams{
		Command:     fmt.Sprintf("%s --soong_docs %s %s", primaryBuilder.String(), docsFile.String(), strings.Join(os.Args[1:], " ")),
		CommandDeps: []string{primaryBuilder.String()},
		Description: fmt.Sprintf("%s docs $out", primaryBuilder.Base()),
	})

	ctx.Build(pctx, BuildParams{
		Rule:   soongDocs,
		Output: docsFile,
	})

	ctx.Build(pctx, BuildParams{
		Rule:   blueprint.Phony,
		Output: PathForPhony(ctx, "soong_docs"),
		Input:  docsFile,
	})
}
