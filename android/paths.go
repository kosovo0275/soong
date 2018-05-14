package android

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/google/blueprint"
	"github.com/google/blueprint/pathtools"
)

type PathContext interface {
	Fs() pathtools.FileSystem
	Config() Config
	AddNinjaFileDeps(deps ...string)
}

type PathGlobContext interface {
	GlobWithDeps(globPattern string, excludes []string) ([]string, error)
}

var _ PathContext = SingletonContext(nil)
var _ PathContext = ModuleContext(nil)

type ModuleInstallPathContext interface {
	PathContext

	androidBaseContext

	InstallInData() bool
	InstallInSanitizerDir() bool
}

var _ ModuleInstallPathContext = ModuleContext(nil)

type errorfContext interface {
	Errorf(format string, args ...interface{})
}

var _ errorfContext = blueprint.SingletonContext(nil)

type moduleErrorf interface {
	ModuleErrorf(format string, args ...interface{})
}

var _ moduleErrorf = blueprint.ModuleContext(nil)

func reportPathError(ctx PathContext, err error) {
	reportPathErrorf(ctx, "%s", err.Error())
}

func reportPathErrorf(ctx PathContext, format string, args ...interface{}) {
	if mctx, ok := ctx.(moduleErrorf); ok {
		mctx.ModuleErrorf(format, args...)
	} else if ectx, ok := ctx.(errorfContext); ok {
		ectx.Errorf(format, args...)
	} else {
		panic(fmt.Sprintf(format, args...))
	}
}

type Path interface {
	String() string
	Ext() string
	Base() string
	Rel() string
}

type WritablePath interface {
	Path
	writablePath()
}

type genPathProvider interface {
	genPathWithExt(ctx ModuleContext, subdir, ext string) ModuleGenPath
}
type objPathProvider interface {
	objPathWithExt(ctx ModuleContext, subdir, ext string) ModuleObjPath
}
type resPathProvider interface {
	resPathWithName(ctx ModuleContext, name string) ModuleResPath
}

func GenPathWithExt(ctx ModuleContext, subdir string, p Path, ext string) ModuleGenPath {
	if path, ok := p.(genPathProvider); ok {
		return path.genPathWithExt(ctx, subdir, ext)
	}
	reportPathErrorf(ctx, "Tried to create generated file from unsupported path: %s(%s)", reflect.TypeOf(p).Name(), p)
	return PathForModuleGen(ctx)
}

func ObjPathWithExt(ctx ModuleContext, subdir string, p Path, ext string) ModuleObjPath {
	if path, ok := p.(objPathProvider); ok {
		return path.objPathWithExt(ctx, subdir, ext)
	}
	reportPathErrorf(ctx, "Tried to create object file from unsupported path: %s (%s)", reflect.TypeOf(p).Name(), p)
	return PathForModuleObj(ctx)
}

func ResPathWithName(ctx ModuleContext, p Path, name string) ModuleResPath {
	if path, ok := p.(resPathProvider); ok {
		return path.resPathWithName(ctx, name)
	}
	reportPathErrorf(ctx, "Tried to create res file from unsupported path: %s (%s)", reflect.TypeOf(p).Name(), p)
	return PathForModuleRes(ctx)
}

type OptionalPath struct {
	valid bool
	path  Path
}

func OptionalPathForPath(path Path) OptionalPath {
	if path == nil {
		return OptionalPath{}
	}
	return OptionalPath{valid: true, path: path}
}

func (p OptionalPath) Valid() bool {
	return p.valid
}

func (p OptionalPath) Path() Path {
	if !p.valid {
		panic("Requesting an invalid path")
	}
	return p.path
}

func (p OptionalPath) String() string {
	if p.valid {
		return p.path.String()
	} else {
		return ""
	}
}

type Paths []Path

func PathsForSource(ctx PathContext, paths []string) Paths {
	ret := make(Paths, len(paths))
	for i, path := range paths {
		ret[i] = PathForSource(ctx, path)
	}
	return ret
}

func ExistentPathsForSources(ctx PathContext, paths []string) Paths {
	ret := make(Paths, 0, len(paths))
	for _, path := range paths {
		p := ExistentPathForSource(ctx, path)
		if p.Valid() {
			ret = append(ret, p.Path())
		}
	}
	return ret
}

