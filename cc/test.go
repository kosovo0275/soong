package cc

import (
	"path/filepath"
	"strings"

	"android/soong/android"
)

type TestProperties struct {
	Gtest *bool
}

type TestBinaryProperties struct {
	Test_per_src *bool

	No_named_install_directory *bool

	Data []string

	Test_suites []string `android:"arch_variant"`
}

func init() {
	android.RegisterModuleType("cc_test", TestFactory)
	android.RegisterModuleType("cc_test_library", TestLibraryFactory)
	android.RegisterModuleType("cc_benchmark", BenchmarkFactory)
	android.RegisterModuleType("cc_test_host", TestHostFactory)
	android.RegisterModuleType("cc_benchmark_host", BenchmarkHostFactory)
}

func TestFactory() android.Module {
	module := NewTest(android.HostAndDeviceSupported)
	return module.Init()
}

func TestLibraryFactory() android.Module {
	module := NewTestLibrary(android.HostAndDeviceSupported)
	return module.Init()
}

func BenchmarkFactory() android.Module {
	module := NewBenchmark(android.HostAndDeviceSupported)
	return module.Init()
}

func TestHostFactory() android.Module {
	module := NewTest(android.HostSupported)
	return module.Init()
}

func BenchmarkHostFactory() android.Module {
	module := NewBenchmark(android.HostSupported)
	return module.Init()
}

type testPerSrc interface {
	testPerSrc() bool
	srcs() []string
	setSrc(string, string)
}

func (test *testBinary) testPerSrc() bool {
	return Bool(test.Properties.Test_per_src)
}

func (test *testBinary) srcs() []string {
	return test.baseCompiler.Properties.Srcs
}

func (test *testBinary) setSrc(name, src string) {
	test.baseCompiler.Properties.Srcs = []string{src}
	test.binaryDecorator.Properties.Stem = StringPtr(name)
}

var _ testPerSrc = (*testBinary)(nil)

func testPerSrcMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok {
		if test, ok := m.linker.(testPerSrc); ok {
			if test.testPerSrc() && len(test.srcs()) > 0 {
				testNames := make([]string, len(test.srcs()))
				for i, src := range test.srcs() {
					testNames[i] = strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
				}
				tests := mctx.CreateLocalVariations(testNames...)
				for i, src := range test.srcs() {
					tests[i].(*Module).linker.(testPerSrc).setSrc(testNames[i], src)
				}
			}
		}
	}
}

type testDecorator struct {
	Properties TestProperties
	linker     *baseLinker
}

func (test *testDecorator) gtest() bool {
	return BoolDefault(test.Properties.Gtest, true)
}

func (test *testDecorator) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	if !test.gtest() {
		return flags
	}

	flags.CFlags = append(flags.CFlags, "-DGTEST_HAS_STD_STRING", "-O0", "-g", "-DGTEST_OS_LINUX_ANDROID")
	return flags
}

func (test *testDecorator) linkerDeps(ctx BaseModuleContext, deps Deps) Deps {
	if test.gtest() {
		deps.StaticLibs = append(deps.StaticLibs, "libgtest_main", "libgtest")
	}
	return deps
}

func (test *testDecorator) linkerInit(ctx BaseModuleContext, linker *baseLinker) {
	runpath := "../../lib"
	if ctx.toolchain().Is64Bit() {
		runpath += "64"
	}
	linker.dynamicProperties.RunPaths = append(linker.dynamicProperties.RunPaths, runpath)
	linker.dynamicProperties.RunPaths = append(linker.dynamicProperties.RunPaths, "")
}

func (test *testDecorator) linkerProps() []interface{} {
	return []interface{}{&test.Properties}
}

func NewTestInstaller() *baseInstaller {
	return NewBaseInstaller("nativetest", "nativetest64", InstallInData)
}

type testBinary struct {
	testDecorator
	*binaryDecorator
	*baseCompiler
	Properties TestBinaryProperties
	data       android.Paths
}

func (test *testBinary) linkerProps() []interface{} {
	props := append(test.testDecorator.linkerProps(), test.binaryDecorator.linkerProps()...)
	props = append(props, &test.Properties)
	return props
}

func (test *testBinary) linkerInit(ctx BaseModuleContext) {
	test.testDecorator.linkerInit(ctx, test.binaryDecorator.baseLinker)
	test.binaryDecorator.linkerInit(ctx)
}

func (test *testBinary) linkerDeps(ctx DepsContext, deps Deps) Deps {
	android.ExtractSourcesDeps(ctx, test.Properties.Data)

	deps = test.testDecorator.linkerDeps(ctx, deps)
	deps = test.binaryDecorator.linkerDeps(ctx, deps)
	return deps
}

