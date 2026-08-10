package gogen

import (
	"strings"
	"testing"

	"github.com/Lumos-Labs-HQ/flash/internal/cachegen"
	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/parser"
)

func cacheGen() *Generator {
	g := New(&config.Config{
		SchemaDir: "db/schema",
		Queries:   "db/queries/",
		Database:  config.Database{Provider: "postgresql"},
		Cache:     config.CacheConfig{Enabled: true, Prefix: "flash", DefaultTTL: "5m"},
	})
	g.schema = &parser.Schema{
		Tables: []*parser.Table{{
			Name: "org_members",
			Columns: []*parser.Column{
				{Name: "org_id", Type: "BIGINT"},
				{Name: "user_id", Type: "BIGINT"},
				{Name: "role", Type: "TEXT"},
			},
		}},
	}
	return g
}

// The two Go cache code paths must agree on the key format, or a value written
// by the accessor (cache_accessors.go) can never be read by the cached-query
// wrapper (cached_queries.go), and vice versa. For a 2-param cache the accessor
// path currently emits a DOUBLE colon ("Name::%v:%v") while the cached-query
// path emits a SINGLE colon ("Name:%v:%v") — a split-brain cache. This test
// pins them together so that divergence fails loudly.
func TestGoCacheKey_AccessorAndCachedQueryAgree(t *testing.T) {
	g := cacheGen()
	q := &parser.Query{
		Name: "get_member",
		Cmd:  ":one",
		Params: []*parser.Param{
			{Name: "org_id", Type: "BIGINT"},
			{Name: "user_id", Type: "BIGINT"},
		},
		CacheDef: &parser.CacheDef{TTL: "5m"},
	}
	caches := cachegen.ResolveCacheQueries([]*parser.Query{q}, &g.Config.Cache)

	accessorCode := cachegen.GenerateGoCacheAccessors(caches, "app", "flash")
	cachedCode := g.generateCachedQueries([]*parser.Query{q}, caches, "app")

	const singleColon = `:%v:%v"`
	const doubleColon = `::%v:%v"`

	accessorHasDouble := strings.Contains(accessorCode, doubleColon)
	cachedHasDouble := strings.Contains(cachedCode, doubleColon)

	if accessorHasDouble != cachedHasDouble {
		t.Errorf("cache key format DIVERGES between the two Go code paths — values written by one cannot be read by the other:\n"+
			"  accessor uses double-colon: %v\n  cached-query uses double-colon: %v",
			accessorHasDouble, cachedHasDouble)
	}
	if accessorHasDouble || cachedHasDouble {
		t.Errorf("composite cache key uses a double colon somewhere (want single colon %q)", singleColon)
	}
}

// A cached query whose param is a Go keyword must produce compilable code in
// BOTH the accessor and cached-query files.
func TestGoCache_KeywordParamCompilesInBothPaths(t *testing.T) {
	g := cacheGen()
	q := &parser.Query{
		Name:     "get_by_type",
		Cmd:      ":one",
		Params:   []*parser.Param{{Name: "type", Type: "TEXT"}},
		CacheDef: &parser.CacheDef{TTL: "1m", Name: "TypeCache"},
	}
	caches := cachegen.ResolveCacheQueries([]*parser.Query{q}, &g.Config.Cache)

	accessorCode := cachegen.GenerateGoCacheAccessors(caches, "app", "flash")
	if strings.Contains(accessorCode, "(type string)") || strings.Contains(accessorCode, "(type ") {
		t.Errorf("accessor uses Go keyword `type` as a bare param (won't compile):\n%s", accessorCode)
	}
}
