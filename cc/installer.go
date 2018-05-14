package cc

import (
	"path/filepath"

	"android/soong/android"
)

type InstallerProperties struct {
	Relative_install_path *string `android:"arch_variant"`
}

type installLocation int

const (
	InstallInSystem       installLocation = 0
	InstallInData                         = iota
	InstallInSanitizerDir                 = iota
)

func NewBaseInstaller(dir, dir64 string, location installLocation) *baseInstaller {
	return &baseInstaller{
		dir:      dir,
		dir64:    dir64,
		location: location,
	}
}

type baseInstaller struct {
	Properties InstallerProperties

	dir      string
	dir64    string
	subDir   string
	relative string
	location installLocation

	path android.OutputPath
}

var _ installer = (*baseInstaller)(nil)

func (installer *baseInstaller) installerProps() []interface{} {
	return []interface{}{&installer.Properties}
}

func (installer *baseInstaller) installDir(ctx ModuleContext) android.OutputPath {
	dir := installer.dir
	if ctx.toolchain().Is64Bit() && installer.dir64 != "" {
		dir = installer.dir64
	}
	if !ctx.Host() && !ctx.Arch().Native {
		dir = filepath.Join(dir, ctx.Arch().ArchType.String())
	}
	if installer.location == InstallInData && ctx.useVndk() {
		dir = filepath.Join(dir, "vendor")
	}
	return android.PathForModuleInstall(ctx, dir, installer.subDir,
		String(installer.Properties.Relative_install_path), installer.relative)
}

func (installer *baseInstaller) install(ctx ModuleContext, file android.Path) {
	installer.path = ctx.InstallFile(installer.installDir(ctx), file.Base(), file)
}

func (installer *baseInstaller) inData() bool {
	return installer.location == InstallInData
}

func (installer *baseInstaller) inSanitizerDir() bool {
	return installer.location == InstallInSanitizerDir
}

func (installer *baseInstaller) hostToolPath() android.OptionalPath {
	return android.OptionalPath{}
}
