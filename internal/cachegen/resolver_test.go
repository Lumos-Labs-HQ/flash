package cachegen

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/parser"
)

// mkParams builds a []*parser.Param from names (ParamNum assigned in order).
func mkParams(names ...string) []*parser.Param {
	ps := make([]*parser.Param, len(names))
	for i, n := range names {
		ps[i] = &parser.Param{Name: n, ParamNum: i + 1}
	}
	return ps
}

// ---------------------------------------------------------------------------
// ParseTTL
// ---------------------------------------------------------------------------

func TestParseTTL_SingleUnits(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"30s", 30},
		{"45sec", 45},
		{"10second", 10},
		{"15seconds", 15},
		{"5m", 300},
		{"5min", 300},
		{"2minute", 120},
		{"2minutes", 120},
		{"1h", 3600},
		{"2hr", 7200},
		{"1hour", 3600},
		{"3hours", 10800},
		{"2d", 172800},
		{"1day", 86400},
		{"3days", 259200},
		{"90", 90},      // bare number == seconds
		{"  5m  ", 300}, // surrounding whitespace
		{"1H", 3600},    // case-insensitive
		{"", 0},
	}
	for _, tt := range tests {
		if got := ParseTTL(tt.in); got != tt.want {
			t.Errorf("ParseTTL(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

// Compound and decimal durations are natural user inputs. They must NOT be
// silently truncated to 1 second. (Current code returns 1 for both.)
func TestParseTTL_CompoundAndDecimal(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"1h30m", 5400},
		{"90m", 5400},
		{"2h15m", 8100},
		{"1m30s", 90},
		{"1.5h", 5400},
		{"0.5h", 1800},
	}
	for _, tt := range tests {
		if got := ParseTTL(tt.in); got != tt.want {
			t.Errorf("ParseTTL(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

// A cache TTL must never be negative and unparseable input must not silently
// produce a live cache with an arbitrary duration.
func TestParseTTL_RejectsBadInput(t *testing.T) {
	if got := ParseTTL("-5m"); got < 0 {
		t.Errorf("ParseTTL(%q) = %d, want >= 0 (negative TTL is invalid)", "-5m", got)
	}
	if got := ParseTTL("abc"); got != 0 {
		t.Errorf("ParseTTL(%q) = %d, want 0 for unparseable input", "abc", got)
	}
	if got := ParseTTL("5x"); got != 0 {
		t.Errorf("ParseTTL(%q) = %d, want 0 for unknown unit", "5x", got)
	}
}

// ---------------------------------------------------------------------------
// BuildCacheKey
// ---------------------------------------------------------------------------

func TestBuildCacheKey(t *testing.T) {
	tests := []struct {
		prefix string
		name   string
		params []string
		want   string
	}{
		{"flash", "UserCache", nil, "flash:UserCache"},
		{"flash", "UserCache", []string{}, "flash:UserCache"},
		{"flash", "UserCache", []string{"1"}, "flash:UserCache:1"},
		{"flash", "UserCache", []string{"1", "2"}, "flash:UserCache:1:2"},
		{"app", "OrgCache", []string{"7", "9", "x"}, "app:OrgCache:7:9:x"},
	}
	for _, tt := range tests {
		if got := BuildCacheKey(tt.prefix, tt.name, tt.params); got != tt.want {
			t.Errorf("BuildCacheKey(%q,%q,%v) = %q, want %q", tt.prefix, tt.name, tt.params, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// collectUniqueTags
// ---------------------------------------------------------------------------

func TestCollectUniqueTags(t *testing.T) {
	caches := []*CacheInfo{
		{Tags: []string{"b", "a"}},
		{Tags: []string{"a", "c"}},
		{Tags: nil},
		{Tags: []string{"c"}},
	}
	got := collectUniqueTags(caches)
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("collectUniqueTags = %v, want %v", got, want)
	}
}

func TestCollectUniqueTags_Empty(t *testing.T) {
	if got := collectUniqueTags(nil); len(got) != 0 {
		t.Errorf("collectUniqueTags(nil) = %v, want empty", got)
	}
}

// ---------------------------------------------------------------------------
// GetCacheInfoForQuery / IsMutationQuery / IsCachedQuery
// ---------------------------------------------------------------------------

func TestGetCacheInfoForQuery(t *testing.T) {
	caches := []*CacheInfo{
		{QueryName: "GetUser", CacheName: "UserCache"},
		{QueryName: "GetOrg", CacheName: "OrgCache"},
	}
	if c := GetCacheInfoForQuery("getuser", caches); c == nil || c.CacheName != "UserCache" {
		t.Errorf("case-insensitive lookup failed: %+v", c)
	}
	if c := GetCacheInfoForQuery("GetOrg", caches); c == nil || c.CacheName != "OrgCache" {
		t.Errorf("exact lookup failed: %+v", c)
	}
	if c := GetCacheInfoForQuery("missing", caches); c != nil {
		t.Errorf("expected nil for missing query, got %+v", c)
	}
}

func TestIsMutationQuery(t *testing.T) {
	cases := map[string]bool{
		":exec":       true,
		"exec":        true,
		":execresult": true,
		"execresult":  true,
		"EXEC":        true,
		":one":        false,
		":many":       false,
		"":            false,
	}
	for cmd, want := range cases {
		if got := IsMutationQuery(&parser.Query{Cmd: cmd}); got != want {
			t.Errorf("IsMutationQuery(%q) = %v, want %v", cmd, got, want)
		}
	}
}

func TestIsCachedQuery(t *testing.T) {
	if IsCachedQuery(&parser.Query{}) {
		t.Error("query without CacheDef must not be cached")
	}
	if !IsCachedQuery(&parser.Query{CacheDef: &parser.CacheDef{}}) {
		t.Error("query with CacheDef must be cached")
	}
}

// ---------------------------------------------------------------------------
// ValidateDeps
// ---------------------------------------------------------------------------

func TestValidateDeps(t *testing.T) {
	queries := []*parser.Query{{Name: "GetUser"}, {Name: "UpdateUser"}}
	caches := []*CacheInfo{
		{QueryName: "GetUser", Dep: []string{"updateuser"}},    // valid (case-insensitive)
		{QueryName: "GetUser", Dep: []string{"DeleteUserXYZ"}}, // invalid
	}
	warnings := ValidateDeps(caches, queries)
	if len(warnings) != 1 {
		t.Fatalf("got %d warnings, want 1: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "DeleteUserXYZ") {
		t.Errorf("warning should name the bad dep, got: %q", warnings[0])
	}
}

func TestValidateDeps_AllValid(t *testing.T) {
	queries := []*parser.Query{{Name: "A"}, {Name: "B"}}
	caches := []*CacheInfo{{QueryName: "A", Dep: []string{"B"}}}
	if w := ValidateDeps(caches, queries); len(w) != 0 {
		t.Errorf("expected no warnings, got %v", w)
	}
}

// ---------------------------------------------------------------------------
// ResolveCacheQueries
// ---------------------------------------------------------------------------

func TestResolveCacheQueries_Defaults(t *testing.T) {
	cfg := &config.CacheConfig{Enabled: true, DefaultTTL: "5m"}
	queries := []*parser.Query{
		{
			Name:     "get_user",
			Cmd:      ":one",
			Params:   []*parser.Param{{Name: "id", Type: "BIGINT"}},
			CacheDef: &parser.CacheDef{Tags: []string{"users"}},
		},
		{Name: "no_cache", Cmd: ":one"}, // no CacheDef -> skipped
	}
	caches := ResolveCacheQueries(queries, cfg)
	if len(caches) != 1 {
		t.Fatalf("got %d caches, want 1", len(caches))
	}
	c := caches[0]
	if c.QueryName != "get_user" {
		t.Errorf("QueryName = %q, want get_user", c.QueryName)
	}
	if c.CacheName != "GetUserCache" {
		t.Errorf("CacheName = %q, want GetUserCache (default derived from query name)", c.CacheName)
	}
	if c.TTL != "5m" {
		t.Errorf("TTL = %q, want 5m (config default applied)", c.TTL)
	}
	if !reflect.DeepEqual(c.KeyParams, []string{"id"}) {
		t.Errorf("KeyParams = %v, want [id]", c.KeyParams)
	}
	if !reflect.DeepEqual(c.KeyTypes, []string{"BIGINT"}) {
		t.Errorf("KeyTypes = %v, want [BIGINT]", c.KeyTypes)
	}
}

func TestResolveCacheQueries_CustomNameAndTTL(t *testing.T) {
	cfg := &config.CacheConfig{Enabled: true, DefaultTTL: "5m"}
	queries := []*parser.Query{
		{
			Name:     "get_user",
			Cmd:      ":one",
			CacheDef: &parser.CacheDef{Name: "MyUser", TTL: "30s"},
		},
	}
	caches := ResolveCacheQueries(queries, cfg)
	if len(caches) != 1 {
		t.Fatalf("got %d caches, want 1", len(caches))
	}
	if caches[0].CacheName != "MyUser" {
		t.Errorf("CacheName = %q, want MyUser (custom name honored)", caches[0].CacheName)
	}
	if caches[0].TTL != "30s" {
		t.Errorf("TTL = %q, want 30s (explicit TTL honored over default)", caches[0].TTL)
	}
}

func TestResolveCacheQueries_DisabledReturnsNil(t *testing.T) {
	cfg := &config.CacheConfig{Enabled: false}
	queries := []*parser.Query{{Name: "x", CacheDef: &parser.CacheDef{}}}
	if caches := ResolveCacheQueries(queries, cfg); caches != nil {
		t.Errorf("disabled cache config must return nil, got %v", caches)
	}
}

// ---------------------------------------------------------------------------
// ResolveDependencyPurges
// ---------------------------------------------------------------------------

func TestResolveDependencyPurges_SingleKeyExactDelete(t *testing.T) {
	caches := []*CacheInfo{{
		CacheName: "UserCache",
		Dep:       []string{"UpdateUser"},
		KeyParams: []string{"id"},
	}}
	purges := ResolveDependencyPurges("UpdateUser", caches, mkParams("id", "name"))
	if len(purges) != 1 {
		t.Fatalf("got %d purges, want 1", len(purges))
	}
	if purges[0].PurgePrefix || purges[0].KeyParam != "id" {
		t.Errorf("want exact delete keyed on id, got %+v", purges[0])
	}
}

func TestResolveDependencyPurges_NoParamMatchPurgesByPrefix(t *testing.T) {
	caches := []*CacheInfo{{
		CacheName: "UserCache",
		Dep:       []string{"UpdateUser"},
		KeyParams: []string{"id"},
	}}
	purges := ResolveDependencyPurges("UpdateUser", caches, mkParams("email"))
	if len(purges) != 1 || !purges[0].PurgePrefix {
		t.Fatalf("no matching param must purge by prefix, got %+v", purges)
	}
}

func TestResolveDependencyPurges_NonDepProducesNothing(t *testing.T) {
	caches := []*CacheInfo{{
		CacheName: "UserCache",
		Dep:       []string{"UpdateUser"},
		KeyParams: []string{"id"},
	}}
	if purges := ResolveDependencyPurges("SomethingElse", caches, mkParams("id")); len(purges) != 0 {
		t.Errorf("non-dependent mutation must produce no purges, got %v", purges)
	}
}

// A composite (multi-param) cache key cannot be reconstructed from a single
// mutation param, so an exact delete carrying only one param can never hit the
// real key. It must purge by prefix (or carry every key param). Current code
// emits a single-param exact delete -> this test flags it.
func TestResolveDependencyPurges_CompositeKeyCannotUseSingleParam(t *testing.T) {
	caches := []*CacheInfo{{
		CacheName: "OrgMemberCache",
		Dep:       []string{"UpdateOrgMember"},
		KeyParams: []string{"org_id", "user_id"},
	}}
	purges := ResolveDependencyPurges("UpdateOrgMember", caches, mkParams("org_id", "user_id", "role"))
	if len(purges) == 1 && !purges[0].PurgePrefix && purges[0].KeyParam != "" {
		t.Errorf("composite-key cache purged with single param %q — cannot form the 2-part key; expected prefix purge or all key params", purges[0].KeyParam)
	}
}
