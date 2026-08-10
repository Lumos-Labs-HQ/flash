package cachegen

import (
	"strings"
	"testing"
)

// These tests exercise the five non-Go cache accessor emitters
// (TypeScript, Python, Rust, Java, Kotlin). Each language is checked for the
// same three properties the Go emitter is checked for:
//
//   GUARD  — correct behavior that must not regress (these PASS today):
//              * empty input  -> empty output
//              * the named accessor container is emitted
//              * composite keys use a SINGLE colon (never "Name::") — this is
//                the bug the Go path HAS, so these pins prove the other five
//                are correct and lock them there.
//              * the TTL is forwarded in the unit that language expects.
//
//   BUG    — a real defect that produces invalid / uncompilable output (FAIL):
//              * tag identifiers collide or contain illegal characters
//              * a key param that is a language keyword is emitted bare
//
// Every BUG assertion below describes valid, compilable output; when it fails
// it is because the generator emitted something that genuinely will not build.

// twoParamCaches: one 0-param cache and one 2-param cache (TTL 5m) so composite
// key formatting and TTL conversion are both observable.
func twoParamCaches() []*CacheInfo {
	return []*CacheInfo{
		{QueryName: "list_all", CacheName: "AllCache", TTL: "1m"},
		{QueryName: "get_two", CacheName: "TwoCache", TTL: "5m",
			KeyParams: []string{"org_id", "user_id"}, KeyTypes: []string{"BIGINT", "BIGINT"}},
	}
}

// collidingTagCaches: two tags that some sanitizers fold into the same
// identifier ("user-posts" / "user_posts" both PascalCase to "UserPosts").
func collidingTagCaches() []*CacheInfo {
	return []*CacheInfo{
		{QueryName: "a", CacheName: "ACache", TTL: "1m", Tags: []string{"user-posts"}, KeyParams: []string{"id"}, KeyTypes: []string{"INT"}},
		{QueryName: "b", CacheName: "BCache", TTL: "1m", Tags: []string{"user_posts"}, KeyParams: []string{"id"}, KeyTypes: []string{"INT"}},
	}
}

// hyphenTagCache: a single hyphenated tag. Emitters that uppercase without
// sanitizing produce "USER-POSTS", which is not a legal identifier anywhere.
func hyphenTagCache() []*CacheInfo {
	return []*CacheInfo{
		{QueryName: "a", CacheName: "ACache", TTL: "1m", Tags: []string{"user-posts"}, KeyParams: []string{"id"}, KeyTypes: []string{"INT"}},
	}
}

