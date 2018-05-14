package cc

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/blueprint"

	"android/soong/android"
	"android/soong/cc/config"
)

const (
	objectExtension        = ".o"
	staticLibraryExtension = ".a"
)

var (
	abiCheckAllowFlags = []string{
		"-allow-unreferenced-changes",
		"-allow-unreferenced-elf-symbol-changes",
	}
)

var (
	pctx = android.NewPackageContext("android/soong/cc")

	cc = pctx.AndroidGomaStaticRule("cc", blueprint.RuleParams{
		Depfile:     "${out}.d",
		Deps:        blueprint.DepsGCC,
		Command:     "${config.CcWrapper}$ccCmd -c $cFlags -MD -MF ${out}.d -o $out $in",
		CommandDeps: []string{"$ccCmd"},
	},
		"ccCmd", "cFlags")

	ld = pctx.AndroidStaticRule("ld", blueprint.RuleParams{
		Command:        "$ldCmd @${out}.rsp ${libFlags} -o ${out} ${ldFlags}",
		CommandDeps:    []string{"$ldCmd"},
		Rspfile:        "${out}.rsp",
		RspfileContent: "${in}",
	},
		"ldCmd", "libFlags", "ldFlags")

	partialLd = pctx.AndroidStaticRule("partialLd", blueprint.RuleParams{

		Command:     "$ldCmd -no-pie -Wl,-r ${in} -o ${out} ${ldFlags}",
		CommandDeps: []string{"$ldCmd"},
	},
		"ldCmd", "ldFlags")

	ar = pctx.AndroidStaticRule("ar", blueprint.RuleParams{
		Command:        "rm -f ${out} && $arCmd $arFlags $out @${out}.rsp",
		CommandDeps:    []string{"$arCmd"},
		Rspfile:        "${out}.rsp",
		RspfileContent: "${in}",
	},
		"arCmd", "arFlags")

	darwinAr = pctx.AndroidStaticRule("darwinAr", blueprint.RuleParams{
		Command:     "rm -f ${out} && ${config.MacArPath} $arFlags $out $in",
		CommandDeps: []string{"${config.MacArPath}"},
	},
		"arFlags")

	darwinAppendAr = pctx.AndroidStaticRule("darwinAppendAr", blueprint.RuleParams{
		Command:     "cp -f ${inAr} ${out}.tmp && ${config.MacArPath} $arFlags ${out}.tmp $in && mv ${out}.tmp ${out}",
		CommandDeps: []string{"${config.MacArPath}", "${inAr}"},
	},
		"arFlags", "inAr")

	darwinStrip = pctx.AndroidStaticRule("darwinStrip", blueprint.RuleParams{
		Command:     "${config.MacStripPath} -u -r -o $out $in",
		CommandDeps: []string{"${config.MacStripPath}"},
	})

	prefixSymbols = pctx.AndroidStaticRule("prefixSymbols", blueprint.RuleParams{
		Command:     "$objcopyCmd --prefix-symbols=${prefix} ${in} ${out}",
		CommandDeps: []string{"$objcopyCmd"},
	},
		"objcopyCmd", "prefix")

	_ = pctx.StaticVariable("stripPath", "build/soong/scripts/strip.sh")
	_ = pctx.StaticVariable("xzCmd", android.TermuxExecutable("xz"))

	strip = pctx.AndroidStaticRule("strip", blueprint.RuleParams{
		Depfile:     "${out}.d",
		Deps:        blueprint.DepsGCC,
		Command:     "XZ=$xzCmd $stripPath ${args} -i ${in} -o ${out} -d ${out}.d",
		CommandDeps: []string{"$stripPath", "$xzCmd"},
	},
		"args")

	emptyFile = pctx.AndroidStaticRule("emptyFile", blueprint.RuleParams{
		Command: "rm -f $out && touch $out",
	})

	_ = pctx.StaticVariable("copyGccLibPath", "build/soong/scripts/copygcclib.sh")

	copyGccLib = pctx.AndroidStaticRule("copyGccLib", blueprint.RuleParams{
		Depfile:     "${out}.d",
		Deps:        blueprint.DepsGCC,
		Command:     "$copyGccLibPath $out $ccCmd $cFlags -print-file-name=${libName}",
		CommandDeps: []string{"$copyGccLibPath", "$ccCmd"},
	},
		"ccCmd", "cFlags", "libName")

	_ = pctx.StaticVariable("tocPath", "build/soong/scripts/toc.sh")

	toc = pctx.AndroidStaticRule("toc", blueprint.RuleParams{
		Depfile:     "${out}.d",
		Deps:        blueprint.DepsGCC,
		Command:     "$tocPath -i ${in} -o ${out} -d ${out}.d",
		CommandDeps: []string{"$tocPath"},
		Restat:      true,
	})

	clangTidy = pctx.AndroidStaticRule("clangTidy", blueprint.RuleParams{
		Command:     "rm -f $out && ${config.ClangBin}/clang-tidy $tidyFlags $in -- $cFlags && touch $out",
		CommandDeps: []string{"${config.ClangBin}/clang-tidy"},
	},
		"cFlags", "tidyFlags")

	_ = pctx.StaticVariable("yasmCmd", android.TermuxExecutable("yasm"))

	yasm = pctx.AndroidStaticRule("yasm", blueprint.RuleParams{
		Command:     "$yasmCmd $asFlags -o $out $in && $yasmCmd $asFlags -M $in >$out.d",
		CommandDeps: []string{"$yasmCmd"},
		Depfile:     "$out.d",
		Deps:        blueprint.DepsGCC,
	},
		"asFlags")

	windres = pctx.AndroidStaticRule("windres", blueprint.RuleParams{
		Command:     "$windresCmd $flags -I$$(dirname $in) -i $in -o $out",
		CommandDeps: []string{"$windresCmd"},
	},
		"windresCmd", "flags")

	_ = pctx.StaticVariable("sAbiDumper", android.TermuxExecutable("header-abi-dumper"))

	sAbiDump = pctx.AndroidStaticRule("sAbiDump", blueprint.RuleParams{
		Command:     "rm -f $out && $sAbiDumper -o ${out} $in $exportDirs -- $cFlags -w -isystem ${config.RSIncludePath}",
		CommandDeps: []string{"$sAbiDumper"},
	},
		"cFlags", "exportDirs")

	_ = pctx.StaticVariable("sAbiLinker", android.TermuxExecutable("header-abi-linker"))

	sAbiLink = pctx.AndroidStaticRule("sAbiLink", blueprint.RuleParams{
		Command:        "$sAbiLinker -o ${out} $symbolFilter -arch $arch  $exportedHeaderFlags @${out}.rsp ",
		CommandDeps:    []string{"$sAbiLinker"},
		Rspfile:        "${out}.rsp",
		RspfileContent: "${in}",
	},
		"symbolFilter", "arch", "exportedHeaderFlags")

	_ = pctx.StaticVariable("sAbiDiffer", android.TermuxExecutable("header-abi-diff"))

	sAbiDiff = pctx.AndroidRuleFunc("sAbiDiff", func(ctx android.PackageRuleContext) blueprint.RuleParams {
		commandStr := "($sAbiDiffer $allowFlags -lib $libName -arch $arch -check-all-apis -o ${out} -new $in -old $referenceDump)"
		distAbiDiffDir := android.PathForDist(ctx, "abidiffs")
		commandStr += "|| (echo ' ---- Please update abi references by running $$ANDROID_BUILD_TOP/development/vndk/tools/header-checker/utils/create_reference_dumps.py -l ${libName} ----'"
		if distAbiDiffDir.Valid() {
			commandStr += " && (mkdir -p " + distAbiDiffDir.String() + " && cp ${out} " + distAbiDiffDir.String() + ")"
		}
		commandStr += " && exit 1)"
		return blueprint.RuleParams{
			Command:     commandStr,
			CommandDeps: []string{"$sAbiDiffer"},
		}
	},
		"allowFlags", "referenceDump", "libName", "arch")

	unzipRefSAbiDump = pctx.AndroidStaticRule("unzipRefSAbiDump", blueprint.RuleParams{
		Command: "gunzip -c $in > $out",
	})
)