func PathsForModuleSrc(ctx ModuleContext, paths []string) Paths {
	ret := make(Paths, len(paths))
	for i, path := range paths {
		ret[i] = PathForModuleSrc(ctx, path)
	}
	return ret
}

func pathsForModuleSrcFromFullPath(ctx ModuleContext, paths []string, incDirs bool) Paths {
	prefix := filepath.Join(ctx.Config().srcDir, ctx.ModuleDir()) + "/"
	if prefix == "./" {
		prefix = ""
	}
	ret := make(Paths, 0, len(paths))
	for _, p := range paths {
		if !incDirs && strings.HasSuffix(p, "/") {
			continue
		}
		path := filepath.Clean(p)
		if !strings.HasPrefix(path, prefix) {
			reportPathErrorf(ctx, "Path '%s' is not in module source directory '%s'", p, prefix)
			continue
		}
		ret = append(ret, PathForModuleSrc(ctx, path[len(prefix):]))
	}
	return ret
}

func PathsWithOptionalDefaultForModuleSrc(ctx ModuleContext, input []string, def string) Paths {
	if len(input) > 0 {
		return PathsForModuleSrc(ctx, input)
	}
	path := filepath.Join(ctx.Config().srcDir, ctx.ModuleDir(), def)
	return ctx.Glob(path, nil)
}

func (p Paths) Strings() []string {
	if p == nil {
		return nil
	}
	ret := make([]string, len(p))
	for i, path := range p {
		ret[i] = path.String()
	}
	return ret
}

func FirstUniquePaths(list Paths) Paths {
	k := 0
outer:
	for i := 0; i < len(list); i++ {
		for j := 0; j < k; j++ {
			if list[i] == list[j] {
				continue outer
			}
		}
		list[k] = list[i]
		k++
	}
	return list[:k]
}

func LastUniquePaths(list Paths) Paths {
	totalSkip := 0
	for i := len(list) - 1; i >= totalSkip; i-- {
		skip := 0
		for j := i - 1; j >= totalSkip; j-- {
			if list[i] == list[j] {
				skip++
			} else {
				list[j+skip] = list[j]
			}
		}
		totalSkip += skip
	}
	return list[totalSkip:]
}

func ReversePaths(list Paths) Paths {
	if list == nil {
		return nil
	}
	ret := make(Paths, len(list))
	for i := range list {
		ret[i] = list[len(list)-1-i]
	}
	return ret
}

func indexPathList(s Path, list []Path) int {
	for i, l := range list {
		if l == s {
			return i
		}
	}

	return -1
}

func inPathList(p Path, list []Path) bool {
	return indexPathList(p, list) != -1
}

func FilterPathList(list []Path, filter []Path) (remainder []Path, filtered []Path) {
	for _, l := range list {
		if inPathList(l, filter) {
			filtered = append(filtered, l)
		} else {
			remainder = append(remainder, l)
		}
	}

	return
}

func (p Paths) HasExt(ext string) bool {
	for _, path := range p {
		if path.Ext() == ext {
			return true
		}
	}

	return false
}

func (p Paths) FilterByExt(ext string) Paths {
	ret := make(Paths, 0, len(p))
	for _, path := range p {
		if path.Ext() == ext {
			ret = append(ret, path)
		}
	}
	return ret
}

func (p Paths) FilterOutByExt(ext string) Paths {
	ret := make(Paths, 0, len(p))
	for _, path := range p {
		if path.Ext() != ext {
			ret = append(ret, path)
		}
	}
	return ret
}

type DirectorySortedPaths Paths

func PathsToDirectorySortedPaths(paths Paths) DirectorySortedPaths {
	ret := append(DirectorySortedPaths(nil), paths...)
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].String() < ret[j].String()
	})
	return ret
}

func (p DirectorySortedPaths) PathsInDirectory(dir string) Paths {
	prefix := filepath.Clean(dir) + "/"
	start := sort.Search(len(p), func(i int) bool {
		return prefix < p[i].String()
	})

	ret := p[start:]

	end := sort.Search(len(ret), func(i int) bool {
		return !strings.HasPrefix(ret[i].String(), prefix)
	})

	ret = ret[:end]

	return Paths(ret)
}

