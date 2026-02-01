package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Service struct {
	client *redis.Client
	ctx    context.Context
}

type KeyInfo struct {
	Key   string      `json:"key"`
	Type  string      `json:"type"`
	TTL   int64       `json:"ttl"`
	Value interface{} `json:"value,omitempty"`
	Size  int64       `json:"size,omitempty"`
}

type KeysResult struct {
	Keys       []KeyInfo `json:"keys"`
	TotalCount int64     `json:"total_count"`
	Cursor     uint64    `json:"cursor"`
}

type ServerInfo struct {
	Version          string `json:"version"`
	Mode             string `json:"mode"`
	OS               string `json:"os"`
	Uptime           int64  `json:"uptime"`
	ConnectedClients int64  `json:"connected_clients"`
	UsedMemory       string `json:"used_memory"`
	PeakMemory       string `json:"peak_memory"`
	MaxMemory        string `json:"max_memory"`
	TotalKeys        int64  `json:"total_keys"`
	CurrentDB        int    `json:"current_db"`
}

type CLIResult struct {
	Command  string      `json:"command"`
	Result   interface{} `json:"result"`
	Duration string      `json:"duration"`
	Error    string      `json:"error,omitempty"`
}

func NewService(client *redis.Client) *Service {
	return &Service{
		client: client,
		ctx:    context.Background(),
	}
}

// GetInfo returns Redis server information
func (s *Service) GetInfo() (*ServerInfo, error) {
	info, err := s.client.Info(s.ctx).Result()
	if err != nil {
		return nil, err
	}

	serverInfo := &ServerInfo{}
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "redis_version:") {
			serverInfo.Version = strings.TrimPrefix(line, "redis_version:")
			serverInfo.Version = strings.TrimSpace(serverInfo.Version)
		} else if strings.HasPrefix(line, "redis_mode:") {
			serverInfo.Mode = strings.TrimPrefix(line, "redis_mode:")
			serverInfo.Mode = strings.TrimSpace(serverInfo.Mode)
		} else if strings.HasPrefix(line, "os:") {
			serverInfo.OS = strings.TrimPrefix(line, "os:")
			serverInfo.OS = strings.TrimSpace(serverInfo.OS)
		} else if strings.HasPrefix(line, "uptime_in_seconds:") {
			uptimeStr := strings.TrimPrefix(line, "uptime_in_seconds:")
			uptimeStr = strings.TrimSpace(uptimeStr)
			serverInfo.Uptime, _ = strconv.ParseInt(uptimeStr, 10, 64)
		} else if strings.HasPrefix(line, "connected_clients:") {
			clientsStr := strings.TrimPrefix(line, "connected_clients:")
			clientsStr = strings.TrimSpace(clientsStr)
			serverInfo.ConnectedClients, _ = strconv.ParseInt(clientsStr, 10, 64)
		} else if strings.HasPrefix(line, "used_memory_human:") {
			serverInfo.UsedMemory = strings.TrimPrefix(line, "used_memory_human:")
			serverInfo.UsedMemory = strings.TrimSpace(serverInfo.UsedMemory)
		} else if strings.HasPrefix(line, "used_memory_peak_human:") {
			serverInfo.PeakMemory = strings.TrimPrefix(line, "used_memory_peak_human:")
			serverInfo.PeakMemory = strings.TrimSpace(serverInfo.PeakMemory)
		}
	}

	dbSize, err := s.client.DBSize(s.ctx).Result()
	if err == nil {
		serverInfo.TotalKeys = dbSize
	}

	maxmemory, err := s.client.ConfigGet(s.ctx, "maxmemory").Result()
	if err == nil && len(maxmemory) > 0 {
		if maxMemVal, ok := maxmemory["maxmemory"]; ok {
			maxMemBytes, _ := strconv.ParseInt(maxMemVal, 10, 64)
			if maxMemBytes == 0 {
				serverInfo.MaxMemory = "Unlimited"
			} else {
				serverInfo.MaxMemory = formatBytes(maxMemBytes)
			}
		}
	}
	if serverInfo.MaxMemory == "" {
		serverInfo.MaxMemory = "Unlimited"
	}

	return serverInfo, nil
}