type builderFlags struct {
	globalFlags            string
	arFlags                string
	asFlags                string
	cFlags                 string
	toolingCFlags          string
	conlyFlags             string
	cppFlags               string
	ldFlags                string
	libFlags               string
	yaccFlags              string
	protoFlags             string
	protoOutParams         string
	tidyFlags              string
	sAbiFlags              string
	yasmFlags              string
	aidlFlags              string
	rsFlags                string
	toolchain              config.Toolchain
	clang                  bool
	tidy                   bool
	coverage               bool
	sAbiDump               bool
	protoRoot              bool
	systemIncludeFlags     string
	groupStaticLibs        bool
	arGoldPlugin           bool
	stripKeepSymbols       bool
	stripKeepMiniDebugInfo bool
	stripAddGnuDebuglink   bool
}

type Objects struct {
	objFiles      android.Paths
	tidyFiles     android.Paths
	coverageFiles android.Paths
	sAbiDumpFiles android.Paths
}

func (a Objects) Copy() Objects {
	return Objects{
		objFiles:      append(android.Paths{}, a.objFiles...),
		tidyFiles:     append(android.Paths{}, a.tidyFiles...),
		coverageFiles: append(android.Paths{}, a.coverageFiles...),
		sAbiDumpFiles: append(android.Paths{}, a.sAbiDumpFiles...),
	}
}

