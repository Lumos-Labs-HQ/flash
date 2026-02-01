package gencommon

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Lumos-Labs-HQ/flash/internal/parser"
)

const cacheFileName = ".flash_cache.json"

// GenerationCache tracks file checksums and dependencies for incremental generation
type GenerationCache struct {
	Version string `json:"version"` // Cache format version for compatibility

	// File checksums (SHA256)
	QueryFileChecksums map[string]string `json:"query_file_checksums"` // filename → hash
	SchemaChecksum     string            `json:"schema_checksum"`      // schema hash
	ConfigChecksum     string            `json:"config_checksum"`      // config hash

	// Dependency tracking (which queries use which tables)
	QueryTableDeps map[string][]string `json:"query_table_deps"` // query file → table names

	// Generated file checksums (to detect manual edits)
	GeneratedFileChecksums map[string]string `json:"generated_file_checksums"` // generated file → hash

	// Metadata
	LastGeneration time.Time `json:"last_generation"`
	
	mu sync.RWMutex
}

// NewGenerationCache creates a new cache or loads existing one
func NewGenerationCache() *GenerationCache {
	cache := &GenerationCache{
		Version:                "1.0",
		QueryFileChecksums:     make(map[string]string),
		SchemaChecksum:         "",
		ConfigChecksum:         "",
		QueryTableDeps:         make(map[string][]string),
		GeneratedFileChecksums: make(map[string]string),
		LastGeneration:         time.Time{},
	}

	// Try to load existing cache
	_ = cache.Load()
	return cache
}

// ComputeFileChecksum computes SHA256 hash of a file
func ComputeFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// ComputeSchemaChecksum computes combined hash of all schema files
func (c *GenerationCache) ComputeSchemaChecksum(schemaDir string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get all SQL files in schema directory
	pattern := filepath.Join(schemaDir, "*.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}

	// Also check for single schema file
	schemaFile := filepath.Join(schemaDir, "../schema.sql")
	if info, err := os.Stat(schemaFile); err == nil && !info.IsDir() {
		files = append(files, schemaFile)
	}

	if len(files) == 0 {
		return "", nil
	}

	// Combine all file hashes
	hash := sha256.New()
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		hash.Write(content)
		hash.Write([]byte(filepath.Base(file))) // Include filename in hash
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// ShouldRegenerateQuery checks if a query file needs regeneration
func (c *GenerationCache) ShouldRegenerateQuery(queryFile string, currentHash string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cachedHash, exists := c.QueryFileChecksums[queryFile]
	if !exists {
		return true // New file
	}

	return cachedHash != currentHash // Changed file
}

// ShouldRegenerateAll checks if full regeneration is needed
func (c *GenerationCache) ShouldRegenerateAll(schemaHash, configHash string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if schema changed
	if c.SchemaChecksum != schemaHash {
		return true
	}

	// Check if config changed
	if c.ConfigChecksum != configHash {
		return true
	}

	return false
}

// GetAffectedQueries returns query files that depend on changed tables
func (c *GenerationCache) GetAffectedQueries(changedTables []string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	affectedSet := make(map[string]bool)
	
	for queryFile, tables := range c.QueryTableDeps {
		for _, table := range tables {
			for _, changedTable := range changedTables {
				if table == changedTable {
					affectedSet[queryFile] = true
					break
				}
			}
		}
	}

	affected := make([]string, 0, len(affectedSet))
	for queryFile := range affectedSet {
		affected = append(affected, queryFile)
	}
	return affected
}

// DetectSchemaChanges detects which tables changed between schema versions
func DetectSchemaChanges(oldSchema, newSchema *parser.Schema) []string {
	if oldSchema == nil || newSchema == nil {
		return nil
	}

	changed := make(map[string]bool)

	// Build old schema table map
	oldTables := make(map[string]*parser.Table)
	for _, table := range oldSchema.Tables {
		oldTables[table.Name] = table
	}

	// Check for new or modified tables
	for _, newTable := range newSchema.Tables {
		oldTable, exists := oldTables[newTable.Name]
		if !exists {
			// New table
			changed[newTable.Name] = true
			continue
		}

		// Check if table structure changed
		if tableChanged(oldTable, newTable) {
			changed[newTable.Name] = true
		}
	}

	// Check for deleted tables
	for _, oldTable := range oldSchema.Tables {
		found := false
		for _, newTable := range newSchema.Tables {
			if newTable.Name == oldTable.Name {
				found = true
				break
			}
		}
		if !found {
			changed[oldTable.Name] = true
		}
	}

	result := make([]string, 0, len(changed))
	for tableName := range changed {
		result = append(result, tableName)
	}
	return result
}

// tableChanged checks if table structure changed
func tableChanged(old, new *parser.Table) bool {
	if len(old.Columns) != len(new.Columns) {
		return true
	}

	// Build column map
	oldCols := make(map[string]*parser.Column)
	for _, col := range old.Columns {
		oldCols[col.Name] = col
	}

	for _, newCol := range new.Columns {
		oldCol, exists := oldCols[newCol.Name]
		if !exists {
			return true // New column
		}

		// Check if column properties changed
		if oldCol.Type != newCol.Type || oldCol.Nullable != newCol.Nullable {
			return true
		}
	}

	return false
}

// UpdateQueryChecksum updates the checksum for a query file
func (c *GenerationCache) UpdateQueryChecksum(queryFile, hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.QueryFileChecksums[queryFile] = hash
}

// UpdateSchemaChecksum updates the schema checksum
func (c *GenerationCache) UpdateSchemaChecksum(hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.SchemaChecksum = hash
}

// UpdateConfigChecksum updates the config checksum
func (c *GenerationCache) UpdateConfigChecksum(hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ConfigChecksum = hash
}

// UpdateQueryDependencies updates which tables a query depends on
func (c *GenerationCache) UpdateQueryDependencies(queryFile string, tables []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.QueryTableDeps[queryFile] = tables
}

// UpdateGeneratedFileChecksum updates checksum of generated file
func (c *GenerationCache) UpdateGeneratedFileChecksum(genFile, hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.GeneratedFileChecksums[genFile] = hash
}

// MarkGeneration updates the last generation timestamp
func (c *GenerationCache) MarkGeneration() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastGeneration = time.Now()
}

// Save persists the cache to disk
func (c *GenerationCache) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cacheFile := filepath.Join("flash_gen", cacheFileName)
	
	// Ensure directory exists
	if err := os.MkdirAll("flash_gen", 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// Load reads the cache from disk
func (c *GenerationCache) Load() error {
	cacheFile := filepath.Join("flash_gen", cacheFileName)

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache file yet
		}
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := json.Unmarshal(data, c); err != nil {
		return err
	}

	// Validate cache version
	if c.Version != "1.0" {
		// Cache version mismatch, invalidate
		c.QueryFileChecksums = make(map[string]string)
		c.SchemaChecksum = ""
		c.ConfigChecksum = ""
		c.QueryTableDeps = make(map[string][]string)
		c.GeneratedFileChecksums = make(map[string]string)
		c.Version = "1.0"
	}

	return nil
}

// Clear removes all cache data
func (c *GenerationCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.QueryFileChecksums = make(map[string]string)
	c.SchemaChecksum = ""
	c.ConfigChecksum = ""
	c.QueryTableDeps = make(map[string][]string)
	c.GeneratedFileChecksums = make(map[string]string)
	c.LastGeneration = time.Time{}
}
