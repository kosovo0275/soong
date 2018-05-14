package cc

import (
	"android/soong/android"
)

func init() {
	android.RegisterModuleType("ndk_headers", ndkHeadersFactory)
	android.RegisterModuleType("ndk_library", ndkLibraryFactory)
	android.RegisterModuleType("preprocessed_ndk_headers", preprocessedNdkHeadersFactory)
	android.RegisterSingletonType("ndk", NdkSingleton)

	pctx.Import("android/soong/common")
}

func getNdkInstallBase(ctx android.PathContext) android.OutputPath {
	return android.PathForOutput(ctx, "ndk")
}

func getNdkSysrootBase(ctx android.PathContext) android.OutputPath {
	return getNdkInstallBase(ctx).Join(ctx, "sysroot")
}

func getNdkBaseTimestampFile(ctx android.PathContext) android.WritablePath {
	return android.PathForOutput(ctx, "ndk_base.timestamp")
}

func getNdkFullTimestampFile(ctx android.PathContext) android.WritablePath {
	return android.PathForOutput(ctx, "ndk.timestamp")
}

func NdkSingleton() android.Singleton {
	return &ndkSingleton{}
}

type ndkSingleton struct{}

func (n *ndkSingleton) GenerateBuildActions(ctx android.SingletonContext) {
	var staticLibInstallPaths android.Paths
	var installPaths android.Paths
	var licensePaths android.Paths
	ctx.VisitAllModules(func(module android.Module) {
		if m, ok := module.(android.Module); ok && !m.Enabled() {
			return
		}

		if m, ok := module.(*headerModule); ok {
			installPaths = append(installPaths, m.installPaths...)
			licensePaths = append(licensePaths, m.licensePath)
		}

		if m, ok := module.(*preprocessedHeaderModule); ok {
			installPaths = append(installPaths, m.installPaths...)
			licensePaths = append(licensePaths, m.licensePath)
		}

		if m, ok := module.(*Module); ok {
			if installer, ok := m.installer.(*stubDecorator); ok {
				installPaths = append(installPaths, installer.installPath)
			}

			if library, ok := m.linker.(*libraryDecorator); ok {
				if library.ndkSysrootPath != nil {
					staticLibInstallPaths = append(
						staticLibInstallPaths, library.ndkSysrootPath)
				}
			}
		}
	})

	combinedLicense := getNdkInstallBase(ctx).Join(ctx, "NOTICE")
	ctx.Build(pctx, android.BuildParams{
		Rule:        android.Cat,
		Description: "combine licenses",
		Output:      combinedLicense,
		Inputs:      licensePaths,
	})

	baseDepPaths := append(installPaths, combinedLicense)

	ctx.Build(pctx, android.BuildParams{
		Rule:      android.Touch,
		Output:    getNdkBaseTimestampFile(ctx),
		Implicits: baseDepPaths,
	})

	fullDepPaths := append(staticLibInstallPaths, getNdkBaseTimestampFile(ctx))

	ctx.Build(pctx, android.BuildParams{
		Rule:      android.Touch,
		Output:    getNdkFullTimestampFile(ctx),
		Implicits: fullDepPaths,
	})
}