func (a Objects) Append(b Objects) Objects {
	return Objects{
		objFiles:      append(a.objFiles, b.objFiles...),
		tidyFiles:     append(a.tidyFiles, b.tidyFiles...),
		coverageFiles: append(a.coverageFiles, b.coverageFiles...),
		sAbiDumpFiles: append(a.sAbiDumpFiles, b.sAbiDumpFiles...),
	}
}

func TransformSourceToObj(ctx android.ModuleContext, subdir string, srcFiles android.Paths, flags builderFlags, pathDeps android.Paths, cFlagsDeps android.Paths) Objects {
	objFiles := make(android.Paths, len(srcFiles))
	var tidyFiles android.Paths
	if flags.tidy && flags.clang {
		tidyFiles = make(android.Paths, 0, len(srcFiles))
	}
	var coverageFiles android.Paths
	if flags.coverage {
		coverageFiles = make(android.Paths, 0, len(srcFiles))
	}

	commonFlags := strings.Join([]string{flags.globalFlags, flags.systemIncludeFlags}, " ")
	toolingCflags := strings.Join([]string{commonFlags, flags.toolingCFlags, flags.conlyFlags}, " ")
	cflags := strings.Join([]string{commonFlags, flags.cFlags, flags.conlyFlags}, " ")
	toolingCppflags := strings.Join([]string{commonFlags, flags.toolingCFlags, flags.cppFlags}, " ")
	cppflags := strings.Join([]string{commonFlags, flags.cFlags, flags.cppFlags}, " ")
	asflags := strings.Join([]string{commonFlags, flags.asFlags}, " ")

	var sAbiDumpFiles android.Paths
	if flags.sAbiDump && flags.clang {
		sAbiDumpFiles = make(android.Paths, 0, len(srcFiles))
	}

	if flags.clang {
		cflags += " ${config.NoOverrideClangGlobalCflags}"
		toolingCflags += " ${config.NoOverrideClangGlobalCflags}"
		cppflags += " ${config.NoOverrideClangGlobalCflags}"
		toolingCppflags += " ${config.NoOverrideClangGlobalCflags}"
	} else {
		cflags += " ${config.NoOverrideGlobalCflags}"
		cppflags += " ${config.NoOverrideGlobalCflags}"
	}

	for i, srcFile := range srcFiles {
		objFile := android.ObjPathWithExt(ctx, subdir, srcFile, "o")
		objFiles[i] = objFile

		switch srcFile.Ext() {
		case ".asm":
			ctx.Build(pctx, android.BuildParams{
				Rule:        yasm,
				Description: "yasm " + srcFile.Rel(),
				Output:      objFile,
				Input:       srcFile,
				Implicits:   cFlagsDeps,
				OrderOnly:   pathDeps,
				Args: map[string]string{
					"asFlags": flags.yasmFlags,
				},
			})
			continue
		case ".rc":
			ctx.Build(pctx, android.BuildParams{
				Rule:        windres,
				Description: "windres " + srcFile.Rel(),
				Output:      objFile,
				Input:       srcFile,
				Implicits:   cFlagsDeps,
				OrderOnly:   pathDeps,
				Args: map[string]string{
					"windresCmd": gccCmd(flags.toolchain, "windres"),
					"flags":      flags.toolchain.WindresFlags(),
				},
			})
			continue
		}

		var moduleCflags string
		var moduleToolingCflags string
		var ccCmd string
		tidy := flags.tidy && flags.clang
		coverage := flags.coverage
		dump := flags.sAbiDump && flags.clang

		switch srcFile.Ext() {
		case ".S", ".s":
			ccCmd = "gcc-7"
			moduleCflags = asflags
			tidy = false
			coverage = false
			dump = false
		case ".c":
			ccCmd = "gcc-7"
			moduleCflags = cflags
			moduleToolingCflags = toolingCflags
		case ".cpp", ".cc", ".mm":
			ccCmd = "g++-7"
			moduleCflags = cppflags
			moduleToolingCflags = toolingCppflags
		default:
			ctx.ModuleErrorf("File %s has unknown extension", srcFile)
			continue
		}

		if flags.clang {
			switch ccCmd {
			case "gcc-7":
				ccCmd = "clang-7"
			case "g++-7":
				ccCmd = "clang++"
			default:
				panic("unrecoginzied ccCmd")
			}
		}

		ccDesc := ccCmd
		ccCmd = android.TermuxExecutable(ccCmd)

		var implicitOutputs android.WritablePaths
		if coverage {
			gcnoFile := android.ObjPathWithExt(ctx, subdir, srcFile, "gcno")
			implicitOutputs = append(implicitOutputs, gcnoFile)
			coverageFiles = append(coverageFiles, gcnoFile)
		}

		ctx.Build(pctx, android.BuildParams{
			Rule:            cc,
			Description:     ccDesc + " " + srcFile.Rel(),
			Output:          objFile,
			ImplicitOutputs: implicitOutputs,
			Input:           srcFile,
			Implicits:       cFlagsDeps,
			OrderOnly:       pathDeps,
			Args: map[string]string{
				"cFlags": moduleCflags,
				"ccCmd":  ccCmd,
			},
		})

		if tidy {
			tidyFile := android.ObjPathWithExt(ctx, subdir, srcFile, "tidy")
			tidyFiles = append(tidyFiles, tidyFile)

			ctx.Build(pctx, android.BuildParams{
				Rule:        clangTidy,
				Description: "clang-tidy " + srcFile.Rel(),
				Output:      tidyFile,
				Input:       srcFile,
				Implicit:    objFile,
				Args: map[string]string{
					"cFlags":    moduleToolingCflags,
					"tidyFlags": flags.tidyFlags,
				},
			})
		}

		if dump {
			sAbiDumpFile := android.ObjPathWithExt(ctx, subdir, srcFile, "sdump")
			sAbiDumpFiles = append(sAbiDumpFiles, sAbiDumpFile)

			ctx.Build(pctx, android.BuildParams{
				Rule:        sAbiDump,
				Description: "header-abi-dumper " + srcFile.Rel(),
				Output:      sAbiDumpFile,
				Input:       srcFile,
				Implicit:    objFile,
				Args: map[string]string{
					"cFlags":     moduleToolingCflags,
					"exportDirs": flags.sAbiFlags,
				},
			})
		}
	}

	return Objects{
		objFiles:      objFiles,
		tidyFiles:     tidyFiles,
		coverageFiles: coverageFiles,
		sAbiDumpFiles: sAbiDumpFiles,
	}
}

