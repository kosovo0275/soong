package cc

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"android/soong/android"
	"android/soong/cc/config"
)

var (
	asanCflags  = []string{"-fno-omit-frame-pointer"}
	asanLdflags = []string{"-Wl,-u,__asan_preinit"}
	asanLibs    = []string{"libasan"}

	cfiCflags = []string{"-flto", "-fsanitize-cfi-cross-dso",
		"-fsanitize-blacklist=external/compiler-rt/lib/cfi/cfi_blacklist.txt"}
	cfiLdflags = []string{"-flto", "-fsanitize-cfi-cross-dso", "-fsanitize=cfi",
		"-Wl,-plugin-opt,O1"}
	cfiExportsMapPath  = "build/soong/cc/config/cfi_exports.map"
	cfiStaticLibsMutex sync.Mutex

	intOverflowCflags   = []string{"-fsanitize-blacklist=build/soong/cc/config/integer_overflow_blacklist.txt"}
	minimalRuntimeFlags = []string{"-fsanitize-minimal-runtime", "-fno-sanitize-trap=integer", "-fno-sanitize-recover=integer"}
)

type sanitizerType int

func boolPtr(v bool) *bool {
	if v {
		return &v
	} else {
		return nil
	}
}

const (
	asan sanitizerType = iota + 1
	tsan
	intOverflow
	cfi
)

func (t sanitizerType) String() string {
	switch t {
	case asan:
		return "asan"
	case tsan:
		return "tsan"
	case intOverflow:
		return "intOverflow"
	case cfi:
		return "cfi"
	default:
		panic(fmt.Errorf("unknown sanitizerType %d", t))
	}
}

type SanitizeProperties struct {
	Sanitize struct {
		Never *bool `android:"arch_variant"`

		Address *bool `android:"arch_variant"`
		Thread  *bool `android:"arch_variant"`

		Undefined        *bool    `android:"arch_variant"`
		All_undefined    *bool    `android:"arch_variant"`
		Misc_undefined   []string `android:"arch_variant"`
		Coverage         *bool    `android:"arch_variant"`
		Safestack        *bool    `android:"arch_variant"`
		Cfi              *bool    `android:"arch_variant"`
		Integer_overflow *bool    `android:"arch_variant"`

		Diag struct {
			Undefined        *bool    `android:"arch_variant"`
			Cfi              *bool    `android:"arch_variant"`
			Integer_overflow *bool    `android:"arch_variant"`
			Misc_undefined   []string `android:"arch_variant"`
		}

		Recover []string

		Blacklist *string
	} `android:"arch_variant"`

	SanitizerEnabled  bool `blueprint:"mutated"`
	SanitizeDep       bool `blueprint:"mutated"`
	MinimalRuntimeDep bool `blueprint:"mutated"`
	UbsanRuntimeDep   bool `blueprint:"mutated"`
	InSanitizerDir    bool `blueprint:"mutated"`
}

type sanitize struct {
	Properties SanitizeProperties

	runtimeLibrary          string
	androidMkRuntimeLibrary string
}

func init() {
	android.RegisterMakeVarsProvider(pctx, cfiMakeVarsProvider)
}

func (sanitize *sanitize) props() []interface{} {
	return []interface{}{&sanitize.Properties}
}