// formatBytes converts bytes to human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// GetDBSize returns the number of keys in current database
func (s *Service) GetDBSize() (int64, error) {
	return s.client.DBSize(s.ctx).Result()
}

// GetKeys returns keys matching pattern with pagination
func (s *Service) GetKeys(pattern string, cursor uint64, count int64) (*KeysResult, error) {
	if pattern == "" {
		pattern = "*"
	}

	keys, nextCursor, err := s.client.Scan(s.ctx, cursor, pattern, count).Result()
	if err != nil {
		return nil, err
	}

	sort.Strings(keys)

	keyInfos := make([]KeyInfo, 0, len(keys))
	for _, key := range keys {
		keyType, err := s.client.Type(s.ctx, key).Result()
		if err != nil {
			continue
		}

		ttl, err := s.client.TTL(s.ctx, key).Result()
		ttlSeconds := int64(-1)
		if err == nil {
			switch ttl {
			case -1:
				ttlSeconds = -1 // No expiry
			case -2:
				ttlSeconds = -2 // Key doesn't exist
			default:
				ttlSeconds = int64(ttl.Seconds())
			}
		}

		keyInfos = append(keyInfos, KeyInfo{
			Key:  key,
			Type: keyType,
			TTL:  ttlSeconds,
		})
	}

	totalCount, _ := s.client.DBSize(s.ctx).Result()

	return &KeysResult{
		Keys:       keyInfos,
		TotalCount: totalCount,
		Cursor:     nextCursor,
	}, nil
}

