package cachegen

import (
	"fmt"
	"strings"

	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/Lumos-Labs-HQ/flash/internal/parser"
	"github.com/Lumos-Labs-HQ/flash/internal/utils"
)

// CacheInfo holds resolved cache metadata for a query
type CacheInfo struct {
	QueryName  string
	CacheName  string   // accessor name (e.g. "UserCache")
	TTL        string   // e.g. "30s", "5m"
	Tags       []string // for bulk purge
	Dep        []string // queries that invalidate this cache
	KeyParams  []string // param names that form the cache key
	KeyTypes   []string // param types (for function signatures)
}

// DependencyPurge holds info about what to purge after a mutation runs
type DependencyPurge struct {
	CacheName   string // e.g. "UserCache"
	KeyParam    string // param name in the mutation that matches the cache key
	PurgePrefix bool   // true = purge by prefix (partial key match)
}

// ResolveCacheQueries extracts cache metadata from parsed queries
func ResolveCacheQueries(queries []*parser.Query, cfg *config.CacheConfig) []*CacheInfo {
	if !cfg.Enabled {
		return nil
	}

	var caches []*CacheInfo
	for _, q := range queries {
		if q.CacheDef == nil {
			continue
		}

		cacheName := q.CacheDef.Name
		if cacheName == "" {
			cacheName = utils.ToPascalCase(q.Name) + "Cache"
		}

		ttl := q.CacheDef.TTL
		if ttl == "" {
			ttl = cfg.DefaultTTL
		}

		// Key params = all params of the cached query
		var keyParams []string
		var keyTypes []string
		for _, p := range q.Params {
			keyParams = append(keyParams, p.Name)
			keyTypes = append(keyTypes, p.Type)
		}

		caches = append(caches, &CacheInfo{
			QueryName: q.Name,
			CacheName: cacheName,
			TTL:       ttl,
			Tags:      q.CacheDef.Tags,
			Dep:       q.CacheDef.Dep,
			KeyParams: keyParams,
			KeyTypes:  keyTypes,
		})
	}
	return caches
}

// ResolveDependencyPurges returns the list of purge actions for a given mutation query name
func ResolveDependencyPurges(mutationName string, caches []*CacheInfo, mutationParams []*parser.Param) []DependencyPurge {
	var purges []DependencyPurge

	for _, cache := range caches {
		isDep := false
		for _, dep := range cache.Dep {
			if strings.EqualFold(dep, mutationName) {
				isDep = true
				break
			}
		}
		if !isDep {
			continue
		}

		// Try to match cache key params with mutation params
		matched := false
		for _, cacheKeyParam := range cache.KeyParams {
			for _, mutParam := range mutationParams {
				if strings.EqualFold(cacheKeyParam, mutParam.Name) {
					purges = append(purges, DependencyPurge{
						CacheName:   cache.CacheName,
						KeyParam:    mutParam.Name,
						PurgePrefix: false,
					})
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}

		// If no exact param match, purge by prefix (wildcard)
		if !matched {
			purges = append(purges, DependencyPurge{
				CacheName:   cache.CacheName,
				KeyParam:    "",
				PurgePrefix: true,
			})
		}
	}
	return purges
}

// BuildCacheKey generates the cache key format string for a given cache
func BuildCacheKey(prefix string, cacheName string, params []string) string {
	if len(params) == 0 {
		return fmt.Sprintf("%s:%s", prefix, cacheName)
	}
	parts := []string{prefix, cacheName}
	parts = append(parts, params...)
	return strings.Join(parts, ":")
}

// ParseTTL converts a TTL string to language-specific duration expressions
func ParseTTL(ttl string) (seconds int64) {
	ttl = strings.TrimSpace(strings.ToLower(ttl))
	var value int64
	var unit string
	fmt.Sscanf(ttl, "%d%s", &value, &unit)
	switch unit {
	case "s", "sec", "second", "seconds":
		return value
	case "m", "min", "minute", "minutes":
		return value * 60
	case "h", "hr", "hour", "hours":
		return value * 3600
	case "d", "day", "days":
		return value * 86400
	default:
		return value // assume seconds
	}
}
