package shared

import (
	"path/filepath"
)

func TempDirForOutDir(outDir string) (tempPath string) {
	return filepath.Join(outDir, ".temp")
}