func (test *testBinary) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	flags = test.binaryDecorator.linkerFlags(ctx, flags)
	flags = test.testDecorator.linkerFlags(ctx, flags)
	return flags
}

func (test *testBinary) install(ctx ModuleContext, file android.Path) {
	test.data = ctx.ExpandSources(test.Properties.Data, nil)

	test.binaryDecorator.baseInstaller.dir = "nativetest"
	test.binaryDecorator.baseInstaller.dir64 = "nativetest64"

	if !Bool(test.Properties.No_named_install_directory) {
		test.binaryDecorator.baseInstaller.relative = ctx.ModuleName()
	} else if String(test.binaryDecorator.baseInstaller.Properties.Relative_install_path) == "" {
		ctx.PropertyErrorf("no_named_install_directory", "Module install directory may only be disabled if relative_install_path is set")
	}

	test.binaryDecorator.baseInstaller.install(ctx, file)
}

func NewTest(hod android.HostOrDeviceSupported) *Module {
	module, binary := NewBinary(hod)
	module.multilib = android.MultilibBoth
	binary.baseInstaller = NewTestInstaller()

	test := &testBinary{
		testDecorator: testDecorator{
			linker: binary.baseLinker,
		},
		binaryDecorator: binary,
		baseCompiler:    NewBaseCompiler(),
	}
	module.compiler = test
	module.linker = test
	module.installer = test
	return module
}

type testLibrary struct {
	testDecorator
	*libraryDecorator
}

func (test *testLibrary) linkerProps() []interface{} {
	return append(test.testDecorator.linkerProps(), test.libraryDecorator.linkerProps()...)
}

func (test *testLibrary) linkerInit(ctx BaseModuleContext) {
	test.testDecorator.linkerInit(ctx, test.libraryDecorator.baseLinker)
	test.libraryDecorator.linkerInit(ctx)
}

func (test *testLibrary) linkerDeps(ctx DepsContext, deps Deps) Deps {
	deps = test.testDecorator.linkerDeps(ctx, deps)
	deps = test.libraryDecorator.linkerDeps(ctx, deps)
	return deps
}

func (test *testLibrary) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	flags = test.libraryDecorator.linkerFlags(ctx, flags)
	flags = test.testDecorator.linkerFlags(ctx, flags)
	return flags
}

func NewTestLibrary(hod android.HostOrDeviceSupported) *Module {
	module, library := NewLibrary(android.HostAndDeviceSupported)
	library.baseInstaller = NewTestInstaller()
	test := &testLibrary{
		testDecorator: testDecorator{
			linker: library.baseLinker,
		},
		libraryDecorator: library,
	}
	module.linker = test
	return module
}

type BenchmarkProperties struct {
	Data []string

	Test_suites []string
}

type benchmarkDecorator struct {
	*binaryDecorator
	Properties BenchmarkProperties
	data       android.Paths
}

func (benchmark *benchmarkDecorator) linkerInit(ctx BaseModuleContext) {
	runpath := "../../lib"
	if ctx.toolchain().Is64Bit() {
		runpath += "64"
	}
	benchmark.baseLinker.dynamicProperties.RunPaths = append(benchmark.baseLinker.dynamicProperties.RunPaths, runpath)
	benchmark.binaryDecorator.linkerInit(ctx)
}

func (benchmark *benchmarkDecorator) linkerProps() []interface{} {
	props := benchmark.binaryDecorator.linkerProps()
	props = append(props, &benchmark.Properties)
	return props
}

func (benchmark *benchmarkDecorator) linkerDeps(ctx DepsContext, deps Deps) Deps {
	android.ExtractSourcesDeps(ctx, benchmark.Properties.Data)
	deps = benchmark.binaryDecorator.linkerDeps(ctx, deps)
	deps.StaticLibs = append(deps.StaticLibs, "libgoogle-benchmark")
	return deps
}

func (benchmark *benchmarkDecorator) install(ctx ModuleContext, file android.Path) {
	benchmark.data = ctx.ExpandSources(benchmark.Properties.Data, nil)
	benchmark.binaryDecorator.baseInstaller.dir = filepath.Join("benchmarktest", ctx.ModuleName())
	benchmark.binaryDecorator.baseInstaller.dir64 = filepath.Join("benchmarktest64", ctx.ModuleName())
	benchmark.binaryDecorator.baseInstaller.install(ctx, file)
}

func NewBenchmark(hod android.HostOrDeviceSupported) *Module {
	module, binary := NewBinary(hod)
	module.multilib = android.MultilibBoth
	binary.baseInstaller = NewBaseInstaller("benchmarktest", "benchmarktest64", InstallInData)

	benchmark := &benchmarkDecorator{
		binaryDecorator: binary,
	}
	module.linker = benchmark
	module.installer = benchmark
	return module
}
