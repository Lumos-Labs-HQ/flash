package seeder

import (
	"strings"
	"testing"
)

// ── DataGenerator ─────────────────────────────────────────────────────────────

func newGen(t *testing.T) *DataGenerator {
	t.Helper()
	g, err := NewDataGenerator()
	if err != nil {
		t.Fatalf("NewDataGenerator: %v", err)
	}
	return g
}

func TestDataGenerator_Generate_Int(t *testing.T) {
	g := newGen(t)
	v := g.Generate("INTEGER", false)
	if _, ok := v.(int); !ok {
		t.Errorf("Generate(INTEGER) = %T, want int", v)
	}
}

func TestDataGenerator_Generate_Text(t *testing.T) {
	g := newGen(t)
	v := g.Generate("TEXT", false)
	if _, ok := v.(string); !ok {
		t.Errorf("Generate(TEXT) = %T, want string", v)
	}
}

func TestDataGenerator_Generate_Bool(t *testing.T) {
	g := newGen(t)
	v := g.Generate("BOOLEAN", false)
	if _, ok := v.(bool); !ok {
		t.Errorf("Generate(BOOLEAN) = %T, want bool", v)
	}
}

func TestDataGenerator_Generate_Float(t *testing.T) {
	g := newGen(t)
	v := g.Generate("DECIMAL", false)
	if _, ok := v.(float64); !ok {
		t.Errorf("Generate(DECIMAL) = %T, want float64", v)
	}
}

func TestDataGenerator_Generate_UUID(t *testing.T) {
	g := newGen(t)
	v := g.Generate("UUID", false)
	s, ok := v.(string)
	if !ok {
		t.Fatalf("Generate(UUID) = %T, want string", v)
	}
	if len(strings.Split(s, "-")) != 5 {
		t.Errorf("UUID format invalid: %q", s)
	}
}

func TestDataGenerator_Generate_Nullable_CanBeNil(t *testing.T) {
	g := newGen(t)
	// With nullable=true, nil is possible. Run enough times to see it.
	gotNil := false
	for i := 0; i < 100; i++ {
		if g.Generate("TEXT", true) == nil {
			gotNil = true
			break
		}
	}
	if !gotNil {
		t.Error("nullable column never returned nil in 100 attempts")
	}
}

func TestDataGenerator_GenerateForColumn_Email(t *testing.T) {
	g := newGen(t)
	v := g.GenerateForColumn("email", "TEXT", false)
	s, ok := v.(string)
	if !ok {
		t.Fatalf("email column = %T, want string", v)
	}
	if !strings.Contains(s, "@") {
		t.Errorf("email = %q, missing @", s)
	}
}

func TestDataGenerator_GenerateForColumn_Phone(t *testing.T) {
	g := newGen(t)
	v := g.GenerateForColumn("phone", "TEXT", false)
	s, ok := v.(string)
	if !ok {
		t.Fatalf("phone column = %T, want string", v)
	}
	if len(s) == 0 {
		t.Error("phone is empty")
	}
}

func TestDataGenerator_GenerateForColumn_Document_IsNil(t *testing.T) {
	g := newGen(t)
	// document columns must always return nil (security rule)
	for i := 0; i < 20; i++ {
		if v := g.GenerateForColumn("document", "TEXT", false); v != nil {
			t.Errorf("document column = %v, want nil", v)
			break
		}
	}
}

func TestDataGenerator_GenerateForColumn_Status(t *testing.T) {
	g := newGen(t)
	v := g.GenerateForColumn("status", "TEXT", false)
	if _, ok := v.(string); !ok {
		t.Errorf("status column = %T, want string", v)
	}
}

// ── DependencyGraph ───────────────────────────────────────────────────────────

func TestDependencyGraph_BuildInsertionOrder_Simple(t *testing.T) {
	g := NewDependencyGraph()
	g.AddTable(&TableInfo{Name: "users", Dependencies: []string{}})
	g.AddTable(&TableInfo{Name: "posts", Dependencies: []string{"users"}})
	g.AddTable(&TableInfo{Name: "comments", Dependencies: []string{"posts", "users"}})

	order, err := g.BuildInsertionOrder()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 3 {
		t.Fatalf("order = %d, want 3", len(order))
	}

	pos := func(name string) int {
		for i, n := range order {
			if n == name {
				return i
			}
		}
		return -1
	}

	if pos("users") >= pos("posts") {
		t.Errorf("users must come before posts: %v", order)
	}
	if pos("posts") >= pos("comments") {
		t.Errorf("posts must come before comments: %v", order)
	}
}

func TestDependencyGraph_BuildInsertionOrder_Circular(t *testing.T) {
	g := NewDependencyGraph()
	g.AddTable(&TableInfo{Name: "a", Dependencies: []string{"b"}})
	g.AddTable(&TableInfo{Name: "b", Dependencies: []string{"a"}})

	_, err := g.BuildInsertionOrder()
	if err == nil {
		t.Error("expected circular dependency error, got nil")
	}
}

func TestDependencyGraph_BuildInsertionOrder_SelfReference(t *testing.T) {
	// Self-referencing tables (e.g. categories with parent_id) must not cause infinite loop.
	g := NewDependencyGraph()
	g.AddTable(&TableInfo{Name: "categories", Dependencies: []string{"categories"}})

	order, err := g.BuildInsertionOrder()
	if err != nil {
		t.Fatalf("self-reference should not error: %v", err)
	}
	if len(order) != 1 {
		t.Errorf("order = %v, want [categories]", order)
	}
}

func TestDependencyGraph_GetOrder(t *testing.T) {
	g := NewDependencyGraph()
	g.AddTable(&TableInfo{Name: "users", Dependencies: []string{}})
	g.BuildInsertionOrder()
	if len(g.GetOrder()) != 1 {
		t.Errorf("GetOrder = %v, want [users]", g.GetOrder())
	}
}