func (sanitize *sanitize) begin(ctx BaseModuleContext) {
	s := &sanitize.Properties.Sanitize

	if ctx.useSdk() {
		s.Never = BoolPtr(true)
	}

	if Bool(s.Never) {
		return
	}

	var globalSanitizers []string
	var globalSanitizersDiag []string

	if ctx.clang() {
		//		if ctx.Host() {
		//			globalSanitizers = ctx.Config().SanitizeHost()
		//		} else {
		arches := ctx.Config().SanitizeDeviceArch()
		if len(arches) == 0 || inList(ctx.Arch().ArchType.Name, arches) {
			globalSanitizers = ctx.Config().SanitizeDevice()
			globalSanitizersDiag = ctx.Config().SanitizeDeviceDiag()
		}
		//		}
	}

	if len(globalSanitizers) > 0 {
		var found bool
		if found, globalSanitizers = removeFromList("undefined", globalSanitizers); found && s.All_undefined == nil {
			s.All_undefined = boolPtr(true)
		}

		if found, globalSanitizers = removeFromList("default-ub", globalSanitizers); found && s.Undefined == nil {
			s.Undefined = boolPtr(true)
		}

		if found, globalSanitizers = removeFromList("address", globalSanitizers); found {
			if s.Address == nil {
				s.Address = boolPtr(true)
			} else if *s.Address == false {

				_, globalSanitizers = removeFromList("coverage", globalSanitizers)
			}
		}

		if found, globalSanitizers = removeFromList("thread", globalSanitizers); found && s.Thread == nil {
			s.Thread = boolPtr(true)
		}

		if found, globalSanitizers = removeFromList("coverage", globalSanitizers); found && s.Coverage == nil {
			s.Coverage = boolPtr(true)
		}

		if found, globalSanitizers = removeFromList("safe-stack", globalSanitizers); found && s.Safestack == nil {
			s.Safestack = boolPtr(true)
		}

		if found, globalSanitizers = removeFromList("cfi", globalSanitizers); found && s.Cfi == nil {
			if !ctx.Config().CFIDisabledForPath(ctx.ModuleDir()) {
				s.Cfi = boolPtr(true)
			}
		}

		if found, globalSanitizers = removeFromList("integer_overflow", globalSanitizers); found && s.Integer_overflow == nil {
			if !ctx.Config().IntegerOverflowDisabledForPath(ctx.ModuleDir()) && !ctx.static() {
				s.Integer_overflow = boolPtr(true)
			}
		}

		if len(globalSanitizers) > 0 {
			ctx.ModuleErrorf("unknown global sanitizer option %s", globalSanitizers[0])
		}

		if found, globalSanitizersDiag = removeFromList("integer_overflow", globalSanitizersDiag); found &&
			s.Diag.Integer_overflow == nil && Bool(s.Integer_overflow) && !ctx.static() {
			s.Diag.Integer_overflow = boolPtr(true)
		}

		if found, globalSanitizersDiag = removeFromList("cfi", globalSanitizersDiag); found &&
			s.Diag.Cfi == nil && Bool(s.Cfi) {
			s.Diag.Cfi = boolPtr(true)
		}

		if len(globalSanitizersDiag) > 0 {
			ctx.ModuleErrorf("unknown global sanitizer diagnostics option %s", globalSanitizersDiag[0])
		}
	}

	if s.Cfi == nil && ctx.Config().CFIEnabledForPath(ctx.ModuleDir()) {
		s.Cfi = boolPtr(true)
		if inList("cfi", ctx.Config().SanitizeDeviceDiag()) {
			s.Diag.Cfi = boolPtr(true)
		}
	}

	if !ctx.Config().EnableCFI() || ctx.Arch().ArchType == android.Mips || ctx.Arch().ArchType == android.Mips64 {
		s.Cfi = nil
		s.Diag.Cfi = nil
	}

	if ctx.Arch().ArchType == android.Arm {
		s.Cfi = nil
		s.Diag.Cfi = nil
	}

	if Bool(s.Address) {
		s.Cfi = nil
		s.Diag.Cfi = nil
	}

	if ctx.staticBinary() {
		s.Address = nil
		s.Coverage = nil
		s.Thread = nil
	}

	if Bool(s.All_undefined) {
		s.Undefined = nil
	}

	if !ctx.toolchain().Is64Bit() {

		s.Thread = nil
		s.Safestack = nil

	}

	if Bool(s.All_undefined) || Bool(s.Undefined) || Bool(s.Address) || Bool(s.Thread) || Bool(s.Coverage) || Bool(s.Safestack) || Bool(s.Cfi) || Bool(s.Integer_overflow) || len(s.Misc_undefined) > 0 {
		sanitize.Properties.SanitizerEnabled = true
	}

	if Bool(s.Coverage) {
		if !Bool(s.Address) {
			ctx.ModuleErrorf(`Use of "coverage" also requires "address"`)
		}
	}
}

