package python

import (
	"android/soong/android"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

type subAndroidMkProvider interface {
	AndroidMk(*Module, *android.AndroidMkData)
}

func (p *Module) subAndroidMk(data *android.AndroidMkData, obj interface{}) {
	if p.subAndroidMkOnce == nil {
		p.subAndroidMkOnce = make(map[subAndroidMkProvider]bool)
	}
	if androidmk, ok := obj.(subAndroidMkProvider); ok {
		if !p.subAndroidMkOnce[androidmk] {
			p.subAndroidMkOnce[androidmk] = true
			androidmk.AndroidMk(p, data)
		}
	}
}

func (p *Module) AndroidMk() android.AndroidMkData {
	ret := android.AndroidMkData{OutputFile: p.installSource}

	p.subAndroidMk(&ret, p.installer)

	return ret
}

func (p *binaryDecorator) AndroidMk(base *Module, ret *android.AndroidMkData) {
	ret.Class = "EXECUTABLES"

	ret.Extra = append(ret.Extra, func(w io.Writer, outputFile android.Path) {
		if len(p.binaryProperties.Test_suites) > 0 {
			fmt.Fprintln(w, "LOCAL_COMPATIBILITY_SUITE :=",
				strings.Join(p.binaryProperties.Test_suites, " "))
		}
	})
	base.subAndroidMk(ret, p.pythonInstaller)
}

func (p *testDecorator) AndroidMk(base *Module, ret *android.AndroidMkData) {
	ret.Class = "NATIVE_TESTS"

	ret.Extra = append(ret.Extra, func(w io.Writer, outputFile android.Path) {
		if len(p.binaryDecorator.binaryProperties.Test_suites) > 0 {
			fmt.Fprintln(w, "LOCAL_COMPATIBILITY_SUITE :=",
				strings.Join(p.binaryDecorator.binaryProperties.Test_suites, " "))
		}
	})
	base.subAndroidMk(ret, p.binaryDecorator.pythonInstaller)
}

func (installer *pythonInstaller) AndroidMk(base *Module, ret *android.AndroidMkData) {
	// Soong installation is only supported for host modules. Have Make
	// installation trigger Soong installation.
	if base.Target().Os.Class == android.Host {
		ret.OutputFile = android.OptionalPathForPath(installer.path)
	}

	ret.Extra = append(ret.Extra, func(w io.Writer, outputFile android.Path) {
		path := installer.path.RelPathString()
		dir, file := filepath.Split(path)
		stem := strings.TrimSuffix(file, filepath.Ext(file))

		fmt.Fprintln(w, "LOCAL_MODULE_SUFFIX := "+filepath.Ext(file))
		fmt.Fprintln(w, "LOCAL_MODULE_PATH := $(OUT_DIR)/"+filepath.Clean(dir))
		fmt.Fprintln(w, "LOCAL_MODULE_STEM := "+stem)
	})
}