type WritablePaths []WritablePath

func (p WritablePaths) Strings() []string {
	if p == nil {
		return nil
	}
	ret := make([]string, len(p))
	for i, path := range p {
		ret[i] = path.String()
	}
	return ret
}

func (p WritablePaths) Paths() Paths {
	if p == nil {
		return nil
	}
	ret := make(Paths, len(p))
	for i, path := range p {
		ret[i] = path
	}
	return ret
}

type basePath struct {
	path   string
	config Config
	rel    string
}

func (p basePath) Ext() string {
	return filepath.Ext(p.path)
}

func (p basePath) Base() string {
	return filepath.Base(p.path)
}

func (p basePath) Rel() string {
	if p.rel != "" {
		return p.rel
	}
	return p.path
}

func (p basePath) String() string {
	return p.path
}

func (p basePath) withRel(rel string) basePath {
	p.path = filepath.Join(p.path, rel)
	p.rel = rel
	return p
}

type SourcePath struct {
	basePath
}

var _ Path = SourcePath{}

func (p SourcePath) withRel(rel string) SourcePath {
	p.basePath = p.basePath.withRel(rel)
	return p
}

func safePathForSource(ctx PathContext, path string) SourcePath {
	p, err := validateSafePath(path)
	if err != nil {
		reportPathError(ctx, err)
	}
	ret := SourcePath{basePath{p, ctx.Config(), ""}}

	abs, err := filepath.Abs(ret.String())
	if err != nil {
		reportPathError(ctx, err)
		return ret
	}
	buildroot, err := filepath.Abs(ctx.Config().buildDir)
	if err != nil {
		reportPathError(ctx, err)
		return ret
	}
	if strings.HasPrefix(abs, buildroot) {
		reportPathErrorf(ctx, "source path %s is in output", abs)
		return ret
	}

	return ret
}

func pathForSource(ctx PathContext, pathComponents ...string) (SourcePath, error) {
	p, err := validatePath(pathComponents...)
	ret := SourcePath{basePath{p, ctx.Config(), ""}}
	if err != nil {
		return ret, err
	}

	abs, err := filepath.Abs(ret.String())
	if err != nil {
		return ret, err
	}
	buildroot, err := filepath.Abs(ctx.Config().buildDir)
	if err != nil {
		return ret, err
	}
	if strings.HasPrefix(abs, buildroot) {
		return ret, fmt.Errorf("source path %s is in output", abs)
	}

	if pathtools.IsGlob(ret.String()) {
		return ret, fmt.Errorf("path may not contain a glob: %s", ret.String())
	}

	return ret, nil
}

func existsWithDependencies(ctx PathContext, path SourcePath) (exists bool, err error) {
	var files []string

	if gctx, ok := ctx.(PathGlobContext); ok {
		files, err = gctx.GlobWithDeps(path.String(), nil)
	} else {
		var deps []string
		files, deps, err = pathtools.Glob(path.String(), nil)
		ctx.AddNinjaFileDeps(deps...)
	}

	if err != nil {
		return false, fmt.Errorf("glob: %s", err.Error())
	}

	return len(files) > 0, nil
}

func PathForSource(ctx PathContext, pathComponents ...string) SourcePath {
	path, err := pathForSource(ctx, pathComponents...)
	if err != nil {
		reportPathError(ctx, err)
	}

	if modCtx, ok := ctx.(ModuleContext); ok && ctx.Config().AllowMissingDependencies() {
		exists, err := existsWithDependencies(ctx, path)
		if err != nil {
			reportPathError(ctx, err)
		}
		if !exists {
			modCtx.AddMissingDependencies([]string{path.String()})
		}
	} else if exists, _, err := ctx.Fs().Exists(path.String()); err != nil {
		reportPathErrorf(ctx, "%s: %s", path, err.Error())
	} else if !exists {
		reportPathErrorf(ctx, "source path %s does not exist", path)
	}
	return path
}

