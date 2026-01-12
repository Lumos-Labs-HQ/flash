package gencommon

import (
	"fmt"

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
