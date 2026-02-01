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

// Helper functions
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
