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

// ModuleInfo информация о модуле Go
type ModuleInfo struct {
	Dir       string // Дирректория модуля (например, "/Users/vasiya/project")
	Path      string // Путь модуля (например, "github.com/user/project")
	GoVersion string // Версия Go
}

func (m *ModuleInfo) ParsePath(s string) (pkgPath, name string, err error) {
	u, err := url.Parse("//" + s)
	if err != nil || u.Path == "" {
		return "", "", fmt.Errorf("invalid import path: %w", err)
	}
	name = path.Ext(u.Path)
	pkgPath = strings.ReplaceAll(u.Path, name, "")
	if name == "" {
		return "", "", fmt.Errorf("invalid import path: %s, example ~/pkg/foo.ContextKey", s)
	}
	name = name[1:]
	if strings.HasPrefix(u.Host, "~") {
		pkgPath = m.Path + pkgPath
	} else {
		pkgPath = u.Host + pkgPath
	}
	return
}

// LoadModuleInfo загружает информацию о модуле из go.mod
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
