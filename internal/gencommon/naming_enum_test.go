package gencommon

import (
	"reflect"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// ExtractEnumValues
// ─────────────────────────────────────────────────────────────────────────────

func TestExtractEnumValues_Guards(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"enum('active','inactive')", []string{"active", "inactive"}},
		{"enum('a', 'b', 'c')", []string{"a", "b", "c"}}, // interior spaces trimmed
		{"ENUM('a','b')", []string{"a", "b"}},            // prefix match is case-insensitive
		{"enum('solo')", []string{"solo"}},
		{"varchar(255)", nil}, // not an enum
		{"text", nil},
	}
	for _, tt := range tests {
		got := ExtractEnumValues(tt.in)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("ExtractEnumValues(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestExtractEnumValues_PreservesValueCase(t *testing.T) {
	got := ExtractEnumValues("enum('Active','Inactive','PENDING')")
	want := []string{"Active", "Inactive", "PENDING"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExtractEnumValues lowercased enum values: got %v, want %v (case must be preserved)", got, want)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ParseCQLInner / ExtractCQLInner / ExtractCollectionInner
// ─────────────────────────────────────────────────────────────────────────────

func TestParseCQLInner(t *testing.T) {
	tests := []struct {
		in    string
		wantK string
		wantV string
	}{
		{"set<uuid>", "uuid", ""},
		{"list<text>", "text", ""},
		{"frozen<address>", "address", ""},
		{"map<text,int>", "text", "int"},
		{"map<text, int>", "text", "int"},                          // space after comma trimmed
		{"map<uuid,frozen<set<int>>>", "uuid", "frozen<set<int>>"}, // nested value type
		{"int", "text", ""},                                        // non-collection fallback
	}
	for _, tt := range tests {
		k, v := ParseCQLInner(tt.in)
		if k != tt.wantK || v != tt.wantV {
			t.Errorf("ParseCQLInner(%q) = (%q,%q), want (%q,%q)", tt.in, k, v, tt.wantK, tt.wantV)
		}
	}
}

func TestExtractCollectionInner(t *testing.T) {
	tests := []struct{ in, want string }{
		{"set<int>", "int"},
		{"list<text>", "text"},
		{"frozen<my_udt>", "my_udt"},
		{"frozen<map<text,int>>", "map<text,int>"},
		{"map<text,int>", "text"}, // map isn't a single-element collection -> fallback
		{"bigint", "text"},        // fallback
	}
	for _, tt := range tests {
		if got := ExtractCollectionInner(tt.in); got != tt.want {
			t.Errorf("ExtractCollectionInner(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestExtractCQLInner(t *testing.T) {
	if inner, ok := ExtractCQLInner("frozen<addr>", "frozen"); !ok || inner != "addr" {
		t.Errorf("ExtractCQLInner frozen<addr> = (%q,%v), want (addr,true)", inner, ok)
	}
	if _, ok := ExtractCQLInner("set<int>", "frozen"); ok {
		t.Error("ExtractCQLInner must not match a different wrapper")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ExtractCQLMap / ExtractMapTypes
// ─────────────────────────────────────────────────────────────────────────────

func TestExtractCQLMap(t *testing.T) {
	k, v, ok := ExtractCQLMap("map<uuid,text>")
	if !ok || k != "uuid" || v != "text" {
		t.Errorf("ExtractCQLMap(map<uuid,text>) = (%q,%q,%v), want (uuid,text,true)", k, v, ok)
	}
	// Nested value containing its own comma must not split early.
	k, v, ok = ExtractCQLMap("map<uuid,map<text,int>>")
	if !ok || k != "uuid" || v != "map<text,int>" {
		t.Errorf("nested map value mis-parsed: (%q,%q,%v)", k, v, ok)
	}
	if _, _, ok := ExtractCQLMap("set<int>"); ok {
		t.Error("ExtractCQLMap must return false for a non-map type")
	}
}

func TestExtractMapTypes_FallbackForNonMap(t *testing.T) {
	if k, v := ExtractMapTypes("list<int>"); k != "string" || v != "string" {
		t.Errorf("ExtractMapTypes(non-map) = (%q,%q), want (string,string) fallback", k, v)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// QueryPascal / ToCamelCase
// ─────────────────────────────────────────────────────────────────────────────

func TestQueryPascal(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"get_user", "GetUser"},
		{"GetUser", "GetUser"},         // already exported -> preserved verbatim
		{"GetUserByID", "GetUserByID"}, // acronym preserved (not re-cased to GetUserById)
	}
	for _, tt := range tests {
		if got := QueryPascal(tt.in); got != tt.want {
			t.Errorf("QueryPascal(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"get_user", "getUser"},
		{"GetUser", "getUser"},
		{"UserID", "userID"}, // only the first rune is lowercased
	}
	for _, tt := range tests {
		if got := ToCamelCase(tt.in); got != tt.want {
			t.Errorf("ToCamelCase(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
