package java

import (
	"fmt"
	"path/filepath"
	"strings"

	"android/soong/android"
)

var resourceExcludes = []string{
	"**/*.java",
	"**/package.html",
	"**/overview.html",
	"**/.*.swp",
	"**/.DS_Store",
	"**/*~",
}

func ResourceDirsToJarArgs(ctx android.ModuleContext,
	resourceDirs, excludeResourceDirs []string) (args []string, deps android.Paths) {
	var excludeDirs []string
	var excludeFiles []string

	for _, exclude := range excludeResourceDirs {
		dirs := ctx.Glob(android.PathForModuleSrc(ctx).Join(ctx, exclude).String(), nil)
		for _, dir := range dirs {
			excludeDirs = append(excludeDirs, dir.String())
			excludeFiles = append(excludeFiles, dir.(android.ModuleSrcPath).Join(ctx, "**/*").String())
		}
	}

	excludeFiles = append(excludeFiles, resourceExcludes...)

	for _, resourceDir := range resourceDirs {
		// resourceDir may be a glob, resolve it first
		dirs := ctx.Glob(android.PathForModuleSrc(ctx).Join(ctx, resourceDir).String(), excludeDirs)
		for _, dir := range dirs {
			files := ctx.GlobFiles(filepath.Join(dir.String(), "**/*"), excludeFiles)

			deps = append(deps, files...)

			if len(files) > 0 {
				args = append(args, "-C", dir.String())

				for _, f := range files {
					path := f.String()
					if !strings.HasPrefix(path, dir.String()) {
						panic(fmt.Errorf("path %q does not start with %q", path, dir))
					}
					args = append(args, "-f", path)
				}
			}
		}
	}

	return args, deps
}

// Convert java_resources properties to arguments to soong_zip -jar, ignoring common patterns
// that should not be treated as resources (including *.java).
func ResourceFilesToJarArgs(ctx android.ModuleContext,
	res, exclude []string) (args []string, deps android.Paths) {

	exclude = append([]string(nil), exclude...)
	exclude = append(exclude, resourceExcludes...)
	return resourceFilesToJarArgs(ctx, res, exclude)
}

// Convert java_resources properties to arguments to soong_zip -jar, keeping files that should
// normally not used as resources like *.java
func SourceFilesToJarArgs(ctx android.ModuleContext,
	res, exclude []string) (args []string, deps android.Paths) {

	return resourceFilesToJarArgs(ctx, res, exclude)
}

func resourceFilesToJarArgs(ctx android.ModuleContext,
	res, exclude []string) (args []string, deps android.Paths) {

	files := ctx.ExpandSources(res, exclude)

	lastDir := ""
	for i, f := range files {
		rel := f.Rel()
		path := f.String()
		if !strings.HasSuffix(path, rel) {
			panic(fmt.Errorf("path %q does not end with %q", path, rel))
		}
		dir := filepath.Clean(strings.TrimSuffix(path, rel))
		if i == 0 || dir != lastDir {
			args = append(args, "-C", dir)
		}
		args = append(args, "-f", path)
		lastDir = dir
	}

	return args, files
}
