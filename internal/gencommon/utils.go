package gencommon

import (
	"fmt"
	"os"

	"github.com/Lumos-Labs-HQ/flash/internal/parser"
	"github.com/Lumos-Labs-HQ/flash/internal/utils"
)

// ExtractTableDependencies extracts which tables a set of queries depends on
// This is shared by all generators (Go, JS, Python)
func ExtractTableDependencies(queries []*parser.Query) []string {
	tableSet := make(map[string]bool)

	for _, query := range queries {
		tableName := utils.ExtractTableName(query.SQL)
		if tableName != "" {
			tableSet[tableName] = true
		}

		for _, col := range query.Columns {
			if col.Table != "" {
				tableSet[col.Table] = true
			}
		}
	}

	tables := make([]string, 0, len(tableSet))
	for table := range tableSet {
		tables = append(tables, table)
	}
	return tables
}

// ShouldRegenerateFile checks if a file needs regeneration based on cache
func ShouldRegenerateFile(cache *GenerationCache, queryFile, currentHash string, fullRegen bool) bool {
	if fullRegen {
		return true
	}
	return cache.ShouldRegenerateQuery(queryFile, currentHash)
}

// ShouldRegenerateFileForOutput additionally requires that at least one of the
// given output files was previously generated (and still exists on disk).
// Multiple generators can share one output directory and therefore one
// .flash_cache.json: the query-file checksum alone can be satisfied by ANOTHER
// generator's entry, causing this generator to skip files it never wrote.
// Requiring our own recorded output entry (and its presence on disk) makes
// the skip decision generator-local.
func ShouldRegenerateFileForOutput(cache *GenerationCache, queryFile, currentHash string, fullRegen bool, outputPaths ...string) bool {
	if fullRegen {
		return true
	}
	if !cache.ShouldRegenerateQuery(queryFile, currentHash) {
		// Query hash matches — but only trust it if one of our own outputs is
		// recorded in the cache and still present.
		for _, p := range outputPaths {
			if _, ok := cache.GeneratedFileChecksums[p]; ok {
				if _, err := os.Stat(p); err == nil {
					return false
				}
			}
		}
		// None of our outputs exist: regenerate.
		return true
	}
	return true
}

// PrintSkipMessage prints a skip message for unchanged files
func PrintSkipMessage(sourceFile, extension string) {
	fmt.Printf("⏭️  Skipping %s%s (unchanged)\n", sourceFile, extension)
}

// PrintGenerateMessage prints a generation message
func PrintGenerateMessage(sourceFile, extension string) {
	fmt.Printf("🔄 Generating %s%s\n", sourceFile, extension)
}

// UpdateCacheForFile updates cache after generating a file
func UpdateCacheForFile(cache *GenerationCache, queryFile, currentHash string, tableDeps []string, generatedPath string) {
	cache.UpdateQueryChecksum(queryFile, currentHash)
	cache.UpdateQueryDependencies(queryFile, tableDeps)

	if genHash, err := ComputeFileChecksum(generatedPath); err == nil {
		cache.UpdateGeneratedFileChecksum(generatedPath, genHash)
	}
}