func ExistentPathForSource(ctx PathContext, pathComponents ...string) OptionalPath {
	path, err := pathForSource(ctx, pathComponents...)
	if err != nil {
		reportPathError(ctx, err)
		return OptionalPath{}
	}

	exists, err := existsWithDependencies(ctx, path)
	if err != nil {
		reportPathError(ctx, err)
		return OptionalPath{}
	}
	if !exists {
		return OptionalPath{}
	}
	return OptionalPathForPath(path)
}

func (p SourcePath) String() string {
	return filepath.Join(p.config.srcDir, p.path)
}

func (p SourcePath) Join(ctx PathContext, paths ...string) SourcePath {
	path, err := validatePath(paths...)
	if err != nil {
		reportPathError(ctx, err)
	}
	return p.withRel(path)
}

func (p SourcePath) OverlayPath(ctx ModuleContext, path Path) OptionalPath {
	var relDir string
	if moduleSrcPath, ok := path.(ModuleSrcPath); ok {
		relDir = moduleSrcPath.path
	} else if srcPath, ok := path.(SourcePath); ok {
		relDir = srcPath.path
	} else {
		reportPathErrorf(ctx, "Cannot find relative path for %s(%s)", reflect.TypeOf(path).Name(), path)
		return OptionalPath{}
	}
	dir := filepath.Join(p.config.srcDir, p.path, relDir)
	if pathtools.IsGlob(dir) {
		reportPathErrorf(ctx, "Path may not contain a glob: %s", dir)
	}
	paths, err := ctx.GlobWithDeps(dir, nil)
	if err != nil {
		reportPathErrorf(ctx, "glob: %s", err.Error())
		return OptionalPath{}
	}
	if len(paths) == 0 {
		return OptionalPath{}
	}
	relPath, err := filepath.Rel(p.config.srcDir, paths[0])
	if err != nil {
		reportPathError(ctx, err)
		return OptionalPath{}
	}
	return OptionalPathForPath(PathForSource(ctx, relPath))
}

type OutputPath struct {
	basePath
}

func (p OutputPath) withRel(rel string) OutputPath {
	p.basePath = p.basePath.withRel(rel)
	return p
}

var _ Path = OutputPath{}

func PathForOutput(ctx PathContext, pathComponents ...string) OutputPath {
	path, err := validatePath(pathComponents...)
	if err != nil {
		reportPathError(ctx, err)
	}
	return OutputPath{basePath{path, ctx.Config(), ""}}
}

func (p OutputPath) writablePath() {}

func (p OutputPath) String() string {
	return filepath.Join(p.config.buildDir, p.path)
}

func (p OutputPath) RelPathString() string {
	return p.path
}

func (p OutputPath) Join(ctx PathContext, paths ...string) OutputPath {
	path, err := validatePath(paths...)
	if err != nil {
		reportPathError(ctx, err)
	}
	return p.withRel(path)
}

func PathForIntermediates(ctx PathContext, paths ...string) OutputPath {
	path, err := validatePath(paths...)
	if err != nil {
		reportPathError(ctx, err)
	}
	return PathForOutput(ctx, ".intermediates", path)
}

type DistPath struct {
	basePath
}

func (p DistPath) withRel(rel string) DistPath {
	p.basePath = p.basePath.withRel(rel)
	return p
}

var _ Path = DistPath{}

func PathForDist(ctx PathContext, pathComponents ...string) DistPath {
	path, err := validatePath(pathComponents...)
	if err != nil {
		reportPathError(ctx, err)
	}
	return DistPath{basePath{path, ctx.Config(), ""}}
}

func (p DistPath) writablePath() {}

func (p DistPath) Valid() bool {
	return p.config.productVariables.DistDir != nil && *p.config.productVariables.DistDir != ""
}

func (p DistPath) String() string {
	if !p.Valid() {
		panic("Requesting an invalid path")
	}
	return filepath.Join(*p.config.productVariables.DistDir, p.path)
}

func (p DistPath) RelPathString() string {
	return p.path
}

type ModuleSrcPath struct {
	SourcePath
}

var _ Path = ModuleSrcPath{}
var _ genPathProvider = ModuleSrcPath{}
var _ objPathProvider = ModuleSrcPath{}
var _ resPathProvider = ModuleSrcPath{}

