package cc

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"android/soong/android"
)

func init() {
	android.RegisterSingletonType("compdb_generator", compDBGeneratorSingleton)
}

func compDBGeneratorSingleton() android.Singleton {
	return &compdbGeneratorSingleton{}
}

type compdbGeneratorSingleton struct{}

const (
	compdbFilename                = "compile_commands.json"
	compdbOutputProjectsDirectory = "out/development/ide/compdb"

	envVariableGenerateCompdb          = "SOONG_GEN_COMPDB"
	envVariableGenerateCompdbDebugInfo = "SOONG_GEN_COMPDB_DEBUG"
	envVariableCompdbLink              = "SOONG_LINK_COMPDB_TO"
)

type compDbEntry struct {
	Directory string   `json:"directory"`
	Arguments []string `json:"arguments"`
	File      string   `json:"file"`
	Output    string   `json:"output,omitempty"`
}

func (c *compdbGeneratorSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	if !ctx.Config().IsEnvTrue(envVariableGenerateCompdb) {
		return
	}

	outputCompdbDebugInfo := ctx.Config().IsEnvTrue(envVariableGenerateCompdbDebugInfo)

	m := make(map[string]compDbEntry)
	ctx.VisitAllModules(func(module android.Module) {
		if ccModule, ok := module.(*Module); ok {
			if compiledModule, ok := ccModule.compiler.(CompiledInterface); ok {
				generateCompdbProject(compiledModule, ctx, ccModule, m)
			}
		}
	})

	dir := filepath.Join(getCompdbAndroidSrcRootDirectory(ctx), compdbOutputProjectsDirectory)
	os.MkdirAll(dir, 0777)
	compDBFile := filepath.Join(dir, compdbFilename)
	f, err := os.Create(compdbFilename)
	if err != nil {
		log.Fatalf("Could not create file %s: %s", filepath.Join(dir, compdbFilename), err)
	}
	defer f.Close()

	v := make([]compDbEntry, 0, len(m))

	for _, value := range m {
		v = append(v, value)
	}
	var dat []byte
	if outputCompdbDebugInfo {
		dat, err = json.MarshalIndent(v, "", " ")
	} else {
		dat, err = json.Marshal(v)
	}
	if err != nil {
		log.Fatalf("Failed to marshal: %s", err)
	}
	f.Write(dat)

	finalLinkPath := filepath.Join(ctx.Config().Getenv(envVariableCompdbLink), compdbFilename)
	if finalLinkPath != "" {
		os.Remove(finalLinkPath)
		if err := os.Symlink(compDBFile, finalLinkPath); err != nil {
			log.Fatalf("Unable to symlink %s to %s: %s", compDBFile, finalLinkPath, err)
		}
	}
}

func expandAllVars(ctx android.SingletonContext, args []string) []string {
	var out []string
	for _, arg := range args {
		if arg != "" {
			if val, err := evalAndSplitVariable(ctx, arg); err == nil {
				out = append(out, val...)
			} else {
				out = append(out, arg)
			}
		}
	}
	return out
}

func getArguments(src android.Path, ctx android.SingletonContext, ccModule *Module) []string {
	var args []string
	isCpp := false
	isAsm := false

	switch src.Ext() {
	case ".S", ".s", ".asm":
		isAsm = true
		isCpp = false
	case ".c":
		isAsm = false
		isCpp = false
	case ".cpp", ".cc", ".mm":
		isAsm = false
		isCpp = true
	default:
		log.Print("Unknown file extension " + src.Ext() + " on file " + src.String())
		isAsm = true
		isCpp = false
	}

	args = append(args, android.TermuxExecutable("true"))
	args = append(args, expandAllVars(ctx, ccModule.flags.GlobalFlags)...)
	args = append(args, expandAllVars(ctx, ccModule.flags.CFlags)...)
	if isCpp {
		args = append(args, expandAllVars(ctx, ccModule.flags.CppFlags)...)
	} else if !isAsm {
		args = append(args, expandAllVars(ctx, ccModule.flags.ConlyFlags)...)
	}
	args = append(args, expandAllVars(ctx, ccModule.flags.SystemIncludeFlags)...)
	args = append(args, src.String())
	return args
}

func generateCompdbProject(compiledModule CompiledInterface, ctx android.SingletonContext, ccModule *Module, builds map[string]compDbEntry) {
	srcs := compiledModule.Srcs()
	if len(srcs) == 0 {
		return
	}

	rootDir := getCompdbAndroidSrcRootDirectory(ctx)
	for _, src := range srcs {
		if _, ok := builds[src.String()]; !ok {
			builds[src.String()] = compDbEntry{
				Directory: rootDir,
				Arguments: getArguments(src, ctx, ccModule),
				File:      src.String(),
			}
		}
	}
}

func evalAndSplitVariable(ctx android.SingletonContext, str string) ([]string, error) {
	evaluated, err := ctx.Eval(pctx, str)
	if err == nil {
		return strings.Split(evaluated, " "), nil
	}
	return []string{""}, err
}

func getCompdbAndroidSrcRootDirectory(ctx android.SingletonContext) string {
	srcPath, _ := filepath.Abs(android.PathForSource(ctx).String())
	return srcPath
}
