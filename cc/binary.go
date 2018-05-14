package cc

import (
	"android/soong/android"
)

type BinaryLinkerProperties struct {
	Static_executable      *bool   `android:"arch_variant"`
	Stem                   *string `android:"arch_variant"`
	Suffix                 *string `android:"arch_variant"`
	Prefix_symbols         *string
	Version_script         *string `android:"arch_variant"`
	Symlink_preferred_arch *bool
	Symlinks               []string `android:"arch_variant"`
	No_pie                 *bool    `android:"arch_variant"`
	DynamicLinker          string   `blueprint:"mutated"`
	Overrides              []string
}

func init() {
	android.RegisterModuleType("cc_binary", binaryFactory)
	android.RegisterModuleType("cc_binary_host", binaryHostFactory)
}

func binaryFactory() android.Module {
	module, _ := NewBinary(android.HostAndDeviceSupported)
	return module.Init()
}

func binaryHostFactory() android.Module {
	module, _ := NewBinary(android.HostSupported)
	return module.Init()
}

type binaryDecorator struct {
	*baseLinker
	*baseInstaller
	stripper
	Properties         BinaryLinkerProperties
	toolPath           android.OptionalPath
	symlinks           []string
	coverageOutputFile android.OptionalPath
}

var _ linker = (*binaryDecorator)(nil)

func (binary *binaryDecorator) linkerProps() []interface{} {
	return append(binary.baseLinker.linkerProps(), &binary.Properties, &binary.stripper.StripProperties)

}

func (binary *binaryDecorator) getStem(ctx BaseModuleContext) string {
	stem := ctx.baseModuleName()
	if String(binary.Properties.Stem) != "" {
		stem = String(binary.Properties.Stem)
	}

	return stem + String(binary.Properties.Suffix)
}

func (binary *binaryDecorator) linkerDeps(ctx DepsContext, deps Deps) Deps {
	deps = binary.baseLinker.linkerDeps(ctx, deps)
	if !binary.static() && inList("libc", deps.StaticLibs) {
		ctx.ModuleErrorf("statically linking libc to dynamic executable, please remove libc\nfrom static libs or set static_executable: true")
	}

	android.ExtractSourceDeps(ctx, binary.Properties.Version_script)

	return deps
}

func (binary *binaryDecorator) isDependencyRoot() bool {
	return true
}

func NewBinary(hod android.HostOrDeviceSupported) (*Module, *binaryDecorator) {
	module := newModule(hod, android.MultilibFirst)
	binary := &binaryDecorator{
		baseLinker:    NewBaseLinker(),
		baseInstaller: NewBaseInstaller("bin", "", InstallInSystem),
	}
	module.compiler = NewBaseCompiler()
	module.linker = binary
	module.installer = binary
	return module, binary
}

func (binary *binaryDecorator) linkerInit(ctx BaseModuleContext) {
	binary.baseLinker.linkerInit(ctx)

	if binary.Properties.Static_executable == nil && ctx.Config().HostStaticBinaries() {
		binary.Properties.Static_executable = BoolPtr(true)
	}
}

func (binary *binaryDecorator) static() bool {
	return Bool(binary.Properties.Static_executable)
}

func (binary *binaryDecorator) staticBinary() bool {
	return binary.static()
}

func (binary *binaryDecorator) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	flags = binary.baseLinker.linkerFlags(ctx, flags)
	flags.CFlags = append(flags.CFlags, "-fPIE")

	if binary.static() {
		flags.LdFlags = append(flags.LdFlags, "-static", "-Bstatic", "-Wl,--gc-sections")
	} else {
		flags.DynamicLinker = "/system/bin/linker64"
		flags.LdFlags = append(flags.LdFlags, "-pie", "-Bdynamic", "-Wl,--gc-sections", "-Wl,-z,nocopyreloc")
	}

	return flags
}

