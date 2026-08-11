package cachegen

import (
	"strings"
	"testing"
)

// The generated cache key scheme must be consistent across arities. The 0- and
// 1-param paths use a single colon (Name / Name:%v); the >=2-param path must
// too. Current code emits fmt.Sprintf("Name::%v:%v", ...) — a double colon —
// producing a different keyspace that the purge/other-language paths never hit.
func TestGenerateGoCacheAccessors_KeyFormats(t *testing.T) {
	caches := []*CacheInfo{
		{QueryName: "list_all", CacheName: "AllCache", TTL: "1m"},
		{QueryName: "get_one", CacheName: "OneCache", TTL: "1m", KeyParams: []string{"id"}, KeyTypes: []string{"BIGINT"}},
		{QueryName: "get_two", CacheName: "TwoCache", TTL: "1m", KeyParams: []string{"org_id", "user_id"}, KeyTypes: []string{"BIGINT", "BIGINT"}},
	}
	out := GenerateGoCacheAccessors(caches, "app", "flash")

	if !strings.Contains(out, `key := "AllCache"`) {
		t.Errorf("0-param cache key wrong; want `key := \"AllCache\"`:\n%s", out)
	}
	if !strings.Contains(out, `fmt.Sprintf("OneCache:%v", id)`) {
		t.Errorf("1-param cache key wrong; want single colon:\n%s", out)
	}
	if strings.Contains(out, "::") {
		t.Errorf("composite cache key contains a double colon (inconsistent keyspace):\n%s", out)
	}
	if !strings.Contains(out, `fmt.Sprintf("TwoCache:%v:%v", org_id, user_id)`) {
		t.Errorf("2-param cache key wrong; want single-colon composite:\n%s", out)
	}
}

// A key param whose name collides with a Go keyword must be escaped, otherwise
// the generated accessor file will not compile. Current code passes `type`
// straight through -> `func (a *typeCacheAccessor) Get(type string)`.
func TestGenerateGoCacheAccessors_KeywordParamIsSafe(t *testing.T) {
	caches := []*CacheInfo{
		{QueryName: "get_by_type", CacheName: "TypeCache", TTL: "1m", KeyParams: []string{"type"}, KeyTypes: []string{"TEXT"}},
	}
	out := GenerateGoCacheAccessors(caches, "app", "flash")
	for _, bad := range []string{"Get(type ", "Del(type ", "(type string)"} {
		if strings.Contains(out, bad) {
			t.Errorf("generated code uses Go keyword `type` as a bare identifier (won't compile): found %q in:\n%s", bad, out)
		}
	}
}

func TestGenerateGoCacheAccessors_TTLConvertedToSeconds(t *testing.T) {
	caches := []*CacheInfo{
		{QueryName: "get_user", CacheName: "UserCache", TTL: "5m", KeyParams: []string{"id"}, KeyTypes: []string{"BIGINT"}},
	}
	out := GenerateGoCacheAccessors(caches, "app", "flash")
	if !strings.Contains(out, "300*time.Second") {
		t.Errorf("TTL 5m must become 300*time.Second in Set:\n%s", out)
	}
}

func TestGenerateGoCacheAccessors_Tags(t *testing.T) {
	caches := []*CacheInfo{
		{QueryName: "get_user", CacheName: "UserCache", TTL: "1m", Tags: []string{"users", "profiles"}, KeyParams: []string{"id"}, KeyTypes: []string{"BIGINT"}},
	}
	out := GenerateGoCacheAccessors(caches, "app", "flash")
	if !strings.Contains(out, `TagUsers CacheTag = "users"`) {
		t.Errorf("expected typed tag constant TagUsers:\n%s", out)
	}
	if !strings.Contains(out, `[]string{"users", "profiles"}`) {
		t.Errorf("expected tags forwarded to Cache.Set:\n%s", out)
	}
}

func TestGenerateGoCacheAccessors_EmptyReturnsEmptyString(t *testing.T) {
	if out := GenerateGoCacheAccessors(nil, "app", "flash"); out != "" {
		t.Errorf("no caches must produce empty output, got %q", out)
	}
}

// Two distinct tag strings that PascalCase to the same identifier ("user-posts"
// and "user_posts" both -> "UserPosts") generate duplicate CacheTag constants,
// which will not compile.
func TestGenerateGoCacheAccessors_TagConstantsUnique(t *testing.T) {
	caches := []*CacheInfo{
		{QueryName: "a", CacheName: "ACache", TTL: "1m", Tags: []string{"user-posts"}, KeyParams: []string{"id"}, KeyTypes: []string{"INT"}},
		{QueryName: "b", CacheName: "BCache", TTL: "1m", Tags: []string{"user_posts"}, KeyParams: []string{"id"}, KeyTypes: []string{"INT"}},
	}
	out := GenerateGoCacheAccessors(caches, "app", "flash")
	if n := strings.Count(out, "TagUserPosts CacheTag"); n > 1 {
		t.Errorf("duplicate CacheTag constant TagUserPosts generated %d times (won't compile):\n%s", n, out)
	}
}
