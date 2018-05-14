package android

func ProtoFlags(ctx ModuleContext, p *ProtoProperties) []string {
	protoFlags := []string{}

	if len(p.Proto.Local_include_dirs) > 0 {
		localProtoIncludeDirs := PathsForModuleSrc(ctx, p.Proto.Local_include_dirs)
		protoFlags = append(protoFlags, JoinWithPrefix(localProtoIncludeDirs.Strings(), "-I"))
	}
	if len(p.Proto.Include_dirs) > 0 {
		rootProtoIncludeDirs := PathsForSource(ctx, p.Proto.Include_dirs)
		protoFlags = append(protoFlags, JoinWithPrefix(rootProtoIncludeDirs.Strings(), "-I"))
	}

	return protoFlags
}

func ProtoCanonicalPathFromRoot(ctx ModuleContext, p *ProtoProperties) bool {
	if p.Proto.Canonical_path_from_root == nil {
		return true
	}
	return *p.Proto.Canonical_path_from_root
}

func ProtoDir(ctx ModuleContext) ModuleGenPath {
	return PathForModuleGen(ctx, "proto")
}

func ProtoSubDir(ctx ModuleContext) ModuleGenPath {
	return PathForModuleGen(ctx, "proto", ctx.ModuleDir())
}

type ProtoProperties struct {
	Proto struct {
		Type                     *string `android:"arch_variant"`
		Include_dirs             []string
		Local_include_dirs       []string
		Canonical_path_from_root *bool
	} `android:"arch_variant"`
}