// GetKey returns the value of a key
func (s *Service) GetKey(key string) (*KeyInfo, error) {
	keyType, err := s.client.Type(s.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if keyType == "none" {
		return nil, fmt.Errorf("key does not exist")
	}

	ttl, _ := s.client.TTL(s.ctx, key).Result()
	ttlSeconds := int64(-1)
	if ttl >= 0 {
		ttlSeconds = int64(ttl.Seconds())
	}

	keyInfo := &KeyInfo{
		Key:  key,
		Type: keyType,
		TTL:  ttlSeconds,
	}

	// Get value based on type
	switch keyType {
	case "string":
		val, err := s.client.Get(s.ctx, key).Result()
		if err != nil {
			return nil, err
		}
		keyInfo.Value = val
		keyInfo.Size = int64(len(val))

	case "list":
		val, err := s.client.LRange(s.ctx, key, 0, -1).Result()
		if err != nil {
			return nil, err
		}
		keyInfo.Value = val
		keyInfo.Size = int64(len(val))

	case "set":
		val, err := s.client.SMembers(s.ctx, key).Result()
		if err != nil {
			return nil, err
		}
		keyInfo.Value = val
		keyInfo.Size = int64(len(val))

	case "zset":
		val, err := s.client.ZRangeWithScores(s.ctx, key, 0, -1).Result()
		if err != nil {
			return nil, err
		}
		result := make([]map[string]interface{}, len(val))
		for i, z := range val {
			result[i] = map[string]interface{}{
				"member": z.Member,
				"score":  z.Score,
			}
		}
		keyInfo.Value = result
		keyInfo.Size = int64(len(val))

	case "hash":
		val, err := s.client.HGetAll(s.ctx, key).Result()
		if err != nil {
			return nil, err
		}
		keyInfo.Value = val
		keyInfo.Size = int64(len(val))

	case "stream":
		val, err := s.client.XRange(s.ctx, key, "-", "+").Result()
		if err != nil {
			return nil, err
		}
		keyInfo.Value = val
		keyInfo.Size = int64(len(val))

	default:
		keyInfo.Value = fmt.Sprintf("Unsupported type: %s", keyType)
	}

	return keyInfo, nil
}

// SetKey creates or updates a key
func (s *Service) SetKey(key string, value interface{}, keyType string, ttl int64) error {
	switch keyType {
	case "string":
		strVal, ok := value.(string)
		if !ok {
			jsonBytes, err := json.Marshal(value)
			if err != nil {
				return fmt.Errorf("invalid string value")
			}
			strVal = string(jsonBytes)
		}
		var expiration time.Duration
		if ttl > 0 {
			expiration = time.Duration(ttl) * time.Second
		}
		return s.client.Set(s.ctx, key, strVal, expiration).Err()

	case "list":
		vals, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("invalid list value")
		}
		if len(vals) > 0 {
			s.client.Del(s.ctx, key)
			if err := s.client.RPush(s.ctx, key, vals...).Err(); err != nil {
				return err
			}
		}

	case "set":
		vals, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("invalid set value")
		}
		if len(vals) > 0 {
			s.client.Del(s.ctx, key)
			if err := s.client.SAdd(s.ctx, key, vals...).Err(); err != nil {
				return err
			}
		}

	case "hash":
		hashVal, ok := value.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid hash value")
		}
		// Only delete and recreate if we have values to set
		if len(hashVal) > 0 {
			s.client.Del(s.ctx, key)
			args := make([]interface{}, 0, len(hashVal)*2)
			for k, v := range hashVal {
				args = append(args, k, v)
			}
			if err := s.client.HSet(s.ctx, key, args...).Err(); err != nil {
				return err
			}
		}

	case "zset":
		vals, ok := value.([]interface{})
		if !ok {
			return fmt.Errorf("invalid zset value")
		}
		members := make([]redis.Z, 0, len(vals))
		for _, v := range vals {
			if m, ok := v.(map[string]interface{}); ok {
				score, _ := m["score"].(float64)
				members = append(members, redis.Z{
					Score:  score,
					Member: m["member"],
				})
			}
		}
		// Only delete and recreate if we have members to set
		if len(members) > 0 {
			s.client.Del(s.ctx, key)
			if err := s.client.ZAdd(s.ctx, key, members...).Err(); err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("unsupported key type: %s", keyType)
	}

	if ttl > 0 && keyType != "string" {
		s.client.Expire(s.ctx, key, time.Duration(ttl)*time.Second)
	}

	return nil
}

// DeleteKey deletes a key
func (s *Service) DeleteKey(key string) error {
	result, err := s.client.Del(s.ctx, key).Result()
	if err != nil {
		return err
	}
	if result == 0 {
		return fmt.Errorf("key does not exist")
	}
	return nil
}

// BulkDeleteKeys deletes multiple keys
func (s *Service) BulkDeleteKeys(keys []string) (int64, error) {
	return s.client.Del(s.ctx, keys...).Result()
}

// GetTTL returns the TTL of a key
func (s *Service) GetTTL(key string) (int64, error) {
	ttl, err := s.client.TTL(s.ctx, key).Result()
	if err != nil {
		return -1, err
	}
	if ttl == -1 {
		return -1, nil
	}
	if ttl == -2 {
		return -2, nil
	}
	return int64(ttl.Seconds()), nil
}

func (s *Service) SetTTL(key string, ttl int64) error {
	if ttl <= 0 {
		return s.client.Persist(s.ctx, key).Err()
	}
	return s.client.Expire(s.ctx, key, time.Duration(ttl)*time.Second).Err()
}

func (s *Service) RenameKey(oldKey, newKey string) error {
	return s.client.Rename(s.ctx, oldKey, newKey).Err()
}

// ExecuteCLI executes a Redis CLI command
func (s *Service) ExecuteCLI(command string) *CLIResult {
	start := time.Now()
	result := &CLIResult{Command: command}

	parts := parseCommand(command)
	if len(parts) == 0 {
		result.Error = "empty command"
		result.Duration = time.Since(start).String()
		return result
	}

	args := make([]interface{}, len(parts))
	for i, p := range parts {
		args[i] = p
	}

	res, err := s.client.Do(s.ctx, args...).Result()
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Result = formatResult(res)
	}

	result.Duration = time.Since(start).String()
	return result
}