func PathForModuleSrc(ctx ModuleContext, paths ...string) ModuleSrcPath {
	p, err := validatePath(paths...)
	if err != nil {
		reportPathError(ctx, err)
	}

	srcPath, err := pathForSource(ctx, ctx.ModuleDir(), p)
	if err != nil {
		reportPathError(ctx, err)
	}

	path := ModuleSrcPath{srcPath}
	path.basePath.rel = p

	if exists, _, err := ctx.Fs().Exists(path.String()); err != nil {
		reportPathErrorf(ctx, "%s: %s", path, err.Error())
	} else if !exists {
		reportPathErrorf(ctx, "module source path %s does not exist", path)
	}

	return path
}

func OptionalPathForModuleSrc(ctx ModuleContext, p *string) OptionalPath {
	if p == nil {
		return OptionalPath{}
	}
	return OptionalPathForPath(PathForModuleSrc(ctx, *p))
}

func (p ModuleSrcPath) genPathWithExt(ctx ModuleContext, subdir, ext string) ModuleGenPath {
	return PathForModuleGen(ctx, subdir, pathtools.ReplaceExtension(p.path, ext))
}

func (p ModuleSrcPath) objPathWithExt(ctx ModuleContext, subdir, ext string) ModuleObjPath {
	return PathForModuleObj(ctx, subdir, pathtools.ReplaceExtension(p.path, ext))
}

func (p ModuleSrcPath) resPathWithName(ctx ModuleContext, name string) ModuleResPath {
	return PathForModuleRes(ctx, p.path, name)
}

func (p ModuleSrcPath) WithSubDir(ctx ModuleContext, subdir string) ModuleSrcPath {
	subdir = PathForModuleSrc(ctx, subdir).String()
	var err error
	rel, err := filepath.Rel(subdir, p.path)
	if err != nil {
		ctx.ModuleErrorf("source file %q is not under path %q", p.path, subdir)
		return p
	}
	p.rel = rel
	return p
}

type ModuleOutPath struct {
	OutputPath
}

var _ Path = ModuleOutPath{}

func pathForModule(ctx ModuleContext) OutputPath {
	return PathForOutput(ctx, ".intermediates", ctx.ModuleDir(), ctx.ModuleName(), ctx.ModuleSubDir())
}

func PathForVndkRefAbiDump(ctx ModuleContext, version, fileName string, vndkOrNdk, isSourceDump bool) OptionalPath {
	arches := ctx.DeviceConfig().Arches()
	currentArch := ctx.Arch()
	archNameAndVariant := currentArch.ArchType.String()
	if currentArch.ArchVariant != "" {
		archNameAndVariant += "_" + currentArch.ArchVariant
	}
	var sourceOrBinaryDir string
	var vndkOrNdkDir string
	var ext string
	if isSourceDump {
		ext = ".lsdump.gz"
		sourceOrBinaryDir = "source-based"
	} else {
		ext = ".bdump.gz"
		sourceOrBinaryDir = "binary-based"
	}
	if vndkOrNdk {
		vndkOrNdkDir = "vndk"
	} else {
		vndkOrNdkDir = "ndk"
	}
	if len(arches) == 0 {
		panic("device build with no primary arch")
	}
	binderBitness := ctx.DeviceConfig().BinderBitness()
	refDumpFileStr := "prebuilts/abi-dumps/" + vndkOrNdkDir + "/" + version + "/" + binderBitness + "/" +
		archNameAndVariant + "/" + sourceOrBinaryDir + "/" + fileName + ext
	return ExistentPathForSource(ctx, refDumpFileStr)
}

func PathForModuleOut(ctx ModuleContext, paths ...string) ModuleOutPath {
	p, err := validatePath(paths...)
	if err != nil {
		reportPathError(ctx, err)
	}
	return ModuleOutPath{
		OutputPath: pathForModule(ctx).withRel(p),
	}
}

type ModuleGenPath struct {
	ModuleOutPath
}

var _ Path = ModuleGenPath{}
var _ genPathProvider = ModuleGenPath{}
var _ objPathProvider = ModuleGenPath{}