func TransformObjToStaticLib(ctx android.ModuleContext, objFiles android.Paths, flags builderFlags, outputFile android.ModuleOutPath, deps android.Paths) {
	arCmd := android.TermuxExecutable("llvm-ar")
	arFlags := "crsD -format=gnu"
	if flags.arGoldPlugin {
		arFlags += " --plugin ${config.LLVMGoldPlugin}"
	}
	if flags.arFlags != "" {
		arFlags += " " + flags.arFlags
	}

	ctx.Build(pctx, android.BuildParams{
		Rule:        ar,
		Description: "static link " + outputFile.Base(),
		Output:      outputFile,
		Inputs:      objFiles,
		Implicits:   deps,
		Args: map[string]string{
			"arFlags": arFlags,
			"arCmd":   arCmd,
		},
	})
}

func transformDarwinObjToStaticLib(ctx android.ModuleContext, objFiles android.Paths, flags builderFlags, outputFile android.ModuleOutPath, deps android.Paths) {
	arFlags := "cqs"

	if len(objFiles) == 0 {
		dummy := android.PathForModuleOut(ctx, "dummy"+objectExtension)
		dummyAr := android.PathForModuleOut(ctx, "dummy"+staticLibraryExtension)

		ctx.Build(pctx, android.BuildParams{
			Rule:        emptyFile,
			Description: "empty object file",
			Output:      dummy,
			Implicits:   deps,
		})

		ctx.Build(pctx, android.BuildParams{
			Rule:        darwinAr,
			Description: "empty static archive",
			Output:      dummyAr,
			Input:       dummy,
			Args: map[string]string{
				"arFlags": arFlags,
			},
		})

		ctx.Build(pctx, android.BuildParams{
			Rule:        darwinAppendAr,
			Description: "static link " + outputFile.Base(),
			Output:      outputFile,
			Input:       dummy,
			Args: map[string]string{
				"arFlags": "d",
				"inAr":    dummyAr.String(),
			},
		})

		return
	}

	objFilesLists, err := splitListForSize(objFiles, 131072)
	if err != nil {
		ctx.ModuleErrorf("%s", err.Error())
	}

	var in, out android.WritablePath
	for i, l := range objFilesLists {
		in = out
		out = outputFile
		if i != len(objFilesLists)-1 {
			out = android.PathForModuleOut(ctx, outputFile.Base()+strconv.Itoa(i))
		}

		build := android.BuildParams{
			Rule:        darwinAr,
			Description: "static link " + out.Base(),
			Output:      out,
			Inputs:      l,
			Implicits:   deps,
			Args: map[string]string{
				"arFlags": arFlags,
			},
		}
		if i != 0 {
			build.Rule = darwinAppendAr
			build.Args["inAr"] = in.String()
		}
		ctx.Build(pctx, build)
	}
}