// SelectDatabase selects a different database
func (s *Service) SelectDatabase(db int) error {
	opts := s.client.Options()
	opts.DB = db
	newClient := redis.NewClient(opts)

	if err := newClient.Ping(s.ctx).Err(); err != nil {
		newClient.Close() // Close the new client on error to prevent resource leak
		return err
	}

	s.client.Close()
	s.client = newClient
	return nil
}

// GetDatabases returns list of databases (0-15 for standard Redis)
func (s *Service) GetDatabases() ([]map[string]interface{}, error) {
	databases := make([]map[string]interface{}, 16)
	currentDB := s.client.Options().DB

	for i := 0; i < 16; i++ {
		databases[i] = map[string]interface{}{
			"index":   i,
			"current": i == currentDB,
		}
	}

	return databases, nil
}

// FlushDB deletes all keys in the current database
func (s *Service) FlushDB() error {
	return s.client.FlushDB(s.ctx).Err()
}

func parseCommand(cmd string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, char := range cmd {
		switch {
		case char == '"' || char == '\'':
			if inQuote && char == quoteChar {
				inQuote = false
				quoteChar = 0
			} else if !inQuote {
				inQuote = true
				quoteChar = char
			} else {
				current.WriteRune(char)
			}
		case char == ' ' && !inQuote:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

func formatResult(result interface{}) interface{} {
	if result == nil {
		return nil
	}

	switch v := result.(type) {
	case []interface{}:
		formatted := make([]interface{}, len(v))
		for i, item := range v {
			formatted[i] = formatResult(item)
		}
		return formatted
	case []byte:
		return string(v)
	case map[interface{}]interface{}:
		formatted := make(map[string]interface{})
		for key, val := range v {
			formatted[fmt.Sprintf("%v", key)] = formatResult(val)
		}
		return formatted
	case map[string]interface{}:
		formatted := make(map[string]interface{})
		for key, val := range v {
			formatted[key] = formatResult(val)
		}
		return formatted
	case string:
		return v
	case int, int8, int16, int32, int64:
		return v
	case uint, uint8, uint16, uint32, uint64:
		return v
	case float32, float64:
		return v
	case bool:
		return v
	case error:
		return v.Error()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ExportedKey represents a key with its value and metadata
type ExportedKey struct {
	Key   string      `json:"key"`
	Type  string      `json:"type"`
	TTL   int64       `json:"ttl"`
	Value interface{} `json:"value"`
}

// ExportKeys exports all keys matching pattern to JSON format
func (s *Service) ExportKeys(pattern string) ([]ExportedKey, error) {
	if pattern == "" {
		pattern = "*"
	}

	var allKeys []string
	var cursor uint64
	for {
		keys, nextCursor, err := s.client.Scan(s.ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}
		allKeys = append(allKeys, keys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	exported := make([]ExportedKey, 0, len(allKeys))
	for _, key := range allKeys {
		keyInfo, err := s.GetKey(key)
		if err != nil {
			continue
		}
		exported = append(exported, ExportedKey{
			Key:   keyInfo.Key,
			Type:  keyInfo.Type,
			TTL:   keyInfo.TTL,
			Value: keyInfo.Value,
		})
	}

	return exported, nil
}

// ImportKeys imports keys from exported JSON format
func (s *Service) ImportKeys(keys []ExportedKey, overwrite bool) (int, int, error) {
	imported := 0
	skipped := 0

	for _, key := range keys {
		exists, _ := s.client.Exists(s.ctx, key.Key).Result()
		if exists > 0 && !overwrite {
			skipped++
			continue
		}

		if err := s.SetKey(key.Key, key.Value, key.Type, key.TTL); err != nil {
			skipped++
			continue
		}
		imported++
	}

	return imported, skipped, nil
}

// MemoryInfo represents memory usage for a key
type MemoryInfo struct {
	Key        string `json:"key"`
	Type       string `json:"type"`
	MemoryUsed int64  `json:"memory_used"`
	TTL        int64  `json:"ttl"`
}

// GetKeyMemory returns memory usage for a specific key
func (s *Service) GetKeyMemory(key string) (*MemoryInfo, error) {
	keyType, err := s.client.Type(s.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	// MEMORY USAGE command (Redis 4.0+)
	memoryUsed, err := s.client.MemoryUsage(s.ctx, key).Result()
	if err != nil {
		memoryUsed = 0
	}

	ttl, _ := s.client.TTL(s.ctx, key).Result()
	ttlSeconds := int64(-1)
	if ttl >= 0 {
		ttlSeconds = int64(ttl.Seconds())
	}

	return &MemoryInfo{
		Key:        key,
		Type:       keyType,
		MemoryUsed: memoryUsed,
		TTL:        ttlSeconds,
	}, nil
}

// GetMemoryStats returns memory statistics for all keys matching pattern
func (s *Service) GetMemoryStats(pattern string, limit int) ([]MemoryInfo, map[string]int64, error) {
	if pattern == "" {
		pattern = "*"
	}
	if limit <= 0 {
		limit = 100
	}

	var keys []string
	var cursor uint64
	for len(keys) < limit {
		scanned, nextCursor, err := s.client.Scan(s.ctx, cursor, pattern, int64(limit)).Result()
		if err != nil {
			return nil, nil, err
		}
		keys = append(keys, scanned...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	if len(keys) > limit {
		keys = keys[:limit]
	}

	memoryInfos := make([]MemoryInfo, 0, len(keys))
	typeStats := make(map[string]int64)

	for _, key := range keys {
		info, err := s.GetKeyMemory(key)
		if err != nil {
			continue
		}
		memoryInfos = append(memoryInfos, *info)
		typeStats[info.Type] += info.MemoryUsed
	}

	// Sort by memory usage descending
	sort.Slice(memoryInfos, func(i, j int) bool {
		return memoryInfos[i].MemoryUsed > memoryInfos[j].MemoryUsed
	})

	return memoryInfos, typeStats, nil
}

// GetMemoryOverview returns overall memory statistics
func (s *Service) GetMemoryOverview() (map[string]interface{}, error) {
	info, err := s.client.Info(s.ctx, "memory").Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				result[key] = value
			}
		}
	}

	return result, nil
}

// SlowLogEntry represents a slow log entry
type SlowLogEntry struct {
	ID            int64    `json:"id"`
	Timestamp     int64    `json:"timestamp"`
	Duration      int64    `json:"duration_us"`
	Command       []string `json:"command"`
	ClientAddr    string   `json:"client_addr"`
	ClientName    string   `json:"client_name"`
	FormattedTime string   `json:"formatted_time"`
}

// GetSlowLog returns slow log entries
func (s *Service) GetSlowLog(count int) ([]SlowLogEntry, error) {
	if count <= 0 {
		count = 50
	}

	result, err := s.client.SlowLogGet(s.ctx, int64(count)).Result()
	if err != nil {
		return nil, err
	}

	entries := make([]SlowLogEntry, 0, len(result))
	for _, log := range result {
		entry := SlowLogEntry{
			ID:            log.ID,
			Timestamp:     log.Time.Unix(),
			Duration:      log.Duration.Microseconds(),
			FormattedTime: log.Time.Format("2006-01-02 15:04:05"),
			ClientAddr:    log.ClientAddr,
			ClientName:    log.ClientName,
		}

		// Convert args to strings
		entry.Command = make([]string, len(log.Args))
		copy(entry.Command, log.Args)

		entries = append(entries, entry)
	}

	return entries, nil
}

// ResetSlowLog clears the slow log
func (s *Service) ResetSlowLog() error {
	return s.client.SlowLogReset(s.ctx).Err()
}

// GetSlowLogLen returns the number of entries in slow log
func (s *Service) GetSlowLogLen() (int64, error) {
	return s.client.Do(s.ctx, "SLOWLOG", "LEN").Int64()
}

// ScriptResult represents the result of a Lua script execution
type ScriptResult struct {
	Result   interface{} `json:"result"`
	Duration string      `json:"duration"`
	Error    string      `json:"error,omitempty"`
}

// ExecuteScript executes a Lua script
func (s *Service) ExecuteScript(script string, keys []string, args []interface{}) *ScriptResult {
	start := time.Now()
	result := &ScriptResult{}

	res, err := s.client.Eval(s.ctx, script, keys, args...).Result()
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Result = formatResult(res)
	}

	result.Duration = time.Since(start).String()
	return result
}

// LoadScript loads a script and returns its SHA
func (s *Service) LoadScript(script string) (string, error) {
	return s.client.ScriptLoad(s.ctx, script).Result()
}

// ExecuteScriptBySHA executes a script by its SHA
func (s *Service) ExecuteScriptBySHA(sha string, keys []string, args []interface{}) *ScriptResult {
	start := time.Now()
	result := &ScriptResult{}

	res, err := s.client.EvalSha(s.ctx, sha, keys, args...).Result()
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Result = formatResult(res)
	}

	result.Duration = time.Since(start).String()
	return result
}

// ScriptExists checks if scripts exist by their SHAs
func (s *Service) ScriptExists(shas []string) ([]bool, error) {
	return s.client.ScriptExists(s.ctx, shas...).Result()
}

// FlushScripts removes all loaded scripts
func (s *Service) FlushScripts() error {
	return s.client.ScriptFlush(s.ctx).Err()
}

// BulkSetTTL sets TTL for all keys matching pattern
func (s *Service) BulkSetTTL(pattern string, ttl int64) (int, error) {
	if pattern == "" {
		return 0, fmt.Errorf("pattern is required")
	}

	var allKeys []string
	var cursor uint64
	for {
		keys, nextCursor, err := s.client.Scan(s.ctx, cursor, pattern, 100).Result()
		if err != nil {
			return 0, err
		}
		allKeys = append(allKeys, keys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	updated := 0
	for _, key := range allKeys {
		var err error
		if ttl <= 0 {
			err = s.client.Persist(s.ctx, key).Err()
		} else {
			err = s.client.Expire(s.ctx, key, time.Duration(ttl)*time.Second).Err()
		}
		if err == nil {
			updated++
		}
	}

	return updated, nil
}

// GetConfig returns Redis configuration
func (s *Service) GetConfig(pattern string) (map[string]string, error) {
	if pattern == "" {
		pattern = "*"
	}

	result, err := s.client.ConfigGet(s.ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	return result, nil
}

// SetConfig sets a Redis configuration parameter
func (s *Service) SetConfig(key, value string) error {
	return s.client.ConfigSet(s.ctx, key, value).Err()
}

// RewriteConfig rewrites the configuration file
func (s *Service) RewriteConfig() error {
	return s.client.ConfigRewrite(s.ctx).Err()
}

// ResetConfigStats resets statistics
func (s *Service) ResetConfigStats() error {
	return s.client.ConfigResetStat(s.ctx).Err()
}

// ReplicationInfo represents replication status
type ReplicationInfo struct {
	Role             string                   `json:"role"`
	ConnectedSlaves  int                      `json:"connected_slaves"`
	MasterHost       string                   `json:"master_host,omitempty"`
	MasterPort       int                      `json:"master_port,omitempty"`
	MasterLinkStatus string                   `json:"master_link_status,omitempty"`
	Slaves           []map[string]interface{} `json:"slaves,omitempty"`
	RawInfo          map[string]string        `json:"raw_info"`
}

// GetReplicationInfo returns replication status
func (s *Service) GetReplicationInfo() (*ReplicationInfo, error) {
	info, err := s.client.Info(s.ctx, "replication").Result()
	if err != nil {
		return nil, err
	}

	result := &ReplicationInfo{
		RawInfo: make(map[string]string),
		Slaves:  make([]map[string]interface{}, 0),
	}

	lines := strings.Split(info, "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				result.RawInfo[key] = value

				switch key {
				case "role":
					result.Role = value
				case "connected_slaves":
					result.ConnectedSlaves, _ = strconv.Atoi(value)
				case "master_host":
					result.MasterHost = value
				case "master_port":
					result.MasterPort, _ = strconv.Atoi(value)
				case "master_link_status":
					result.MasterLinkStatus = value
				}

				// Parse slave info
				if strings.HasPrefix(key, "slave") && strings.Contains(value, "ip=") {
					slave := make(map[string]interface{})
					slave["index"] = key
					for _, pair := range strings.Split(value, ",") {
						kv := strings.SplitN(pair, "=", 2)
						if len(kv) == 2 {
							slave[kv[0]] = kv[1]
						}
					}
					result.Slaves = append(result.Slaves, slave)
				}
			}
		}
	}

	return result, nil
}

// ClusterInfo represents cluster information
type ClusterInfo struct {
	Enabled   bool                     `json:"enabled"`
	State     string                   `json:"state,omitempty"`
	SlotsOk   int                      `json:"slots_ok,omitempty"`
	SlotsPfail int                      `json:"slots_pfail,omitempty"`
	SlotsFail int                      `json:"slots_fail,omitempty"`
	KnownNodes int                     `json:"known_nodes,omitempty"`
	Size      int                      `json:"size,omitempty"`
	Nodes     []map[string]interface{} `json:"nodes,omitempty"`
	RawInfo   map[string]string        `json:"raw_info,omitempty"`
}

// GetClusterInfo returns cluster information
func (s *Service) GetClusterInfo() (*ClusterInfo, error) {
	result := &ClusterInfo{
		RawInfo: make(map[string]string),
	}

	// Check if cluster is enabled
	info, err := s.client.ClusterInfo(s.ctx).Result()
	if err != nil {
		// Cluster not enabled
		result.Enabled = false
		return result, nil
	}

	result.Enabled = true

	lines := strings.Split(info, "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				result.RawInfo[key] = value

				switch key {
				case "cluster_state":
					result.State = value
				case "cluster_slots_ok":
					result.SlotsOk, _ = strconv.Atoi(value)
				case "cluster_slots_pfail":
					result.SlotsPfail, _ = strconv.Atoi(value)
				case "cluster_slots_fail":
					result.SlotsFail, _ = strconv.Atoi(value)
				case "cluster_known_nodes":
					result.KnownNodes, _ = strconv.Atoi(value)
				case "cluster_size":
					result.Size, _ = strconv.Atoi(value)
				}
			}
		}
	}

	// Get cluster nodes
	nodes, err := s.client.ClusterNodes(s.ctx).Result()
	if err == nil {
		result.Nodes = parseClusterNodes(nodes)
	}

	return result, nil
}

func parseClusterNodes(nodesStr string) []map[string]interface{} {
	nodes := make([]map[string]interface{}, 0)
	lines := strings.Split(nodesStr, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 8 {
			node := map[string]interface{}{
				"id":      parts[0],
				"addr":    parts[1],
				"flags":   parts[2],
				"master":  parts[3],
				"ping":    parts[4],
				"pong":    parts[5],
				"epoch":   parts[6],
				"state":   parts[7],
			}
			if len(parts) > 8 {
				node["slots"] = strings.Join(parts[8:], " ")
			}
			nodes = append(nodes, node)
		}
	}

	return nodes
}

// ACLUser represents an ACL user
type ACLUser struct {
	Username string   `json:"username"`
	Flags    []string `json:"flags"`
	Keys     []string `json:"keys"`
	Channels []string `json:"channels"`
	Commands string   `json:"commands"`
	RawRules string   `json:"raw_rules"`
}

// GetACLUsers returns list of ACL users
func (s *Service) GetACLUsers() ([]string, error) {
	return s.client.ACLList(s.ctx).Result()
}

// GetACLUser returns details for a specific user
func (s *Service) GetACLUser(username string) (*ACLUser, error) {
	result, err := s.client.Do(s.ctx, "ACL", "GETUSER", username).Result()
	if err != nil {
		return nil, err
	}

	user := &ACLUser{
		Username: username,
	}

	// Parse ACL result - it comes as a slice of alternating keys and values
	if slice, ok := result.([]interface{}); ok {
		for i := 0; i < len(slice)-1; i += 2 {
			key := fmt.Sprintf("%v", slice[i])
			value := slice[i+1]

			switch key {
			case "flags":
				if flags, ok := value.([]interface{}); ok {
					user.Flags = make([]string, len(flags))
					for j, f := range flags {
						user.Flags[j] = fmt.Sprintf("%v", f)
					}
				}
			case "keys":
				if keys, ok := value.([]interface{}); ok {
					user.Keys = make([]string, len(keys))
					for j, k := range keys {
						user.Keys[j] = fmt.Sprintf("%v", k)
					}
				}
			case "channels":
				if channels, ok := value.([]interface{}); ok {
					user.Channels = make([]string, len(channels))
					for j, c := range channels {
						user.Channels[j] = fmt.Sprintf("%v", c)
					}
				}
			case "commands":
				user.Commands = fmt.Sprintf("%v", value)
			}
		}
	}

	return user, nil
}

// CreateACLUser creates a new ACL user
func (s *Service) CreateACLUser(username string, rules []string) error {
	args := make([]interface{}, 0, len(rules)+2)
	args = append(args, "ACL", "SETUSER", username)
	for _, rule := range rules {
		args = append(args, rule)
	}
	return s.client.Do(s.ctx, args...).Err()
}

// DeleteACLUser deletes an ACL user
func (s *Service) DeleteACLUser(username string) error {
	_, err := s.client.ACLDelUser(s.ctx, username).Result()
	return err
}

// GetACLLog returns ACL security log
func (s *Service) GetACLLog(count int) ([]map[string]interface{}, error) {
	if count <= 0 {
		count = 10
	}

	result, err := s.client.ACLLog(s.ctx, int64(count)).Result()
	if err != nil {
		return nil, err
	}

	logs := make([]map[string]interface{}, 0, len(result))
	for _, entry := range result {
		log := map[string]interface{}{
			"count":           entry.Count,
			"reason":          entry.Reason,
			"context":         entry.Context,
			"object":          entry.Object,
			"username":        entry.Username,
			"age_seconds":     entry.AgeSeconds,
			"client_info":     entry.ClientInfo,
			"entry_id":        entry.EntryID,
			"timestamp_created": entry.TimestampCreated,
			"timestamp_last_updated": entry.TimestampLastUpdated,
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// ResetACLLog clears the ACL log
func (s *Service) ResetACLLog() error {
	return s.client.ACLLogReset(s.ctx).Err()
}

// Publish publishes a message to a channel
func (s *Service) Publish(channel string, message interface{}) (int64, error) {
	return s.client.Publish(s.ctx, channel, message).Result()
}

// GetPubSubChannels returns list of active channels
func (s *Service) GetPubSubChannels(pattern string) ([]string, error) {
	if pattern == "" {
		pattern = "*"
	}
	return s.client.PubSubChannels(s.ctx, pattern).Result()
}

// GetPubSubNumSub returns number of subscribers per channel
func (s *Service) GetPubSubNumSub(channels []string) (map[string]int64, error) {
	return s.client.PubSubNumSub(s.ctx, channels...).Result()
}

// GetPubSubNumPat returns number of pattern subscriptions
func (s *Service) GetPubSubNumPat() (int64, error) {
	return s.client.PubSubNumPat(s.ctx).Result()
}
