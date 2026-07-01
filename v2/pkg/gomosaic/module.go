package gomosaic

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

// ModuleInfo содержит информацию о модуле Go.
type ModuleInfo struct {
	Dir       string
	Path      string
	GoVersion string
}

// ParsePath разбирает путь вида "~/pkg/foo.TypeName" на путь пакета и имя типа.
func (m *ModuleInfo) ParsePath(s string) (pkgPath, name string, err error) {
	u, err := url.Parse("//" + s)
	if err != nil || u.Path == "" {
		return "", "", fmt.Errorf("invalid import path: %w", err)
	}

	pkgPath = strings.ReplaceAll(u.Path, name, "")

	name = path.Ext(u.Path)
	if name == "" {
		return "", "", fmt.Errorf("invalid import path: %s, example ~/pkg/foo.ContextKey", s)
	}

	name = name[1:]

	if strings.HasPrefix(u.Host, "~") {
		pkgPath = m.Path + pkgPath
	} else {
		pkgPath = u.Host + pkgPath
	}

	return pkgPath, name, err
}

// LoadModuleInfo загружает информацию о модуле из go.mod.
func LoadModuleInfo(goModPath string) (*ModuleInfo, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, err
	}

	modFile, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, err
	}

	return &ModuleInfo{
		Dir:       filepath.Dir(goModPath),
		Path:      modFile.Module.Mod.Path,
		GoVersion: modFile.Go.Version,
	}, nil
}
