// Package golden предоставляет утилиты для тестирования сгенерированного кода
// с использованием эталонов.
//
// эталон-файлы хранятся в fixtures/ относительно тестируемого пакета.
// Для обновления эталонов необходимо использовать флаг -update:
//
//	go test ./... -update
package golden

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mosaic/gomosaic/v2/pkg/gomosaic"
)

var update = flag.Bool("update", false, "обновить golden-файлы")

const (
	dirMode      = 0o755
	fileMode     = 0o600
	maxDiffLines = 20
)

// AssertBytes сравнивает сгенерированные байты с эталоном name.
func AssertBytes(t *testing.T, generated []byte, name string) {
	t.Helper()

	goldenPath := filepath.Join("fixtures", name+".golden")

	if *update {
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, dirMode); err != nil {
			t.Fatalf("не удалось создать директорию: %v", err)
		}

		if err := os.WriteFile(goldenPath, generated, fileMode); err != nil {
			t.Fatalf("не удалось записать golden-файл: %v", err)
		}

		t.Logf("обновлён golden-файл: %s", goldenPath)

		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("не удалось прочитать golden-файл %s: %v\nИспользуйте -update для создания", goldenPath, err)
	}

	if !bytes.Equal(generated, expected) {
		diff := simpleDiff(string(expected), string(generated))
		t.Errorf("код отличается от эталона:\n%s", diff)
	}
}

// AssertFile рендерит GoFile и сравнивает результат с эталоном name.
func AssertFile(t *testing.T, f *gomosaic.GoFile, name string) {
	t.Helper()

	var buf bytes.Buffer

	if err := f.Render(&buf, "test"); err != nil {
		t.Fatalf("ошибка генерации: %v", err)
	}

	generated := buf.Bytes()

	goldenPath := filepath.Join("fixtures", name+".golden")

	if *update {
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, dirMode); err != nil {
			t.Fatalf("не удалось создать директорию: %v", err)
		}

		if err := os.WriteFile(goldenPath, generated, fileMode); err != nil {
			t.Fatalf("не удалось записать golden-файл: %v", err)
		}

		t.Logf("обновлён golden-файл: %s", goldenPath)

		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("не удалось прочитать golden-файл %s: %v\nИспользуйте -update для создания", goldenPath, err)
	}

	if !bytes.Equal(generated, expected) {
		diff := simpleDiff(string(expected), string(generated))
		t.Errorf("код отличается от эталона:\n%s", diff)
	}
}

// ParseAndGenerate загружает модуль и разбирает типы из goFile.
func ParseAndGenerate(t *testing.T, dir, goFile string) (*gomosaic.ModuleInfo, []*gomosaic.NameTypeInfo) {
	t.Helper()

	absPath, err := filepath.Abs(filepath.Join(dir, goFile))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}

	modDir := filepath.Dir(absPath)
	for {
		if _, err := os.Stat(filepath.Join(modDir, "go.mod")); err == nil {
			break
		}

		parent := filepath.Dir(modDir)
		if parent == modDir {
			t.Fatalf("не найден go.mod начиная с %s", filepath.Dir(absPath))
		}

		modDir = parent
	}

	module, err := gomosaic.LoadModuleInfo(filepath.Join(modDir, "go.mod"))
	if err != nil {
		t.Fatalf("LoadModuleInfo: %v", err)
	}

	testDataDir := filepath.Dir(absPath)
	types, err := gomosaic.ParsePackage(testDataDir, []string{testDataDir})
	if err != nil {
		t.Fatalf("ParseDir(%s): %v", testDataDir, err)
	}

	if len(types) == 0 {
		t.Fatalf("ParseDir не нашёл типы в %s", testDataDir)
	}

	relPath, _ := filepath.Rel(modDir, testDataDir)
	for _, nt := range types {
		if nt.Package != nil && nt.Package.Path == "" {
			nt.Package.Path = filepath.Join(module.Path, relPath)
		}
	}

	return module, types
}

func simpleDiff(expected, actual string) string {
	expLines := strings.Split(expected, "\n")
	actLines := strings.Split(actual, "\n")

	var diff strings.Builder

	maxLen := max(len(actLines), len(expLines))

	diffs := 0
	for i := range maxLen {
		var expLine, actLine string

		if i < len(expLines) {
			expLine = expLines[i]
		}

		if i < len(actLines) {
			actLine = actLines[i]
		}

		if expLine != actLine {
			if diffs < maxDiffLines {
				fmt.Fprintf(&diff, "  line %d:\n", i+1)

				if expLine != "" {
					fmt.Fprintf(&diff, "    - %s\n", expLine)
				}

				if actLine != "" {
					fmt.Fprintf(&diff, "    + %s\n", actLine)
				}
			}

			diffs++
		}
	}

	if diffs > maxDiffLines {
		fmt.Fprintf(&diff, "  ... и ещё %d отличий\n", diffs-maxDiffLines)
	}

	return diff.String()
}