func (sanitize *sanitize) deps(ctx BaseModuleContext, deps Deps) Deps {
	if !sanitize.Properties.SanitizerEnabled {
		return deps
	}

	if ctx.Device() {
		if Bool(sanitize.Properties.Sanitize.Address) {
			deps.StaticLibs = append(deps.StaticLibs, asanLibs...)
		}
	}

	return deps
}

func (sanitize *sanitize) flags(ctx ModuleContext, flags Flags) Flags {
	minimalRuntimePath := "${config.ClangAsanLibDir}/" + config.UndefinedBehaviorSanitizerMinimalRuntimeLibrary(ctx.toolchain()) + ".a"

	if sanitize.Properties.MinimalRuntimeDep {
		flags.LdFlags = append(flags.LdFlags, minimalRuntimePath)
	}
	if !sanitize.Properties.SanitizerEnabled && !sanitize.Properties.UbsanRuntimeDep {
		return flags
	}

	if !ctx.clang() {
		ctx.ModuleErrorf("Use of sanitizers requires clang")
	}

	var sanitizers []string
	var diagSanitizers []string

	if Bool(sanitize.Properties.Sanitize.All_undefined) {
		sanitizers = append(sanitizers, "undefined")
	} else {
		if Bool(sanitize.Properties.Sanitize.Undefined) {
			sanitizers = append(sanitizers,
				"bool",
				"integer-divide-by-zero",
				"return",
				"returns-nonnull-attribute",
				"shift-exponent",
				"unreachable",
				"vla-bound",
			)
		}
		sanitizers = append(sanitizers, sanitize.Properties.Sanitize.Misc_undefined...)
	}

	if Bool(sanitize.Properties.Sanitize.Diag.Undefined) {
		diagSanitizers = append(diagSanitizers, "undefined")
	}

	diagSanitizers = append(diagSanitizers, sanitize.Properties.Sanitize.Diag.Misc_undefined...)

	if Bool(sanitize.Properties.Sanitize.Address) {
		flags.CFlags = append(flags.CFlags, asanCflags...)
		flags.LdFlags = append(flags.LdFlags, asanLdflags...)

		//		if ctx.Host() {
		flags.LdFlags = append(flags.LdFlags, "-Wl,--no-as-needed")
		//		} else {
		//			flags.CFlags = append(flags.CFlags, "-mllvm", "-asan-globals=0")
		//			flags.DynamicLinker = "/system/bin/linker_asan64"
		//		}
		sanitizers = append(sanitizers, "address")
		diagSanitizers = append(diagSanitizers, "address")
	}

	if Bool(sanitize.Properties.Sanitize.Thread) {
		sanitizers = append(sanitizers, "thread")
	}

	if Bool(sanitize.Properties.Sanitize.Coverage) {
		flags.CFlags = append(flags.CFlags, "-fsanitize-coverage=trace-pc-guard,indirect-calls,trace-cmp")
	}

	if Bool(sanitize.Properties.Sanitize.Safestack) {
		sanitizers = append(sanitizers, "safe-stack")
	}

	if Bool(sanitize.Properties.Sanitize.Cfi) {
		if ctx.Arch().ArchType == android.Arm {

			flags.RequiredInstructionSet = "thumb"
		}
		sanitizers = append(sanitizers, "cfi")

		flags.CFlags = append(flags.CFlags, cfiCflags...)

		if !inList("-fvisibility=hidden", flags.CFlags) {
			flags.CFlags = append(flags.CFlags, "-fvisibility=default")
		}
		flags.LdFlags = append(flags.LdFlags, cfiLdflags...)
		flags.ArGoldPlugin = true
		if Bool(sanitize.Properties.Sanitize.Diag.Cfi) {
			diagSanitizers = append(diagSanitizers, "cfi")
		}

		if ctx.staticBinary() {
			_, flags.CFlags = removeFromList("-fsanitize-cfi-cross-dso", flags.CFlags)
			_, flags.LdFlags = removeFromList("-fsanitize-cfi-cross-dso", flags.LdFlags)
		}
	}

	if Bool(sanitize.Properties.Sanitize.Integer_overflow) {
		sanitizers = append(sanitizers, "unsigned-integer-overflow")
		sanitizers = append(sanitizers, "signed-integer-overflow")
		flags.CFlags = append(flags.CFlags, intOverflowCflags...)
		if Bool(sanitize.Properties.Sanitize.Diag.Integer_overflow) {
			diagSanitizers = append(diagSanitizers, "unsigned-integer-overflow")
			diagSanitizers = append(diagSanitizers, "signed-integer-overflow")
		}
	}

	if len(sanitizers) > 0 {
		sanitizeArg := "-fsanitize=" + strings.Join(sanitizers, ",")

		flags.CFlags = append(flags.CFlags, sanitizeArg)
		//		if ctx.Host() {
		//			flags.CFlags = append(flags.CFlags, "-fno-sanitize-recover=all")
		//			flags.LdFlags = append(flags.LdFlags, sanitizeArg)

		//			_, flags.LdFlags = removeFromList("-Wl,--no-undefined", flags.LdFlags)
		//		} else {
		flags.CFlags = append(flags.CFlags, "-fsanitize-trap=all", "-ftrap-function=abort")

		if enableMinimalRuntime(sanitize) {
			flags.CFlags = append(flags.CFlags, strings.Join(minimalRuntimeFlags, " "))
			flags.libFlags = append([]string{minimalRuntimePath}, flags.libFlags...)
		}
		//		}
	}

	if len(diagSanitizers) > 0 {
		flags.CFlags = append(flags.CFlags, "-fno-sanitize-trap="+strings.Join(diagSanitizers, ","))
	}

	if sanitize.Properties.Sanitize.Recover != nil {
		flags.CFlags = append(flags.CFlags, "-fsanitize-recover="+
			strings.Join(sanitize.Properties.Sanitize.Recover, ","))
	}

	runtimeLibrary := ""
	if Bool(sanitize.Properties.Sanitize.Address) {
		runtimeLibrary = config.AddressSanitizerRuntimeLibrary(ctx.toolchain())
	} else if Bool(sanitize.Properties.Sanitize.Thread) {
		runtimeLibrary = config.ThreadSanitizerRuntimeLibrary(ctx.toolchain())
	} else if len(diagSanitizers) > 0 || sanitize.Properties.UbsanRuntimeDep {
		runtimeLibrary = config.UndefinedBehaviorSanitizerRuntimeLibrary(ctx.toolchain())
	}

	if runtimeLibrary != "" {
		runtimeLibraryPath := "${config.ClangAsanLibDir}/" + runtimeLibrary
		if !ctx.static() {
			runtimeLibraryPath = runtimeLibraryPath + ctx.toolchain().ShlibSuffix()
		} else {
			runtimeLibraryPath = runtimeLibraryPath + ".a"
		}

		flags.libFlags = append([]string{runtimeLibraryPath}, flags.libFlags...)
		sanitize.runtimeLibrary = runtimeLibrary

		sanitize.androidMkRuntimeLibrary = sanitize.runtimeLibrary
		if ctx.useVndk() {
			sanitize.androidMkRuntimeLibrary = sanitize.runtimeLibrary + vendorSuffix
		}
	}

	blacklist := android.OptionalPathForModuleSrc(ctx, sanitize.Properties.Sanitize.Blacklist)
	if blacklist.Valid() {
		flags.CFlags = append(flags.CFlags, "-fsanitize-blacklist="+blacklist.String())
		flags.CFlagsDeps = append(flags.CFlagsDeps, blacklist.Path())
	}

	return flags
}

