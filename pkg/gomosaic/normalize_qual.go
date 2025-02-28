package gomosaic

import (
	"os"
	"strings"
)

// NormalizePkgPath проверяет и, если необходимо, корректирует путь к пакету,
// чтобы обеспечить правильное сохранение сгенерированного кода.
// В частности, он учитывает случай, когда тип находится в том же месте,
// что и директория сохранения кода, и в этом случае путь к пакету не используется.
func NormalizePkgPath(modInfo *ModuleInfo, outputDir, pkgPath string) string {
	packagePath := strings.ReplaceAll(outputDir, modInfo.Dir, "")
	packagePath = strings.TrimLeft(packagePath, string(os.PathSeparator))
	if strings.EqualFold(modInfo.Path+"/"+packagePath, pkgPath) {
		return ""
	}

	return pkgPath
}
