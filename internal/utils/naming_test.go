package utils

import (
	"testing"
)

func TestToPascalCase(t *testing.T) {
	cases := []struct{ in, want string }{
		{"user_id", "UserId"},
		{"created_at", "CreatedAt"},
		{"id", "Id"},
		{"first_name", "FirstName"},
		{"USER_NAME", "UserName"}, // toTitleCase lowercases rest
		{"", ""},
	}
	for _, c := range cases {
		got := ToPascalCase(c.in)
		if got != c.want {
			t.Errorf("ToPascalCase(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestToSnakeCase(t *testing.T) {
	cases := []struct{ in, want string }{
		{"UserId", "user_id"},
		{"CreatedAt", "created_at"},
		{"firstName", "first_name"},
		{"", ""},
	}
	for _, c := range cases {
		got := ToSnakeCase(c.in)
		if got != c.want {
			t.Errorf("ToSnakeCase(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCapitalize(t *testing.T) {
	cases := []struct{ in, want string }{
		{"users", "Users"},
		{"user_profile", "UserProfile"},
		{"", ""},
	}
	for _, c := range cases {
		got := Capitalize(c.in)
		if got != c.want {
			t.Errorf("Capitalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestUncapitalize(t *testing.T) {
	got := Uncapitalize("GetUser")
	if got != "getUser" {
		t.Errorf("Uncapitalize(%q) = %q, want getUser", "GetUser", got)
	}
}
