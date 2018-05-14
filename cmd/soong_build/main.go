package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/blueprint/bootstrap"

	"android/soong/android"
)

var (
	docFile string
)

func init() {
	flag.StringVar(&docFile, "soong_docs", "", "build documentation file to output")
}

func newNameResolver(config android.Config) *android.NameResolver {
	namespacePathsToExport := make(map[string]bool)

	for _, namespaceName := range config.ExportedNamespaces() {
		namespacePathsToExport[namespaceName] = true
	}

	namespacePathsToExport["."] = true // always export the root namespace

	exportFilter := func(namespace *android.Namespace) bool {
		return namespacePathsToExport[namespace.Path]
	}

	return android.NewNameResolver(exportFilter)
}

func main() {
	flag.Parse()

	// The top-level Blueprints file is passed as the first argument.
	srcDir := filepath.Dir(flag.Arg(0))

	ctx := android.NewContext()
	ctx.Register()

	configuration, err := android.NewConfig(srcDir, bootstrap.BuildDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}

	if docFile != "" {
		configuration.SetStopBefore(bootstrap.StopBeforePrepareBuildActions)
	}

	ctx.SetNameInterface(newNameResolver(configuration))

	ctx.SetAllowMissingDependencies(configuration.AllowMissingDependencies())

	bootstrap.Main(ctx.Context, configuration, configuration.ConfigFileName, configuration.ProductVariablesFileName)

	if docFile != "" {
		writeDocs(ctx, docFile)
	}
}
