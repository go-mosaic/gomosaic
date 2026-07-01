package gomosaic

import (
	"os"
	"strings"
)

// PkgPathNormalizer нормализует пути пакетов.
type PkgPathNormalizer struct {
	modInfo   *ModuleInfo
	outputDir string
}

func NewPkgPathNormalizer(modInfo *ModuleInfo, outputDir string) *PkgPathNormalizer {
	return &PkgPathNormalizer{
		modInfo:   modInfo,
		outputDir: outputDir,
	}
}

// Normalize нормализует путь пакета. Если тип находится в той же директории,
// что и директория сохранения, возвращает пустую строку.
func (n *PkgPathNormalizer) Normalize(pkgPath string) string {
	packagePath := strings.ReplaceAll(n.outputDir, n.modInfo.Dir, "")
	packagePath = strings.TrimLeft(packagePath, string(os.PathSeparator))

	if strings.EqualFold(n.modInfo.Path+"/"+packagePath, pkgPath) {
		return ""
	}

	return pkgPath
}