func (sanitize *sanitize) AndroidMk(ctx AndroidMkContext, ret *android.AndroidMkData) {
	ret.Extra = append(ret.Extra, func(w io.Writer, outputFile android.Path) {
		if sanitize.androidMkRuntimeLibrary != "" {
			fmt.Fprintln(w, "LOCAL_SHARED_LIBRARIES += "+sanitize.androidMkRuntimeLibrary)
		}
	})

	if ret.Class == "STATIC_LIBRARIES" && Bool(sanitize.Properties.Sanitize.Cfi) {
		ret.SubName += ".cfi"
	}
}

func (sanitize *sanitize) inSanitizerDir() bool {
	return sanitize.Properties.InSanitizerDir
}

func (sanitize *sanitize) getSanitizerBoolPtr(t sanitizerType) *bool {
	switch t {
	case asan:
		return sanitize.Properties.Sanitize.Address
	case tsan:
		return sanitize.Properties.Sanitize.Thread
	case intOverflow:
		return sanitize.Properties.Sanitize.Integer_overflow
	case cfi:
		return sanitize.Properties.Sanitize.Cfi
	default:
		panic(fmt.Errorf("unknown sanitizerType %d", t))
	}
}

func (sanitize *sanitize) isUnsanitizedVariant() bool {
	return !sanitize.isSanitizerEnabled(asan) &&
		!sanitize.isSanitizerEnabled(tsan) &&
		!sanitize.isSanitizerEnabled(cfi)
}

