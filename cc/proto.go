package cc

import (
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/pathtools"

	"android/soong/android"
)

func init() {
	pctx.HostBinToolVariable("protocCmd", "aprotoc")
}

var (
	proto = pctx.AndroidStaticRule("protoc",
		blueprint.RuleParams{
			Command:     "$protocCmd --cpp_out=$protoOutParams:$outDir -I $protoBase $protoFlags $in",
			CommandDeps: []string{"$protocCmd"},
		}, "protoFlags", "protoOutParams", "protoBase", "outDir")
)

func genProto(ctx android.ModuleContext, protoFile android.Path,
	protoFlags, protoOutParams string, root bool) (ccFile, headerFile android.WritablePath) {

	var protoBase string
	if root {
		protoBase = "."
		ccFile = android.GenPathWithExt(ctx, "proto", protoFile, "pb.cc")
		headerFile = android.GenPathWithExt(ctx, "proto", protoFile, "pb.h")
	} else {
		rel := protoFile.Rel()
		protoBase = strings.TrimSuffix(protoFile.String(), rel)
		ccFile = android.PathForModuleGen(ctx, "proto", pathtools.ReplaceExtension(rel, "pb.cc"))
		headerFile = android.PathForModuleGen(ctx, "proto", pathtools.ReplaceExtension(rel, "pb.h"))
	}

	ctx.Build(pctx, android.BuildParams{
		Rule:        proto,
		Description: "protoc " + protoFile.Rel(),
		Outputs:     android.WritablePaths{ccFile, headerFile},
		Input:       protoFile,
		Args: map[string]string{
			"outDir":         android.ProtoDir(ctx).String(),
			"protoFlags":     protoFlags,
			"protoOutParams": protoOutParams,
			"protoBase":      protoBase,
		},
	})

	return ccFile, headerFile
}

func protoDeps(ctx BaseModuleContext, deps Deps, p *android.ProtoProperties, static bool) Deps {
	var lib string

	switch String(p.Proto.Type) {
	case "full":
		if ctx.useSdk() {
			lib = "libprotobuf-cpp-full-ndk"
			static = true
		} else {
			lib = "libprotobuf-cpp-full"
		}
	case "lite", "":
		if ctx.useSdk() {
			lib = "libprotobuf-cpp-lite-ndk"
			static = true
		} else {
			lib = "libprotobuf-cpp-lite"
		}
	default:
		ctx.PropertyErrorf("proto.type", "unknown proto type %q",
			String(p.Proto.Type))
	}

	if static {
		deps.StaticLibs = append(deps.StaticLibs, lib)
		deps.ReexportStaticLibHeaders = append(deps.ReexportStaticLibHeaders, lib)
	} else {
		deps.SharedLibs = append(deps.SharedLibs, lib)
		deps.ReexportSharedLibHeaders = append(deps.ReexportSharedLibHeaders, lib)
	}

	return deps
}

func protoFlags(ctx ModuleContext, flags Flags, p *android.ProtoProperties) Flags {
	flags.CFlags = append(flags.CFlags, "-DGOOGLE_PROTOBUF_NO_RTTI")

	flags.ProtoRoot = android.ProtoCanonicalPathFromRoot(ctx, p)
	if flags.ProtoRoot {
		flags.GlobalFlags = append(flags.GlobalFlags, "-I"+android.ProtoSubDir(ctx).String())
	}
	flags.GlobalFlags = append(flags.GlobalFlags, "-I"+android.ProtoDir(ctx).String())

	flags.protoFlags = android.ProtoFlags(ctx, p)

	if String(p.Proto.Type) == "lite" {
		flags.protoOutParams = append(flags.protoOutParams, "lite")
	}

	return flags
}