func TransformObjToDynamicBinary(ctx android.ModuleContext, objFiles, sharedLibs, staticLibs, lateStaticLibs, wholeStaticLibs, deps android.Paths, groupLate bool, flags builderFlags, outputFile android.WritablePath) {
	var ldCmd string
	if flags.clang {
		ldCmd = android.TermuxExecutable("clang++")
	} else {
		ldCmd = android.TermuxExecutable("g++-7")
	}

	var libFlagsList []string

	if len(flags.libFlags) > 0 {
		libFlagsList = append(libFlagsList, flags.libFlags)
	}

	if len(wholeStaticLibs) > 0 {
		libFlagsList = append(libFlagsList, "-Wl,--whole-archive ")
		libFlagsList = append(libFlagsList, wholeStaticLibs.Strings()...)
		libFlagsList = append(libFlagsList, "-Wl,--no-whole-archive ")
	}

	if flags.groupStaticLibs && len(staticLibs) > 0 {
		libFlagsList = append(libFlagsList, "-Wl,--start-group")
	}
	libFlagsList = append(libFlagsList, staticLibs.Strings()...)
	if flags.groupStaticLibs && len(staticLibs) > 0 {
		libFlagsList = append(libFlagsList, "-Wl,--end-group")
	}

	if groupLate && len(lateStaticLibs) > 0 {
		libFlagsList = append(libFlagsList, "-Wl,--start-group")
	}
	libFlagsList = append(libFlagsList, lateStaticLibs.Strings()...)
	if groupLate && len(lateStaticLibs) > 0 {
		libFlagsList = append(libFlagsList, "-Wl,--end-group")
	}

	for _, lib := range sharedLibs {
		libFlagsList = append(libFlagsList, lib.String())
	}

	deps = append(deps, staticLibs...)
	deps = append(deps, lateStaticLibs...)
	deps = append(deps, wholeStaticLibs...)

	ctx.Build(pctx, android.BuildParams{
		Rule:        ld,
		Description: "link " + outputFile.Base(),
		Output:      outputFile,
		Inputs:      objFiles,
		Implicits:   deps,
		Args: map[string]string{
			"ldCmd":    ldCmd,
			"libFlags": strings.Join(libFlagsList, " "),
			"ldFlags":  flags.ldFlags,
		},
	})
}