func (sanitize *sanitize) SetSanitizer(t sanitizerType, b bool) {
	switch t {
	case asan:
		sanitize.Properties.Sanitize.Address = boolPtr(b)
		if !b {
			sanitize.Properties.Sanitize.Coverage = nil
		}
	case tsan:
		sanitize.Properties.Sanitize.Thread = boolPtr(b)
	case intOverflow:
		sanitize.Properties.Sanitize.Integer_overflow = boolPtr(b)
	case cfi:
		sanitize.Properties.Sanitize.Cfi = boolPtr(b)
		sanitize.Properties.Sanitize.Diag.Cfi = boolPtr(b)
	default:
		panic(fmt.Errorf("unknown sanitizerType %d", t))
	}
	if b {
		sanitize.Properties.SanitizerEnabled = true
	}
}

func (sanitize *sanitize) isSanitizerExplicitlyDisabled(t sanitizerType) bool {
	if sanitize == nil {
		return false
	}

	sanitizerVal := sanitize.getSanitizerBoolPtr(t)
	return sanitizerVal != nil && *sanitizerVal == false
}

func (sanitize *sanitize) isSanitizerEnabled(t sanitizerType) bool {
	if sanitize == nil {
		return false
	}

	sanitizerVal := sanitize.getSanitizerBoolPtr(t)
	return sanitizerVal != nil && *sanitizerVal == true
}

func sanitizerDepsMutator(t sanitizerType) func(android.TopDownMutatorContext) {
	return func(mctx android.TopDownMutatorContext) {
		if c, ok := mctx.Module().(*Module); ok && c.sanitize.isSanitizerEnabled(t) {
			mctx.VisitDepsDepthFirst(func(module android.Module) {
				if d, ok := module.(*Module); ok && d.sanitize != nil &&
					!Bool(d.sanitize.Properties.Sanitize.Never) &&
					!d.sanitize.isSanitizerExplicitlyDisabled(t) {
					if (t == cfi && d.static()) || t != cfi {
						d.sanitize.Properties.SanitizeDep = true
					}
				}
			})
		}
	}
}

