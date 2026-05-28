package common

import "testing"

func TestValidateIdentifier(t *testing.T) {
	valid := []string{"users", "user_posts", "_private", "Table123", "a"}
	for _, name := range valid {
		if err := ValidateIdentifier(name); err != nil {
			t.Errorf("ValidateIdentifier(%q) returned error: %v", name, err)
		}
	}

	invalid := []string{"", "123table", "table name", "table;DROP", "table--comment",
		"table/*comment*/", `table"inject`, "public.users", "a.b.c"}
	for _, name := range invalid {
		if err := ValidateIdentifier(name); err == nil {
			t.Errorf("ValidateIdentifier(%q) should have returned error", name)
		}
	}
}

func TestValidateQualifiedIdentifier(t *testing.T) {
	valid := []string{"users", "public.users", "my_schema.my_table", "a.b.c"}
	for _, name := range valid {
		if err := ValidateQualifiedIdentifier(name); err != nil {
			t.Errorf("ValidateQualifiedIdentifier(%q) returned error: %v", name, err)
		}
	}

	invalid := []string{"", "public.", ".users", "public..users", "a;b.c", "a.b;DROP"}
	for _, name := range invalid {
		if err := ValidateQualifiedIdentifier(name); err == nil {
			t.Errorf("ValidateQualifiedIdentifier(%q) should have returned error", name)
		}
	}
}