func (binary *binaryDecorator) link(ctx ModuleContext, flags Flags, deps PathDeps, objs Objects) android.Path {

	versionScript := ctx.ExpandOptionalSource(binary.Properties.Version_script, "version_script")
	fileName := binary.getStem(ctx) + flags.Toolchain.ExecutableSuffix()
	outputFile := android.PathForModuleOut(ctx, fileName)
	ret := outputFile

	var linkerDeps android.Paths

	sharedLibs := deps.SharedLibs
	sharedLibs = append(sharedLibs, deps.LateSharedLibs...)

	if versionScript.Valid() {
		flags.LdFlags = append(flags.LdFlags, "-Wl,--version-script,"+versionScript.String())
		linkerDeps = append(linkerDeps, versionScript.Path())
	}

	if deps.LinkerScript.Valid() {
		flags.LdFlags = append(flags.LdFlags, "-Wl,-T,"+deps.LinkerScript.String())
		linkerDeps = append(linkerDeps, deps.LinkerScript.Path())
	}

	if flags.DynamicLinker != "" {
		flags.LdFlags = append(flags.LdFlags, " -Wl,-dynamic-linker,"+flags.DynamicLinker)
	}

	builderFlags := flagsToBuilderFlags(flags)

	if binary.stripper.needsStrip(ctx) {
		strippedOutputFile := outputFile
		outputFile = android.PathForModuleOut(ctx, "unstripped", fileName)
		binary.stripper.strip(ctx, outputFile, strippedOutputFile, builderFlags)
	}

	if String(binary.Properties.Prefix_symbols) != "" {
		afterPrefixSymbols := outputFile
		outputFile = android.PathForModuleOut(ctx, "unprefixed", fileName)
		TransformBinaryPrefixSymbols(ctx, String(binary.Properties.Prefix_symbols), outputFile, flagsToBuilderFlags(flags), afterPrefixSymbols)
	}

	if Bool(binary.baseLinker.Properties.Use_version_lib) && ctx.Host() {
		versionedOutputFile := outputFile
		outputFile = android.PathForModuleOut(ctx, "unversioned", fileName)
		binary.injectVersionSymbol(ctx, outputFile, versionedOutputFile)
	}

	linkerDeps = append(linkerDeps, deps.SharedLibsDeps...)
	linkerDeps = append(linkerDeps, deps.LateSharedLibsDeps...)
	linkerDeps = append(linkerDeps, objs.tidyFiles...)
	linkerDeps = append(linkerDeps, flags.LdFlagsDeps...)

	TransformObjToDynamicBinary(ctx, objs.objFiles, sharedLibs, deps.StaticLibs, deps.LateStaticLibs, deps.WholeStaticLibs, linkerDeps, true, builderFlags, outputFile)

	objs.coverageFiles = append(objs.coverageFiles, deps.StaticLibObjs.coverageFiles...)
	objs.coverageFiles = append(objs.coverageFiles, deps.WholeStaticLibObjs.coverageFiles...)
	binary.coverageOutputFile = TransformCoverageFilesToLib(ctx, objs, builderFlags, binary.getStem(ctx))

	return ret
}

func (binary *binaryDecorator) install(ctx ModuleContext, file android.Path) {
	binary.baseInstaller.install(ctx, file)
	for _, symlink := range binary.Properties.Symlinks {
		binary.symlinks = append(binary.symlinks, symlink+String(binary.Properties.Suffix)+ctx.toolchain().ExecutableSuffix())
	}

	if Bool(binary.Properties.Symlink_preferred_arch) {
		if String(binary.Properties.Stem) == "" && String(binary.Properties.Suffix) == "" {
			ctx.PropertyErrorf("symlink_preferred_arch", "must also specify stem or suffix")
		}
		if ctx.TargetPrimary() {
			binary.symlinks = append(binary.symlinks, ctx.baseModuleName())
		}
	}

	for _, symlink := range binary.symlinks {
		ctx.InstallSymlink(binary.baseInstaller.installDir(ctx), symlink, binary.baseInstaller.path)
	}

	if ctx.Os().Class == android.Host {
		binary.toolPath = android.OptionalPathForPath(binary.baseInstaller.path)
	}
}

func (binary *binaryDecorator) hostToolPath() android.OptionalPath {
	return binary.toolPath
}
