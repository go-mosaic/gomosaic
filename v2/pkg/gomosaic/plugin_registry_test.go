package gomosaic

import (
	"testing"
)

func TestNewPluginRegistry(t *testing.T) {
	r := NewPluginRegistry()

	if r == nil {
		t.Fatal("NewPluginRegistry() returned nil")
	}

	if len(r.List()) != 0 {
		t.Error("новый реестр должен быть пустым")
	}
}

func TestPluginRegistry_Register(t *testing.T) {
	r := NewPluginRegistry()

	p := &testPlugin{name: "test-plugin"}

	err := r.Register(p)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if len(r.List()) != 1 {
		t.Error("после регистрации должен быть 1 плагин")
	}

	err = r.Register(p)
	if err == nil {
		t.Error("повторная регистрация должна вернуть ошибку")
	}
}

func TestPluginRegistry_Register_EmptyName(t *testing.T) {
	r := NewPluginRegistry()

	p := &testPlugin{name: ""}
	err := r.Register(p)
	if err == nil {
		t.Error("регистрация плагина с пустым именем должна вернуть ошибку")
	}
}

func TestPluginRegistry_Get(t *testing.T) {
	r := NewPluginRegistry()
	p := &testPlugin{name: "my-plugin"}
	r.MustRegister(p)

	got, err := r.Get("my-plugin")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Name() != "my-plugin" {
		t.Errorf("Get() name = %s, want %s", got.Name(), "my-plugin")
	}

	_, err = r.Get("non-existent")
	if err == nil {
		t.Error("Get() несуществующего плагина должна вернуть ошибку")
	}
}

func TestPluginRegistry_MustRegister(t *testing.T) {
	r := NewPluginRegistry()

	r.MustRegister(&testPlugin{name: "ok"})

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustRegister() при дубликате должна паниковать")
		}
	}()

	r.MustRegister(&testPlugin{name: "ok"})
}

func TestPluginRegistry_RegisterAll(t *testing.T) {
	r := NewPluginRegistry()

	err := r.RegisterAll(
		&testPlugin{name: "plugin-1"},
		&testPlugin{name: "plugin-2"},
		&testPlugin{name: "plugin-3"},
	)
	if err != nil {
		t.Fatalf("RegisterAll() error = %v", err)
	}

	if len(r.List()) != 3 {
		t.Errorf("должно быть 3 плагина, получено %d", len(r.List()))
	}

	names := r.Names()
	if len(names) != 3 {
		t.Errorf("должно быть 3 имени, получено %d", len(names))
	}
}

func TestPluginRegistry_Names(t *testing.T) {
	r := NewPluginRegistry()
	r.MustRegister(&testPlugin{name: "alpha"})
	r.MustRegister(&testPlugin{name: "beta"})

	names := r.Names()
	if len(names) != 2 {
		t.Fatalf("должно быть 2 имени, получено %d", len(names))
	}
}