func TransformDumpToLinkedDump(ctx android.ModuleContext, sAbiDumps android.Paths, soFile android.Path, baseName, exportedHeaderFlags string) android.OptionalPath {
	outputFile := android.PathForModuleOut(ctx, baseName+".lsdump")
	sabiLock.Lock()
	lsdumpPaths = append(lsdumpPaths, outputFile.String())
	sabiLock.Unlock()
	symbolFilterStr := "-so " + soFile.String()
	ctx.Build(pctx, android.BuildParams{
		Rule:        sAbiLink,
		Description: "header-abi-linker " + outputFile.Base(),
		Output:      outputFile,
		Inputs:      sAbiDumps,
		Implicit:    soFile,
		Args: map[string]string{
			"symbolFilter":        symbolFilterStr,
			"arch":                ctx.Arch().ArchType.Name,
			"exportedHeaderFlags": exportedHeaderFlags,
		},
	})
	return android.OptionalPathForPath(outputFile)
}

func UnzipRefDump(ctx android.ModuleContext, zippedRefDump android.Path, baseName string) android.Path {
	outputFile := android.PathForModuleOut(ctx, baseName+"_ref.lsdump")
	ctx.Build(pctx, android.BuildParams{
		Rule:        unzipRefSAbiDump,
		Description: "gunzip" + outputFile.Base(),
		Output:      outputFile,
		Input:       zippedRefDump,
	})
	return outputFile
}

func SourceAbiDiff(ctx android.ModuleContext, inputDump android.Path, referenceDump android.Path, baseName, exportedHeaderFlags string, isVndkExt bool) android.OptionalPath {
	outputFile := android.PathForModuleOut(ctx, baseName+".abidiff")
	localAbiCheckAllowFlags := append([]string(nil), abiCheckAllowFlags...)
	if exportedHeaderFlags == "" {
		localAbiCheckAllowFlags = append(localAbiCheckAllowFlags, "-advice-only")
	}
	if isVndkExt {
		localAbiCheckAllowFlags = append(localAbiCheckAllowFlags, "-allow-extensions")
	}

	ctx.Build(pctx, android.BuildParams{
		Rule:        sAbiDiff,
		Description: "header-abi-diff " + outputFile.Base(),
		Output:      outputFile,
		Input:       inputDump,
		Implicit:    referenceDump,
		Args: map[string]string{
			"referenceDump": referenceDump.String(),
			"libName":       baseName[0:(len(baseName) - len(filepath.Ext(baseName)))],
			"arch":          ctx.Arch().ArchType.Name,
			"allowFlags":    strings.Join(localAbiCheckAllowFlags, " "),
		},
	})
	return android.OptionalPathForPath(outputFile)
}

func TransformSharedObjectToToc(ctx android.ModuleContext, inputFile android.Path, outputFile android.WritablePath, flags builderFlags) {
	ctx.Build(pctx, android.BuildParams{
		Rule:        toc,
		Description: "generate toc " + inputFile.Base(),
		Output:      outputFile,
		Input:       inputFile,
	})
}