// keywordParamCache builds a cache whose single key param is `name`.
func keywordParamCache(cacheName, param string) []*CacheInfo {
	return []*CacheInfo{
		{QueryName: "q", CacheName: cacheName, TTL: "1m", KeyParams: []string{param}, KeyTypes: []string{"TEXT"}},
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// TypeScript
// ─────────────────────────────────────────────────────────────────────────────

func TestGenerateTypeScriptCacheAccessors_Guards(t *testing.T) {
	if out := GenerateTypeScriptCacheAccessors(nil); out != "" {
		t.Errorf("empty input must yield empty output, got %q", out)
	}
	out := GenerateTypeScriptCacheAccessors(twoParamCaches())
	if !strings.Contains(out, "export const TwoCache = {") {
		t.Errorf("missing named accessor `export const TwoCache`:\n%s", out)
	}
	// Composite key: single colon between name and each param.
	if !strings.Contains(out, "TwoCache:${org_id}:${user_id}") {
		t.Errorf("2-param TS key must be single-colon `TwoCache:${org_id}:${user_id}`:\n%s", out)
	}
	if strings.Contains(out, "TwoCache::") {
		t.Errorf("TS composite key must not use a double colon:\n%s", out)
	}
	// TypeScript TTL is expressed in milliseconds (ParseTTL * 1000): 5m -> 300000.
	if !strings.Contains(out, "300000") {
		t.Errorf("TS TTL for 5m must be 300000 ms:\n%s", out)
	}
}

// BUG: "user-posts" and "user_posts" both PascalCase to "UserPosts", producing
// two `UserPosts = ...` members in the same enum — a duplicate-member compile
// error in TypeScript.
func TestGenerateTypeScriptCacheAccessors_TagCollision(t *testing.T) {
	out := GenerateTypeScriptCacheAccessors(collidingTagCaches())
	if n := strings.Count(out, "UserPosts ="); n > 1 {
		t.Errorf("duplicate CacheTag enum member `UserPosts` emitted %d times (won't compile):\n%s", n, out)
	}
}

// BUG: `function` is a reserved word and cannot be a parameter binding
// identifier in a TS module — the generated arrow fn won't compile.
func TestGenerateTypeScriptCacheAccessors_KeywordParam(t *testing.T) {
	out := GenerateTypeScriptCacheAccessors(keywordParamCache("FnCache", "function"))
	if strings.Contains(out, "function: string") {
		t.Errorf("reserved word `function` used as a bare param name (won't compile):\n%s", out)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Python
// ─────────────────────────────────────────────────────────────────────────────

func TestGeneratePythonCacheAccessors_Guards(t *testing.T) {
	if out := GeneratePythonCacheAccessors(nil); out != "" {
		t.Errorf("empty input must yield empty output, got %q", out)
	}
	out := GeneratePythonCacheAccessors(twoParamCaches())
	if !strings.Contains(out, "class TwoCache:") {
		t.Errorf("missing named accessor `class TwoCache`:\n%s", out)
	}
	if !strings.Contains(out, "TwoCache:{org_id}:{user_id}") {
		t.Errorf("2-param Python key must be single-colon `TwoCache:{org_id}:{user_id}`:\n%s", out)
	}
	if strings.Contains(out, "TwoCache::") {
		t.Errorf("Python composite key must not use a double colon:\n%s", out)
	}
	// Python TTL is seconds: 5m -> 300.
	if !strings.Contains(out, "value, 300,") {
		t.Errorf("Python TTL for 5m must be 300 seconds in set():\n%s", out)
	}
}

// BUG: ToUpper("user-posts") == "USER-POSTS"; a hyphen is illegal in a Python
// Enum member name, so `USER-POSTS = "user-posts"` is a SyntaxError.
func TestGeneratePythonCacheAccessors_TagInvalidIdentifier(t *testing.T) {
	out := GeneratePythonCacheAccessors(hyphenTagCache())
	if strings.Contains(out, "USER-POSTS") {
		t.Errorf("enum member `USER-POSTS` contains an illegal hyphen (SyntaxError):\n%s", out)
	}
}

// BUG: `class` is a Python keyword and cannot be a parameter name.
func TestGeneratePythonCacheAccessors_KeywordParam(t *testing.T) {
	out := GeneratePythonCacheAccessors(keywordParamCache("ClsCache", "class"))
	if strings.Contains(out, "get(class)") {
		t.Errorf("Python keyword `class` used as a bare param name (SyntaxError):\n%s", out)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Rust
// ─────────────────────────────────────────────────────────────────────────────

func TestGenerateRustCacheAccessors_Guards(t *testing.T) {
	if out := GenerateRustCacheAccessors(nil); out != "" {
		t.Errorf("empty input must yield empty output, got %q", out)
	}
	out := GenerateRustCacheAccessors(twoParamCaches())
	if !strings.Contains(out, "pub struct TwoCache;") {
		t.Errorf("missing named accessor `pub struct TwoCache`:\n%s", out)
	}
	// Rust output legitimately contains `::` (path separator), so probe the key
	// literal specifically rather than a blanket "::" search.
	if !strings.Contains(out, "TwoCache:{}:{}") {
		t.Errorf("2-param Rust key must be single-colon `TwoCache:{}:{}`:\n%s", out)
	}
	if strings.Contains(out, "TwoCache::{}") {
		t.Errorf("Rust composite key must not use a double colon:\n%s", out)
	}
	// Rust TTL is seconds via Duration::from_secs: 5m -> 300.
	if !strings.Contains(out, "Duration::from_secs(300)") {
		t.Errorf("Rust TTL for 5m must be Duration::from_secs(300):\n%s", out)
	}
}

// BUG: both tags PascalCase to `UserPosts`, producing two identical enum
// variants — a duplicate-variant compile error in Rust.
func TestGenerateRustCacheAccessors_TagCollision(t *testing.T) {
	out := GenerateRustCacheAccessors(collidingTagCaches())
	if n := strings.Count(out, "    UserPosts,"); n > 1 {
		t.Errorf("duplicate CacheTag variant `UserPosts` emitted %d times (won't compile):\n%s", n, out)
	}
}

// BUG: `type` is a Rust keyword; used bare in a fn signature it won't compile
// (it would need raw-identifier escaping, `r#type`).
func TestGenerateRustCacheAccessors_KeywordParam(t *testing.T) {
	out := GenerateRustCacheAccessors(keywordParamCache("TypeCache", "type"))
	if strings.Contains(out, "type: impl") {
		t.Errorf("Rust keyword `type` used as a bare param name (won't compile):\n%s", out)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Java
// ─────────────────────────────────────────────────────────────────────────────

func TestGenerateJavaCacheAccessors_Guards(t *testing.T) {
	if out := GenerateJavaCacheAccessors(nil, "app"); out != "" {
		t.Errorf("empty input must yield empty output, got %q", out)
	}
	out := GenerateJavaCacheAccessors(twoParamCaches(), "app")
	if !strings.Contains(out, "public class TwoCache {") {
		t.Errorf("missing named accessor `public class TwoCache`:\n%s", out)
	}
	// Java builds the key by string concat: "TwoCache:" + org_id + ":" + user_id.
	if !strings.Contains(out, `"TwoCache:" + org_id + ":" + user_id`) {
		t.Errorf("2-param Java key must be single-colon concat:\n%s", out)
	}
	if strings.Contains(out, "TwoCache::") {
		t.Errorf("Java composite key must not use a double colon:\n%s", out)
	}
	// Java TTL is a long-seconds literal: 5m -> 300L.
	if !strings.Contains(out, "300L") {
		t.Errorf("Java TTL for 5m must be 300L:\n%s", out)
	}
}

// BUG: ToUpper("user-posts") == "USER-POSTS"; a hyphen is illegal in a Java
// enum constant name, so the enum won't compile.
func TestGenerateJavaCacheAccessors_TagInvalidIdentifier(t *testing.T) {
	out := GenerateJavaCacheAccessors(hyphenTagCache(), "app")
	if strings.Contains(out, "USER-POSTS") {
		t.Errorf("Java enum constant `USER-POSTS` contains an illegal hyphen (won't compile):\n%s", out)
	}
}

// BUG: `class` is a Java keyword and cannot be a parameter name.
func TestGenerateJavaCacheAccessors_KeywordParam(t *testing.T) {
	out := GenerateJavaCacheAccessors(keywordParamCache("ClsCache", "class"), "app")
	if strings.Contains(out, "Object class") {
		t.Errorf("Java keyword `class` used as a bare param name (won't compile):\n%s", out)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Kotlin
// ─────────────────────────────────────────────────────────────────────────────

func TestGenerateKotlinCacheAccessors_Guards(t *testing.T) {
	if out := GenerateKotlinCacheAccessors(nil, "app"); out != "" {
		t.Errorf("empty input must yield empty output, got %q", out)
	}
	out := GenerateKotlinCacheAccessors(twoParamCaches(), "app")
	if !strings.Contains(out, "object TwoCache {") {
		t.Errorf("missing named accessor `object TwoCache`:\n%s", out)
	}
	if !strings.Contains(out, "TwoCache:$org_id:$user_id") {
		t.Errorf("2-param Kotlin key must be single-colon `TwoCache:$org_id:$user_id`:\n%s", out)
	}
	if strings.Contains(out, "TwoCache::") {
		t.Errorf("Kotlin composite key must not use a double colon:\n%s", out)
	}
	// Kotlin TTL is seconds: 5m -> 300.
	if !strings.Contains(out, "value, 300,") {
		t.Errorf("Kotlin TTL for 5m must be 300 seconds in set():\n%s", out)
	}
}

// BUG: ToUpper("user-posts") == "USER-POSTS"; a hyphen is illegal in a Kotlin
// enum entry name, so the enum won't compile.
func TestGenerateKotlinCacheAccessors_TagInvalidIdentifier(t *testing.T) {
	out := GenerateKotlinCacheAccessors(hyphenTagCache(), "app")
	if strings.Contains(out, "USER-POSTS") {
		t.Errorf("Kotlin enum entry `USER-POSTS` contains an illegal hyphen (won't compile):\n%s", out)
	}
}

// BUG: `class` is a hard keyword in Kotlin and cannot be a parameter name
// without backtick escaping.
func TestGenerateKotlinCacheAccessors_KeywordParam(t *testing.T) {
	out := GenerateKotlinCacheAccessors(keywordParamCache("ClsCache", "class"), "app")
	if strings.Contains(out, "class: Any") {
		t.Errorf("Kotlin keyword `class` used as a bare param name (won't compile):\n%s", out)
	}
}
