package database

import (
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/database/mongodb"
	"github.com/Lumos-Labs-HQ/flash/internal/database/mysql"
	"github.com/Lumos-Labs-HQ/flash/internal/database/postgres"
	"github.com/Lumos-Labs-HQ/flash/internal/database/sqlite"
)

func TestNewAdapter_ReturnsCorrectType(t *testing.T) {
	cases := []struct {
		provider string
		wantType interface{}
	}{
		{"postgresql", &postgres.Adapter{}},
		{"postgres", &postgres.Adapter{}},
		{"mysql", &mysql.Adapter{}},
		{"sqlite", &sqlite.Adapter{}},
		{"sqlite3", &sqlite.Adapter{}},
		{"mongodb", &mongodb.Adapter{}},
		{"mongo", &mongodb.Adapter{}},
		{"unknown", &postgres.Adapter{}}, // default
	}

	for _, c := range cases {
		adapter := NewAdapter(c.provider)
		if adapter == nil {
			t.Errorf("NewAdapter(%q) returned nil", c.provider)
			continue
		}
		switch c.wantType.(type) {
		case *postgres.Adapter:
			if _, ok := adapter.(*postgres.Adapter); !ok {
				t.Errorf("NewAdapter(%q) = %T, want *postgres.Adapter", c.provider, adapter)
			}
		case *mysql.Adapter:
			if _, ok := adapter.(*mysql.Adapter); !ok {
				t.Errorf("NewAdapter(%q) = %T, want *mysql.Adapter", c.provider, adapter)
			}
		case *sqlite.Adapter:
			if _, ok := adapter.(*sqlite.Adapter); !ok {
				t.Errorf("NewAdapter(%q) = %T, want *sqlite.Adapter", c.provider, adapter)
			}
		case *mongodb.Adapter:
			if _, ok := adapter.(*mongodb.Adapter); !ok {
				t.Errorf("NewAdapter(%q) = %T, want *mongodb.Adapter", c.provider, adapter)
			}
		}
	}
}

func TestNewAdapter_ImplementsInterface(t *testing.T) {
	// Compile-time check: all adapters must satisfy DatabaseAdapter.
	// If any adapter is missing a method, this won't compile.
	providers := []string{"postgresql", "mysql", "sqlite", "mongodb"}
	for _, p := range providers {
		adapter := NewAdapter(p)
		var _ DatabaseAdapter = adapter
		if adapter == nil {
			t.Errorf("NewAdapter(%q) returned nil", p)
		}
	}
}
