package gomosaic

import (
	"os"
	"strings"
)

type PkgPathNormalizer struct {
	modInfo   *ModuleInfo
	outputDir string
}

// Normalize нормализует путь к пакету, когда тип находится в том же месте,
// что и директория сохранения кода тогда путь к пакету не используется.
func (n *PkgPathNormalizer) Normalize(pkgPath string) string {
	packagePath := strings.ReplaceAll(n.outputDir, n.modInfo.Dir, "")
	packagePath = strings.TrimLeft(packagePath, string(os.PathSeparator))
	if strings.EqualFold(n.modInfo.Path+"/"+packagePath, pkgPath) {
		return ""
	}

	return pkgPath
}

func NewPkgPathNormalizer(
	modInfo *ModuleInfo,
	outputDir string,
) *PkgPathNormalizer {
	return &PkgPathNormalizer{
		modInfo:   modInfo,
		outputDir: outputDir,
	}
}