func sanitizerRuntimeDepsMutator() func(android.TopDownMutatorContext) {
	return func(mctx android.TopDownMutatorContext) {
		if c, ok := mctx.Module().(*Module); ok && c.sanitize != nil {
			mctx.VisitDepsDepthFirst(func(module android.Module) {
				if d, ok := module.(*Module); ok && d.static() && d.sanitize != nil {

					if enableMinimalRuntime(d.sanitize) {

						c.sanitize.Properties.MinimalRuntimeDep = true
					} else if Bool(d.sanitize.Properties.Sanitize.Diag.Integer_overflow) ||
						len(d.sanitize.Properties.Sanitize.Diag.Misc_undefined) > 0 {

						c.sanitize.Properties.UbsanRuntimeDep = true
					}
				}
			})
		}
	}
}

func sanitizerMutator(t sanitizerType) func(android.BottomUpMutatorContext) {
	return func(mctx android.BottomUpMutatorContext) {
		if c, ok := mctx.Module().(*Module); ok && c.sanitize != nil {
			if c.isDependencyRoot() && c.sanitize.isSanitizerEnabled(t) {
				modules := mctx.CreateVariations(t.String())
				modules[0].(*Module).sanitize.SetSanitizer(t, true)
			} else if c.sanitize.isSanitizerEnabled(t) || c.sanitize.Properties.SanitizeDep {

				isSanitizerEnabled := c.sanitize.isSanitizerEnabled(t)

				modules := mctx.CreateVariations("", t.String())
				modules[0].(*Module).sanitize.SetSanitizer(t, false)
				modules[1].(*Module).sanitize.SetSanitizer(t, true)

				modules[0].(*Module).sanitize.Properties.SanitizeDep = false
				modules[1].(*Module).sanitize.Properties.SanitizeDep = false

				if t == cfi {
					if c.static() {
						if !mctx.Device() {
							if isSanitizerEnabled {
								modules[0].(*Module).Properties.PreventInstall = true
								modules[0].(*Module).Properties.HideFromMake = true
							} else {
								modules[1].(*Module).Properties.PreventInstall = true
								modules[1].(*Module).Properties.HideFromMake = true
							}
						} else {
							cfiStaticLibs := cfiStaticLibs(mctx.Config())

							cfiStaticLibsMutex.Lock()
							*cfiStaticLibs = append(*cfiStaticLibs, c.Name())
							cfiStaticLibsMutex.Unlock()
						}
					} else {
						modules[0].(*Module).Properties.PreventInstall = true
						modules[0].(*Module).Properties.HideFromMake = true
					}
				} else if t == asan {
					if mctx.Device() {

						modules[1].(*Module).sanitize.Properties.InSanitizerDir = true
						modules[1].(*Module).sanitize.SetSanitizer(cfi, false)
					}
					if isSanitizerEnabled {
						modules[0].(*Module).Properties.PreventInstall = true
						modules[0].(*Module).Properties.HideFromMake = true
					} else {
						modules[1].(*Module).Properties.PreventInstall = true
						modules[1].(*Module).Properties.HideFromMake = true
					}
				}
			}
			c.sanitize.Properties.SanitizeDep = false
		}
	}
}

func cfiStaticLibs(config android.Config) *[]string {
	return config.Once("cfiStaticLibs", func() interface{} {
		return &[]string{}
	}).(*[]string)
}

func enableMinimalRuntime(sanitize *sanitize) bool {
	if !Bool(sanitize.Properties.Sanitize.Address) &&
		(Bool(sanitize.Properties.Sanitize.Integer_overflow) ||
			len(sanitize.Properties.Sanitize.Misc_undefined) > 0) &&
		!(Bool(sanitize.Properties.Sanitize.Diag.Integer_overflow) ||
			Bool(sanitize.Properties.Sanitize.Diag.Cfi) ||
			len(sanitize.Properties.Sanitize.Diag.Misc_undefined) > 0) {
		return true
	}
	return false
}

func cfiMakeVarsProvider(ctx android.MakeVarsContext) {
	cfiStaticLibs := cfiStaticLibs(ctx.Config())
	sort.Strings(*cfiStaticLibs)
	ctx.Strict("SOONG_CFI_STATIC_LIBRARIES", strings.Join(*cfiStaticLibs, " "))
}