func TransformObjsToObj(ctx android.ModuleContext, objFiles android.Paths, flags builderFlags, outputFile android.WritablePath) {
	var ldCmd string
	if flags.clang {
		ldCmd = android.TermuxExecutable("clang++")
	} else {
		ldCmd = android.TermuxExecutable("g++-7")
	}

	ctx.Build(pctx, android.BuildParams{
		Rule:        partialLd,
		Description: "link " + outputFile.Base(),
		Output:      outputFile,
		Inputs:      objFiles,
		Args: map[string]string{
			"ldCmd":   ldCmd,
			"ldFlags": flags.ldFlags,
		},
	})
}

func TransformBinaryPrefixSymbols(ctx android.ModuleContext, prefix string, inputFile android.Path, flags builderFlags, outputFile android.WritablePath) {
	objcopyCmd := android.TermuxExecutable("objcopy")
	ctx.Build(pctx, android.BuildParams{
		Rule:        prefixSymbols,
		Description: "prefix symbols " + outputFile.Base(),
		Output:      outputFile,
		Input:       inputFile,
		Args: map[string]string{
			"objcopyCmd": objcopyCmd,
			"prefix":     prefix,
		},
	})
}

func TransformStrip(ctx android.ModuleContext, inputFile android.Path, outputFile android.WritablePath, flags builderFlags) {
	args := ""
	if flags.stripAddGnuDebuglink {
		args += " --add-gnu-debuglink"
	}
	if flags.stripKeepMiniDebugInfo {
		args += " --keep-mini-debug-info"
	}
	if flags.stripKeepSymbols {
		args += " --keep-symbols"
	}

	ctx.Build(pctx, android.BuildParams{
		Rule:        strip,
		Description: "strip " + outputFile.Base(),
		Output:      outputFile,
		Input:       inputFile,
		Args: map[string]string{
			"args": args,
		},
	})
}

func TransformDarwinStrip(ctx android.ModuleContext, inputFile android.Path, outputFile android.WritablePath) {
	ctx.Build(pctx, android.BuildParams{
		Rule:        darwinStrip,
		Description: "strip " + outputFile.Base(),
		Output:      outputFile,
		Input:       inputFile,
	})
}

func TransformCoverageFilesToLib(ctx android.ModuleContext, inputs Objects, flags builderFlags, baseName string) android.OptionalPath {
	if len(inputs.coverageFiles) > 0 {
		outputFile := android.PathForModuleOut(ctx, baseName+".gcnodir")
		TransformObjToStaticLib(ctx, inputs.coverageFiles, flags, outputFile, nil)
		return android.OptionalPathForPath(outputFile)
	}
	return android.OptionalPath{}
}

func CopyGccLib(ctx android.ModuleContext, libName string, flags builderFlags, outputFile android.WritablePath) {
	ctx.Build(pctx, android.BuildParams{
		Rule:        copyGccLib,
		Description: "copy gcc library " + libName,
		Output:      outputFile,
		Args: map[string]string{
			"ccCmd":   android.TermuxExecutable("gcc-7"),
			"cFlags":  flags.globalFlags,
			"libName": libName,
		},
	})
}

func gccCmd(toolchain config.Toolchain, cmd string) string {
	return filepath.Join(toolchain.GccRoot(), "bin", cmd)
}

func splitListForSize(list android.Paths, limit int) (lists []android.Paths, err error) {
	var i int

	start := 0
	bytes := 0
	for i = range list {
		l := len(list[i].String())
		if l > limit {
			return nil, fmt.Errorf("list element greater than size limit (%d)", limit)
		}
		if bytes+l > limit {
			lists = append(lists, list[start:i])
			start = i
			bytes = 0
		}
		bytes += l + 1
	}

	lists = append(lists, list[start:])
	totalLen := 0
	for _, l := range lists {
		totalLen += len(l)
	}
	if totalLen != len(list) {
		panic(fmt.Errorf("Failed breaking up list, %d != %d", len(list), totalLen))
	}
	return lists, nil
}
