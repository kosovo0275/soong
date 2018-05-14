package python

import (
	"path/filepath"

	"android/soong/android"
)

// This file handles installing python executables into their final location

type installLocation int

const (
	InstallInData installLocation = iota
)

type pythonInstaller struct {
	dir      string
	dir64    string
	relative string

	path android.OutputPath
}

func NewPythonInstaller(dir, dir64 string) *pythonInstaller {
	return &pythonInstaller{
		dir:   dir,
		dir64: dir64,
	}
}

var _ installer = (*pythonInstaller)(nil)

func (installer *pythonInstaller) installDir(ctx android.ModuleContext) android.OutputPath {
	dir := installer.dir
	if ctx.Arch().ArchType.Multilib == "lib64" && installer.dir64 != "" {
		dir = installer.dir64
	}
	if !ctx.Host() && !ctx.Arch().Native {
		dir = filepath.Join(dir, ctx.Arch().ArchType.String())
	}
	return android.PathForModuleInstall(ctx, dir, installer.relative)
}

func (installer *pythonInstaller) install(ctx android.ModuleContext, file android.Path) {
	installer.path = ctx.InstallFile(installer.installDir(ctx), file.Base(), file)
}
