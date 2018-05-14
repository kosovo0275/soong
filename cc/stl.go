package cc

import (
	"android/soong/android"
	"fmt"
)

func getNdkStlFamily(ctx android.ModuleContext, m *Module) string {
	stl := m.stl.Properties.SelectedStl
	switch stl {
	case "ndk_libc++_shared", "ndk_libc++_static":
		return "libc++"
	case "ndk_system":
		return "system"
	case "libc++_static":
		return stl
	case "":
		return "none"
	default:
		ctx.ModuleErrorf("stl: %q is not a valid STL", stl)
		return ""
	}
}

type StlProperties struct {
	Stl         *string `android:"arch_variant"`
	SelectedStl string  `blueprint:"mutated"`
}

type stl struct {
	Properties StlProperties
}

func (stl *stl) props() []interface{} {
	return []interface{}{&stl.Properties}
}

func (stl *stl) begin(ctx BaseModuleContext) {
	stl.Properties.SelectedStl = func() string {
		s := ""
		if stl.Properties.Stl != nil {
			s = *stl.Properties.Stl
		}
		switch s {
		case "libc++", "libc++_static":
			return s
		case "c++_shared":
			return "libc++"
		case "c++_static":
			return "lib" + s
		case "none":
			return ""
		case "":
			if ctx.static() {
				return "libc++_static"
			} else {
				return "libc++"
			}
		default:
			ctx.ModuleErrorf("stl: %q is not a supported STL", s)
			return ""
		}
	}()
}

func (stl *stl) deps(ctx BaseModuleContext, deps Deps) Deps {
	return deps
}

func (stl *stl) flags(ctx ModuleContext, flags Flags) Flags {
	flags.LdFlags = append(flags.LdFlags, ldDirs)
	switch stl.Properties.SelectedStl {
	case "libc++", "libc++_static":
		flags.CFlags = append(flags.CFlags, "-D_USING_LIBCXX")
		if ctx.staticBinary() {
			flags.LdFlags = append(flags.LdFlags, "-l:libc++_static.a")
			flags.LdFlags = append(flags.LdFlags, hostStaticGccLibs[ctx.Os()]...)
		} else {
			flags.LdFlags = append(flags.LdFlags, "-l:libc++.so")
			flags.LdFlags = append(flags.LdFlags, hostDynamicGccLibs[ctx.Os()]...)
		}
	case "ndk_system":
		ndkSrcRoot := android.PathForSource(ctx, "prebuilts/ndk/current/sources/cxx-stl/system/include")
		flags.CFlags = append(flags.CFlags, "-isystem "+ndkSrcRoot.String())
	case "ndk_libc++_shared", "ndk_libc++_static", "libstdc++":
		// Nothing.
	case "":
		if ctx.staticBinary() {
			flags.LdFlags = append(flags.LdFlags, hostStaticGccLibs[ctx.Os()]...)
		} else {
			flags.LdFlags = append(flags.LdFlags, hostDynamicGccLibs[ctx.Os()]...)
		}
	default:
		panic(fmt.Errorf("Unknown stl: %q", stl.Properties.SelectedStl))
	}
	return flags
}

var hostDynamicGccLibs, hostStaticGccLibs map[android.OsType][]string
var ldDirs string

func init() {
	ldDirs = ldDirsToFlags([]string{"prebuilts/ndk/current/sources/cxx-stl/llvm-libc++/libs/arm64-v8a", "/data/data/com.termux/files/usr/lib"})
	hostDynamicGccLibs = map[android.OsType][]string{
		android.Android: []string{"-lgcc", "-ldl", "-lc", "-lgcc", "-ldl"},
		android.Linux:   []string{"-lgcc", "-ldl", "-lc", "-lgcc", "-ldl"},
	}
	hostStaticGccLibs = map[android.OsType][]string{
		android.Android: []string{"-Wl,--start-group", "-lgcc", "-lc", "-Wl,--end-group"},
		android.Linux:   []string{"-Wl,--start-group", "-lgcc", "-lc", "-Wl,--end-group"},
	}
}
