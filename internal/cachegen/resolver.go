package cachegen

import (
	"fmt"
	"regexp"
	"strconv"
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

		// A single-colon cache key is CacheName[:p1[:p2...]]. To target an exact
		// entry every key param must be reconstructable from the mutation's
		// params. If the cache key is composite (>=2 params) we cannot form it
		// from one param, so fall back to a prefix purge. Only a single-param
		// key supports an exact delete keyed on that one param.
		var matchedParam string
		if len(cache.KeyParams) == 1 {
			for _, mutParam := range mutationParams {
				if strings.EqualFold(cache.KeyParams[0], mutParam.Name) {
					matchedParam = mutParam.Name
					break
				}
			}
		}

		if matchedParam != "" {
			purges = append(purges, DependencyPurge{
				CacheName:   cache.CacheName,
				KeyParam:    matchedParam,
				PurgePrefix: false,
			})
		} else {
			// Composite key, or no exact single-param match: purge by prefix.
			purges = append(purges, DependencyPurge{
				CacheName:   cache.CacheName,
				KeyParam:    "",
				PurgePrefix: true,
			})
		}
	}
	return purges
}

// IsCachedQuery returns true if the query has a @cache annotation
func IsCachedQuery(q *parser.Query) bool {
	return q.CacheDef != nil
}

// IsMutationQuery returns true if the query is an :exec or :execresult
func IsMutationQuery(q *parser.Query) bool {
	cmd := strings.ToLower(q.Cmd)
	return cmd == ":exec" || cmd == "exec" || cmd == ":execresult" || cmd == "execresult"
}

// ValidateDeps checks that all dep references point to actual query names
func ValidateDeps(caches []*CacheInfo, queries []*parser.Query) []string {
	queryNames := make(map[string]bool)
	for _, q := range queries {
		queryNames[strings.ToLower(q.Name)] = true
	}

	var warnings []string
	for _, cache := range caches {
		for _, dep := range cache.Dep {
			if !queryNames[strings.ToLower(dep)] {
				warnings = append(warnings, fmt.Sprintf("@cache on %s: dep \"%s\" does not match any query name", cache.QueryName, dep))
			}
		}
	}
	return warnings
}

// GetCacheInfoForQuery returns the CacheInfo for a given query, or nil
func GetCacheInfoForQuery(queryName string, caches []*CacheInfo) *CacheInfo {
	for _, c := range caches {
		if strings.EqualFold(c.QueryName, queryName) {
			return c
		}
	}
	return nil
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

// ttlTokenRegex matches one "<number><unit>" segment, e.g. "5m", "1h", "1.5h".
var ttlTokenRegex = regexp.MustCompile(`([0-9]+\.?[0-9]*)([a-z]+)`)

func ttlUnitSeconds(unit string) (float64, bool) {
	switch unit {
	case "s", "sec", "second", "seconds":
		return 1, true
	case "m", "min", "minute", "minutes":
		return 60, true
	case "h", "hr", "hour", "hours":
		return 3600, true
	case "d", "day", "days":
		return 86400, true
	}
	return 0, false
}

// ParseTTL converts a TTL string to seconds. It supports:
//   - a bare number ("90"), interpreted as seconds
//   - a single unit ("30s", "5m", "1h", "2d"), case-insensitively
//   - decimals ("1.5h" -> 5400)
//   - compound durations ("1h30m" -> 5400, "1m30s" -> 90)
//
// Unparseable input ("abc"), unknown units ("5x"), and negative values ("-5m")
// return 0 rather than a silently-truncated or negative TTL.
func ParseTTL(ttl string) (seconds int64) {
	ttl = strings.TrimSpace(strings.ToLower(ttl))
	if ttl == "" {
		return 0
	}

	// A bare number is a count of seconds.
	if n, err := strconv.ParseInt(ttl, 10, 64); err == nil {
		if n < 0 {
			return 0
		}
		return n
	}

	// Otherwise parse a sequence of <number><unit> segments. Whitespace between
	// segments is ignored; anything else (a leading '-', stray letters) makes
	// the whole value invalid.
	stripped := strings.ReplaceAll(ttl, " ", "")
	segments := ttlTokenRegex.FindAllStringSubmatch(stripped, -1)
	if len(segments) == 0 {
		return 0
	}

	var total float64
	var rebuilt strings.Builder
	for _, seg := range segments {
		mult, ok := ttlUnitSeconds(seg[2])
		if !ok {
			return 0 // unknown unit, e.g. "5x"
		}
		val, err := strconv.ParseFloat(seg[1], 64)
		if err != nil {
			return 0
		}
		total += val * mult
		rebuilt.WriteString(seg[1])
		rebuilt.WriteString(seg[2])
	}
	// If the matched segments don't account for the entire (space-stripped)
	// input, there was garbage between/around them — reject.
	if rebuilt.String() != stripped {
		return 0
	}
	if total < 0 {
		return 0
	}
	return int64(total)
}

// collectUniqueTags extracts all unique tag names from cache definitions, sorted.
func collectUniqueTags(caches []*CacheInfo) []string {
	seen := make(map[string]bool)
	for _, c := range caches {
		for _, t := range c.Tags {
			seen[t] = true
		}
	}
	tags := make([]string, 0, len(seen))
	for t := range seen {
		tags = append(tags, t)
	}
	// Sort for deterministic output
	for i := 0; i < len(tags); i++ {
		for j := i + 1; j < len(tags); j++ {
			if tags[i] > tags[j] {
				tags[i], tags[j] = tags[j], tags[i]
			}
		}
	}
	return tags
}
