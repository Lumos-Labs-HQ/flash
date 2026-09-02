package cachegen

import (
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/parser"
)

// ── runtime cache modules (Generate<X>Cache) ────────────────────────────────

func TestGenerateGoCache_ModuleBasics(t *testing.T) {
	cfg := &config.CacheConfig{Enabled: true, DefaultTTL: "5m", Prefix: "flash", RedisURLEnv: "REDIS_URL"}
	out := GenerateGoCache(cfg, "app")
	for _, want := range []string{
		"package app",
		`os.Getenv("REDIS_URL")`,
		"type FlashCache struct",
		"func InitCache()",
		`prefix: "flash"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Go cache module missing %q:\n%s", want, out)
		}
	}
}

func TestGeneratePythonCache_ModuleBasics(t *testing.T) {
	cfg := &config.CacheConfig{Enabled: true, DefaultTTL: "5m", Prefix: "flash", RedisURLEnv: "REDIS_URL"}
	out := GeneratePythonCache(cfg)
	for _, want := range []string{
		`os.environ.get("REDIS_URL"`,
		`_PREFIX = "flash"`,
		"class FlashCache:",
		"def purge_by_tag",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Python cache module missing %q:\n%s", want, out)
		}
	}
}

func TestGenerateTypeScriptCache_ModuleBasics(t *testing.T) {
	cfg := &config.CacheConfig{Enabled: true, DefaultTTL: "5m", Prefix: "flash", RedisURLEnv: "REDIS_URL"}
	out := GenerateTypeScriptCache(cfg)
	for _, want := range []string{
		`process.env.REDIS_URL`,
		`const PREFIX = "flash";`,
		"export class FlashCache",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("TS cache module missing %q:\n%s", want, out)
		}
	}
}

func TestGenerateRustCache_ModuleBasics(t *testing.T) {
	cfg := &config.CacheConfig{Enabled: true, DefaultTTL: "5m", Prefix: "flash", RedisURLEnv: "REDIS_URL"}
	out := GenerateRustCache(cfg)
	for _, want := range []string{
		`std::env::var("REDIS_URL")`,
		`prefix: "flash".to_string()`,
		"pub struct FlashCache",
		"pub fn purge_by_tag",
		"lazy_static::lazy_static!",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Rust cache module missing %q:\n%s", want, out)
		}
	}
}

func TestGenerateJavaCache_ModuleBasics(t *testing.T) {
	cfg := &config.CacheConfig{Enabled: true, DefaultTTL: "5m", Prefix: "flash", RedisURLEnv: "REDIS_URL"}
	out := GenerateJavaCache(cfg, "com.example")
	for _, want := range []string{
		"package com.example;",
		`System.getenv("REDIS_URL")`,
		`PREFIX = "flash"`,
		"public class FlashCache",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Java cache module missing %q:\n%s", want, out)
		}
	}
}

func TestGenerateKotlinCache_ModuleBasics(t *testing.T) {
	cfg := &config.CacheConfig{Enabled: true, DefaultTTL: "5m", Prefix: "flash", RedisURLEnv: "REDIS_URL"}
	out := GenerateKotlinCache(cfg, "com.example")
	for _, want := range []string{
		"package com.example",
		`System.getenv("REDIS_URL")`,
		`PREFIX = "flash"`,
		"object FlashCache",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Kotlin cache module missing %q:\n%s", want, out)
		}
	}
}

// Prefix must be honored in the runtime module so the default doesn't leak
// into every project.
func TestCacheModule_PrefixPropagation(t *testing.T) {
	cfg := &config.CacheConfig{Enabled: true, DefaultTTL: "1m", Prefix: "myapp"}
	if out := GeneratePythonCache(cfg); !strings.Contains(out, "myapp") {
		t.Errorf("Python cache module ignores configured prefix:\n%s", out)
	}
	if out := GenerateRustCache(cfg); !strings.Contains(out, "myapp") {
		t.Errorf("Rust cache module ignores configured prefix:\n%s", out)
	}
}

// RedisURLEnv customization must flow into the runtime modules.
func TestCacheModule_CustomRedisEnvVar(t *testing.T) {
	cfg := &config.CacheConfig{Enabled: true, DefaultTTL: "1m", RedisURLEnv: "PROD_CACHE_URL"}
	if out := GenerateGoCache(cfg, "app"); !strings.Contains(out, "PROD_CACHE_URL") {
		t.Errorf("Go cache module ignores custom Redis env var:\n%s", out)
	}
	if out := GeneratePythonCache(cfg); !strings.Contains(out, "PROD_CACHE_URL") {
		t.Errorf("Python cache module ignores custom Redis env var:\n%s", out)
	}
}

// ── accessor edge cases across languages ────────────────────────────────────

// threeCacheSet exercises 0/1/3-param caches with mixed TTL units.
func threeCacheSet() []*CacheInfo {
	return []*CacheInfo{
		{QueryName: "list", CacheName: "ListCache", TTL: "1h", Tags: []string{"all"}},
		{QueryName: "by_id", CacheName: "IdCache", TTL: "45s", KeyParams: []string{"id"}, KeyTypes: []string{"INT"}},
		{QueryName: "by_three", CacheName: "ThreeCache", TTL: "1d", KeyParams: []string{"a", "b", "c"}, KeyTypes: []string{"INT", "TEXT", "FLOAT"}},
	}
}

func TestAccessors_AllLanguages_HourAndDayTTLs(t *testing.T) {
	cases := []struct {
		name string
		out  string
	}{
		{"go", GenerateGoCacheAccessors(threeCacheSet(), "app", "flash")},
		{"ts", GenerateTypeScriptCacheAccessors(threeCacheSet())},
		{"python", GeneratePythonCacheAccessors(threeCacheSet())},
		{"rust", GenerateRustCacheAccessors(threeCacheSet())},
		{"java", GenerateJavaCacheAccessors(threeCacheSet(), "app")},
		{"kotlin", GenerateKotlinCacheAccessors(threeCacheSet(), "app")},
	}
	for _, c := range cases {
		if !strings.Contains(c.out, "ListCache") {
			t.Errorf("%s: 0-param accessor missing", c.name)
		}
		// 1h = 3600 (s/ms conversions verified per language below)
		switch c.name {
		case "go":
			if !strings.Contains(c.out, "3600*time.Second") {
				t.Errorf("go: 1h TTL must be 3600*time.Second:\n%s", c.out)
			}
			if !strings.Contains(c.out, "86400*time.Second") {
				t.Errorf("go: 1d TTL must be 86400*time.Second:\n%s", c.out)
			}
		case "ts":
			if !strings.Contains(c.out, "3600000") {
				t.Errorf("ts: 1h TTL must be 3600000 ms:\n%s", c.out)
			}
		case "python", "kotlin":
			if !strings.Contains(c.out, "3600") {
				t.Errorf("%s: 1h TTL must include 3600 seconds:\n%s", c.name, c.out)
			}
		case "rust":
			if !strings.Contains(c.out, "from_secs(3600)") {
				t.Errorf("rust: 1h TTL must be Duration::from_secs(3600):\n%s", c.out)
			}
		case "java":
			if !strings.Contains(c.out, "3600L") {
				t.Errorf("java: 1h TTL must be 3600L:\n%s", c.out)
			}
		}
	}
}

// An invalid TTL must not crash generation and must not produce a bogus
// duration (0 is the current fallback — cache entries expire immediately,
// which is safe albeit degraded).
func TestAccessors_InvalidTTLSanitized(t *testing.T) {
	caches := []*CacheInfo{
		{QueryName: "q", CacheName: "BadCache", TTL: "not-a-ttl", KeyParams: []string{"id"}, KeyTypes: []string{"INT"}},
	}
	// None of these may panic.
	_ = GenerateGoCacheAccessors(caches, "app", "flash")
	_ = GenerateTypeScriptCacheAccessors(caches)
	_ = GeneratePythonCacheAccessors(caches)
	_ = GenerateRustCacheAccessors(caches)
	_ = GenerateJavaCacheAccessors(caches, "app")
	_ = GenerateKotlinCacheAccessors(caches, "app")
}

// Multiple caches with the same tag: the tag constant must exist once and the
// runtime purge-by-tag path is exercised via tags lists on each accessor.
func TestAccessors_SharedTagAcrossCaches(t *testing.T) {
	caches := []*CacheInfo{
		{QueryName: "a", CacheName: "ACache", TTL: "1m", Tags: []string{"users"}, KeyParams: []string{"id"}, KeyTypes: []string{"INT"}},
		{QueryName: "b", CacheName: "BCache", TTL: "1m", Tags: []string{"users", "admin"}, KeyParams: []string{"id"}, KeyTypes: []string{"INT"}},
	}
	for name, out := range map[string]string{
		"go":     GenerateGoCacheAccessors(caches, "app", "flash"),
		"ts":     GenerateTypeScriptCacheAccessors(caches),
		"python": GeneratePythonCacheAccessors(caches),
		"rust":   GenerateRustCacheAccessors(caches),
	} {
		if n := strings.Count(out, "users"); n < 2 {
			t.Errorf("%s: shared tag 'users' should appear on both accessors (found %d):\n%s", name, n, out)
		}
	}
}

// ── resolver edge cases ─────────────────────────────────────────────────────

func TestResolveCacheQueries_MultipleCaches(t *testing.T) {
	cfg := &config.CacheConfig{Enabled: true, DefaultTTL: "10s"}
	queries := []*parser.Query{
		{Name: "get_user", Cmd: ":one", CacheDef: &parser.CacheDef{Name: "UserCache", TTL: "30s"}, Params: []*parser.Param{{Name: "id"}}},
		{Name: "get_post", Cmd: ":one", CacheDef: &parser.CacheDef{Name: "PostCache", Dep: []string{"UpdatePost"}}, Params: []*parser.Param{{Name: "id"}}},
		{Name: "no_annot", Cmd: ":one"},
	}
	caches := ResolveCacheQueries(queries, cfg)
	if len(caches) != 2 {
		t.Fatalf("got %d caches, want 2", len(caches))
	}
	// Default TTL only where not set explicitly.
	if caches[0].TTL != "30s" {
		t.Errorf("UserCache TTL = %q, want explicit 30s", caches[0].TTL)
	}
	if caches[1].TTL != "10s" {
		t.Errorf("PostCache TTL = %q, want default 10s", caches[1].TTL)
	}
}

func TestResolveCacheQueries_NilCacheDefSkipped(t *testing.T) {
	cfg := &config.CacheConfig{Enabled: true}
	queries := []*parser.Query{
		{Name: "one", Cmd: ":one"},
		{Name: "two", Cmd: ":one", CacheDef: nil},
	}
	if caches := ResolveCacheQueries(queries, cfg); len(caches) != 0 {
		t.Errorf("nil CacheDef queries must be skipped, got %d", len(caches))
	}
}

func TestResolveCacheQueries_ParamlessCache(t *testing.T) {
	cfg := &config.CacheConfig{Enabled: true}
	queries := []*parser.Query{
		{Name: "list_all", Cmd: ":many", CacheDef: &parser.CacheDef{}},
	}
	caches := ResolveCacheQueries(queries, cfg)
	if len(caches) != 1 {
		t.Fatalf("got %d caches", len(caches))
	}
	if len(caches[0].KeyParams) != 0 || len(caches[0].KeyTypes) != 0 {
		t.Errorf("paramless query must yield empty key params: %+v", caches[0])
	}
}

func TestResolveDependencyPurges_MultipleDepsAndCaches(t *testing.T) {
	caches := []*CacheInfo{
		{CacheName: "UserCache", Dep: []string{"UpdateUser", "DeleteUser"}, KeyParams: []string{"id"}},
		{CacheName: "ListCache", Dep: []string{"UpdateUser"}, KeyParams: nil},
	}
	// UpdateUser invalidates both.
	purges := ResolveDependencyPurges("UpdateUser", caches, mkParams("id", "name"))
	if len(purges) != 2 {
		t.Fatalf("got %d purges, want 2", len(purges))
	}
	byCache := map[string]DependencyPurge{}
	for _, p := range purges {
		byCache[p.CacheName] = p
	}
	if p := byCache["UserCache"]; p.PurgePrefix || p.KeyParam != "id" {
		t.Errorf("UserCache: want exact delete on id, got %+v", p)
	}
	if p := byCache["ListCache"]; !p.PurgePrefix {
		t.Errorf("ListCache: no key params -> must purge by prefix, got %+v", p)
	}

	// DeleteUser invalidates only UserCache.
	purges = ResolveDependencyPurges("DeleteUser", caches, mkParams("id"))
	if len(purges) != 1 || purges[0].CacheName != "UserCache" {
		t.Fatalf("DeleteUser purges = %+v, want only UserCache", purges)
	}
}

func TestResolveDependencyPurges_CaseInsensitiveDep(t *testing.T) {
	caches := []*CacheInfo{
		{CacheName: "UserCache", Dep: []string{"updateuser"}, KeyParams: []string{"id"}},
	}
	purges := ResolveDependencyPurges("UpdateUser", caches, mkParams("id"))
	if len(purges) != 1 {
		t.Fatalf("dep matching must be case-insensitive, got %+v", purges)
	}
}

func TestResolveDependencyPurges_ExactMatchRequiresSameName(t *testing.T) {
	caches := []*CacheInfo{
		{CacheName: "UserCache", Dep: []string{"UpdateUser"}, KeyParams: []string{"user_id"}},
	}
	// Mutation has "id" but the cache key is "user_id" — no exact param match.
	purges := ResolveDependencyPurges("UpdateUser", caches, mkParams("id"))
	if len(purges) != 1 || !purges[0].PurgePrefix {
		t.Fatalf("param name mismatch must fall back to prefix purge, got %+v", purges)
	}
	// Same-name param yields exact delete.
	purges = ResolveDependencyPurges("UpdateUser", caches, mkParams("user_id"))
	if len(purges) != 1 || purges[0].PurgePrefix {
		t.Fatalf("matching param must exact-delete, got %+v", purges)
	}
}

// ── ParseTTL exhaustive ─────────────────────────────────────────────────────

func TestParseTTL_ExhaustiveTable(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"0", 0},
		{"0s", 0},
		{"1", 1},
		{"59s", 59},
		{"60s", 60},
		{"1m", 60},
		{"90s", 90},
		{"1h", 3600},
		{"24h", 86400},
		{"1d", 86400},
		{"7d", 604800},
		{"1h0m0s", 3600},
		{"2h45m30s", 9930},
		{"0.5d", 43200},
		{"1.25h", 4500},
		{"  10m  ", 600},
		{"10 m", 600},
		{"100", 100},
		// Invalid inputs all yield 0
		{"", 0},
		{" ", 0},
		{"-1", 0},
		{"-30s", 0},
		{"abc", 0},
		{"5x", 0},
		{"s", 0},
		{"m5", 0},
		{"1h-", 0},
		{"1.2.3h", 0},
		{"1h m", 0},
		{"5 seconds", 5}, // internal spaces are stripped; "5seconds" is a valid long unit
		{"infinity", 0},
	}
	for _, tt := range tests {
		if got := ParseTTL(tt.in); got != tt.want {
			t.Errorf("ParseTTL(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

// ── BuildCacheKey edge cases ────────────────────────────────────────────────

func TestBuildCacheKey_EmptyPrefix(t *testing.T) {
	if got := BuildCacheKey("", "UserCache", []string{"1"}); got != ":UserCache:1" {
		t.Errorf("BuildCacheKey empty prefix = %q", got)
	}
}

func TestBuildCacheKey_ParamWithColon(t *testing.T) {
	// A param value containing ':' would corrupt the key structure — the key
	// builder joins blindly. This documents the limitation (params are trusted
	// to be scalar ids).
	got := BuildCacheKey("flash", "Cache", []string{"a:b"})
	if got != "flash:Cache:a:b" {
		t.Errorf("BuildCacheKey = %q", got)
	}
}
