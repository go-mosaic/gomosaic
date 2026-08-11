package gomosaic

import (
	"context"
	"testing"

	"github.com/dave/jennifer/jen"
)

type testPlugin struct {
	name string
}

func (p *testPlugin) Name() string { return p.name }
func (p *testPlugin) Generate(_ context.Context, _ *ModuleInfo, _ []*NameTypeInfo) (map[string]File, error) {
	return nil, nil
}

func TestBuilder_NewBuilder(t *testing.T) {
	b := NewBuilder()
	if b == nil {
		t.Fatal("NewBuilder() returned nil")
	}

	if b.pluginReg == nil {
		t.Error("plugin registry не должен быть nil")
	}

	if b.transformReg == nil {
		t.Error("transform registry не должен быть nil")
	}
}

func TestBuilder_WithPlugins(t *testing.T) {
	b := NewBuilder()

	b.WithPlugins(
		&testPlugin{name: "plugin-1"},
		&testPlugin{name: "plugin-2"},
	)

	if len(b.pluginReg.List()) != 2 {
		t.Errorf("должно быть 2 плагина, получено %d", len(b.pluginReg.List()))
	}
}

func TestBuilder_Build(t *testing.T) {
	b := NewBuilder()

	b.WithPlugins(&testPlugin{name: "test"})

	fs := NewFileSystem("test", "./output")

	gen := b.Build(fs)
	if gen == nil {
		t.Fatal("Build() returned nil")
	}
}

func TestBuilder_WithTransformer(t *testing.T) {
	b := NewBuilder()

	called := false

	b.WithTransformer(func() Transformer {
		called = true
		return &testTransformer{}
	})

	if !called {
		t.Error("фабрика трансформера не была вызвана")
	}
}

func TestBuilder_WithTransformRegistry(t *testing.T) {
	customReg := NewTransformRegistry()

	b := NewBuilder(WithTransformRegistry(customReg))

	if b.transformReg != customReg {
		t.Error("кастомный реестр не был установлен")
	}
}

func TestBuilder_ChainedMethods(t *testing.T) {
	b := NewBuilder().
		WithPlugins(&testPlugin{name: "p1"}).
		WithPlugins(&testPlugin{name: "p2"}).
		WithPlugins(&testPlugin{name: "p3"})

	if len(b.pluginReg.List()) != 3 {
		t.Errorf("должно быть 3 плагина, получено %d", len(b.pluginReg.List()))
	}
}

type testTransformer struct{}

func (t *testTransformer) Support(typeInfo *TypeInfo) bool { return false }
func (t *testTransformer) Parse(valueID, assignID jen.Code, typeInfo *TypeInfo, errorStatements []jen.Code, qualFn QualFunc) jen.Code {
	return nil
}
func (t *testTransformer) Format(valueID jen.Code, typeInfo *TypeInfo, qualFn QualFunc) jen.Code {
	return nil
}

func TestTransformRegistry_For(t *testing.T) {
	r := DefaultTransformRegistry()

	tr := r.For(&TypeInfo{
		Name:      "string",
		IsBasic:   true,
		BasicInfo: IsString,
	})

	if tr == nil {
		t.Fatal("For() returned nil")
	}
}

func TestTransformRegistry_Merge(t *testing.T) {
	r1 := NewTransformRegistry()
	r2 := DefaultTransformRegistry()

	r1.Merge(r2)

	tr := r1.For(&TypeInfo{
		Name:      "string",
		IsBasic:   true,
		BasicInfo: IsString,
	})

	code := tr.SetAssignID(jen.Id("result")).SetValueID(jen.Lit("hello")).Parse()
	if code == nil {
		t.Error("Parse() вернул nil после слияния реестров")
	}
}

func TestContextWithOutputDir(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithOutputDir(ctx, "/tmp/output")

	dir := OutputDirFromContext(ctx)
	if dir != "/tmp/output" {
		t.Errorf("OutputDir = %s, want /tmp/output", dir)
	}
}

func TestHasError(t *testing.T) {
	vars := []*VarInfo{
		{Name: "result", IsError: false},
		{Name: "err", IsError: true},
	}

	v, ok := HasError(vars)
	if !ok {
		t.Error("HasError() должен найти ошибку")
	}

	if v.Name != "err" {
		t.Errorf("имя ошибки = %s, want err", v.Name)
	}

	varsNoErr := []*VarInfo{
		{Name: "result", IsError: false},
	}

	_, ok = HasError(varsNoErr)
	if ok {
		t.Error("HasError() не должен находить ошибку")
	}
}