func PathForModuleGen(ctx ModuleContext, paths ...string) ModuleGenPath {
	p, err := validatePath(paths...)
	if err != nil {
		reportPathError(ctx, err)
	}
	return ModuleGenPath{
		ModuleOutPath: ModuleOutPath{
			OutputPath: pathForModule(ctx).withRel("gen").withRel(p),
		},
	}
}

func (p ModuleGenPath) genPathWithExt(ctx ModuleContext, subdir, ext string) ModuleGenPath {
	return PathForModuleGen(ctx, subdir, pathtools.ReplaceExtension(p.path, ext))
}

func (p ModuleGenPath) objPathWithExt(ctx ModuleContext, subdir, ext string) ModuleObjPath {
	return PathForModuleObj(ctx, subdir, pathtools.ReplaceExtension(p.path, ext))
}

type ModuleObjPath struct {
	ModuleOutPath
}

var _ Path = ModuleObjPath{}

func PathForModuleObj(ctx ModuleContext, pathComponents ...string) ModuleObjPath {
	p, err := validatePath(pathComponents...)
	if err != nil {
		reportPathError(ctx, err)
	}
	return ModuleObjPath{PathForModuleOut(ctx, "obj", p)}
}

type ModuleResPath struct {
	ModuleOutPath
}

var _ Path = ModuleResPath{}

func PathForModuleRes(ctx ModuleContext, pathComponents ...string) ModuleResPath {
	p, err := validatePath(pathComponents...)
	if err != nil {
		reportPathError(ctx, err)
	}

	return ModuleResPath{PathForModuleOut(ctx, "res", p)}
}

func PathForModuleInstall(ctx ModuleInstallPathContext, pathComponents ...string) OutputPath {
	var outPaths []string
	if ctx.Device() {
		var partition string
		if ctx.InstallInData() {
			partition = "data"
		} else if ctx.SocSpecific() {
			partition = ctx.DeviceConfig().VendorPath()
		} else if ctx.DeviceSpecific() {
			partition = ctx.DeviceConfig().OdmPath()
		} else if ctx.ProductSpecific() {
			partition = ctx.DeviceConfig().ProductPath()
		} else {
			partition = "system"
		}

		if ctx.InstallInSanitizerDir() {
			partition = "data/asan/" + partition
		}
		outPaths = []string{"target", "product", ctx.Config().DeviceName(), partition}
	} else {
		switch ctx.Os() {
		case Linux:
			outPaths = []string{"host", "linux-x86"}
		case LinuxBionic:
			outPaths = []string{"host", "linux_bionic-x86"}
		default:
			outPaths = []string{"host", ctx.Os().String() + "-x86"}
		}
	}
	if ctx.Debug() {
		outPaths = append([]string{"debug"}, outPaths...)
	}
	outPaths = append(outPaths, pathComponents...)
	return PathForOutput(ctx, outPaths...)
}

func validateSafePath(pathComponents ...string) (string, error) {
	return filepath.Join(pathComponents...), nil
}

func validatePath(pathComponents ...string) (string, error) {
	for _, path := range pathComponents {
		if strings.Contains(path, "$") {
			return "", fmt.Errorf("Path contains invalid character($): %s", path)
		}
	}
	return validateSafePath(pathComponents...)
}

func PathForPhony(ctx PathContext, phony string) WritablePath {
	if strings.ContainsAny(phony, "$/") {
		reportPathErrorf(ctx, "Phony target contains invalid character ($ or /): %s", phony)
	}
	return PhonyPath{basePath{phony, ctx.Config(), ""}}
}

type PhonyPath struct {
	basePath
}

func (p PhonyPath) writablePath() {}

var _ Path = PhonyPath{}
var _ WritablePath = PhonyPath{}

type testPath struct {
	basePath
}

func (p testPath) String() string {
	return p.path
}

func PathForTesting(paths ...string) Path {
	p, err := validateSafePath(paths...)
	if err != nil {
		panic(err)
	}
	return testPath{basePath{path: p, rel: p}}
}

func PathsForTesting(strs []string) Paths {
	p := make(Paths, len(strs))
	for i, s := range strs {
		p[i] = PathForTesting(s)
	}

	return p
}
