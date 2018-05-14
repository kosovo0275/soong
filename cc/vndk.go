package cc

import (
	"sort"
	"strings"
	"sync"

	"android/soong/android"
)

type VndkProperties struct {
	Vndk struct {
		Enabled *bool

		Support_system_process *bool

		Extends *string
	}
}

type vndkdep struct {
	Properties VndkProperties
}

func (vndk *vndkdep) props() []interface{} {
	return []interface{}{&vndk.Properties}
}

func (vndk *vndkdep) begin(ctx BaseModuleContext) {}

func (vndk *vndkdep) deps(ctx BaseModuleContext, deps Deps) Deps {
	return deps
}

func (vndk *vndkdep) isVndk() bool {
	return Bool(vndk.Properties.Vndk.Enabled)
}

func (vndk *vndkdep) isVndkSp() bool {
	return Bool(vndk.Properties.Vndk.Support_system_process)
}

func (vndk *vndkdep) isVndkExt() bool {
	return vndk.Properties.Vndk.Extends != nil
}

func (vndk *vndkdep) getVndkExtendsModuleName() string {
	return String(vndk.Properties.Vndk.Extends)
}

func (vndk *vndkdep) typeName() string {
	if !vndk.isVndk() {
		return "native:vendor"
	}
	if !vndk.isVndkExt() {
		if !vndk.isVndkSp() {
			return "native:vendor:vndk"
		}
		return "native:vendor:vndksp"
	}
	if !vndk.isVndkSp() {
		return "native:vendor:vndkext"
	}
	return "native:vendor:vndkspext"
}

func (vndk *vndkdep) vndkCheckLinkType(ctx android.ModuleContext, to *Module, tag dependencyTag) {
	if to.linker == nil {
		return
	}
	if !vndk.isVndk() {

		violation := false
		if lib, ok := to.linker.(*llndkStubDecorator); ok && !Bool(lib.Properties.Vendor_available) {
			violation = true
		} else {
			if _, ok := to.linker.(libraryInterface); ok && to.VendorProperties.Vendor_available != nil && !Bool(to.VendorProperties.Vendor_available) {

				violation = true
			}
		}
		if violation {
			ctx.ModuleErrorf("Vendor module that is not VNDK should not link to %q which is marked as `vendor_available: false`", to.Name())
		}
	}
	if lib, ok := to.linker.(*libraryDecorator); !ok || !lib.shared() {

		return
	}
	if !to.Properties.UseVndk {
		ctx.ModuleErrorf("(%s) should not link to %q which is not a vendor-available library",
			vndk.typeName(), to.Name())
		return
	}
	if tag == vndkExtDepTag {

		if to.vndkdep == nil || !to.vndkdep.isVndk() {
			ctx.ModuleErrorf("`extends` refers a non-vndk module %q", to.Name())
			return
		}
		if vndk.isVndkSp() != to.vndkdep.isVndkSp() {
			ctx.ModuleErrorf(
				"`extends` refers a module %q with mismatched support_system_process",
				to.Name())
			return
		}
		if !Bool(to.VendorProperties.Vendor_available) {
			ctx.ModuleErrorf(
				"`extends` refers module %q which does not have `vendor_available: true`",
				to.Name())
			return
		}
	}
	if to.vndkdep == nil {
		return
	}

	if !vndkIsVndkDepAllowed(vndk, to.vndkdep) {
		ctx.ModuleErrorf("(%s) should not link to %q (%s)",
			vndk.typeName(), to.Name(), to.vndkdep.typeName())
		return
	}
}

func vndkIsVndkDepAllowed(from *vndkdep, to *vndkdep) bool {

	if from.isVndkExt() {
		if from.isVndkSp() {

			return to.isVndkSp() || !to.isVndk()
		}

		return true
	}
	if from.isVndk() {
		if to.isVndkExt() {

			return false
		}
		if from.isVndkSp() {

			return to.isVndkSp()
		}

		return to.isVndk()
	}

	return true
}

var (
	vndkCoreLibraries    []string
	vndkSpLibraries      []string
	llndkLibraries       []string
	vndkPrivateLibraries []string
	vndkLibrariesLock    sync.Mutex
)

func vndkMutator(mctx android.BottomUpMutatorContext) {
	if m, ok := mctx.Module().(*Module); ok && m.Enabled() {
		if lib, ok := m.linker.(*llndkStubDecorator); ok {
			vndkLibrariesLock.Lock()
			defer vndkLibrariesLock.Unlock()
			name := strings.TrimSuffix(m.Name(), llndkLibrarySuffix)
			if !inList(name, llndkLibraries) {
				llndkLibraries = append(llndkLibraries, name)
				sort.Strings(llndkLibraries)
			}
			if !Bool(lib.Properties.Vendor_available) {
				if !inList(name, vndkPrivateLibraries) {
					vndkPrivateLibraries = append(vndkPrivateLibraries, name)
					sort.Strings(vndkPrivateLibraries)
				}
			}
		} else {
			lib, is_lib := m.linker.(*libraryDecorator)
			prebuilt_lib, is_prebuilt_lib := m.linker.(*prebuiltLibraryLinker)
			if (is_lib && lib.shared()) || (is_prebuilt_lib && prebuilt_lib.shared()) {
				name := strings.TrimPrefix(m.Name(), "prebuilt_")
				if m.vndkdep.isVndk() && !m.vndkdep.isVndkExt() {
					vndkLibrariesLock.Lock()
					defer vndkLibrariesLock.Unlock()
					if m.vndkdep.isVndkSp() {
						if !inList(name, vndkSpLibraries) {
							vndkSpLibraries = append(vndkSpLibraries, name)
							sort.Strings(vndkSpLibraries)
						}
					} else {
						if !inList(name, vndkCoreLibraries) {
							vndkCoreLibraries = append(vndkCoreLibraries, name)
							sort.Strings(vndkCoreLibraries)
						}
					}
					if !Bool(m.VendorProperties.Vendor_available) {
						if !inList(name, vndkPrivateLibraries) {
							vndkPrivateLibraries = append(vndkPrivateLibraries, name)
							sort.Strings(vndkPrivateLibraries)
						}
					}
				}
			}
		}
	}
}
