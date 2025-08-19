package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"openmanus-go/pkg/tool"
)

// RedisTool Redis 数据库工具
type RedisTool struct {
	*tool.BaseTool
	client *redis.Client
}

// NewRedisTool 创建 Redis 工具
func NewRedisTool(addr, password string, db int) *RedisTool {
	inputSchema := tool.CreateJSONSchema("object", map[string]any{
		"operation": tool.StringProperty("操作类型：get, set, del, exists, keys, hget, hset, hdel, lpush, rpop, sadd, srem, zadd, zrange"),
		"key":       tool.StringProperty("键名"),
		"value":     tool.StringProperty("值（用于 set, hset, lpush, sadd, zadd 等操作）"),
		"field":     tool.StringProperty("字段名（用于 hash 操作）"),
		"score":     tool.NumberProperty("分数（用于 sorted set 操作）"),
		"start":     tool.NumberProperty("开始位置（用于 range 操作）"),
		"stop":      tool.NumberProperty("结束位置（用于 range 操作）"),
		"pattern":   tool.StringProperty("匹配模式（用于 keys 操作）"),
		"ttl":       tool.NumberProperty("过期时间（秒）"),
	}, []string{"operation", "key"})

	outputSchema := tool.CreateJSONSchema("object", map[string]any{
		"success": tool.BooleanProperty("操作是否成功"),
		"result":  tool.StringProperty("操作结果"),
		"value":   tool.StringProperty("获取的值"),
		"values":  tool.ArrayProperty("值列表", tool.StringProperty("")),
		"count":   tool.NumberProperty("计数结果"),
		"exists":  tool.BooleanProperty("键是否存在"),
		"error":   tool.StringProperty("错误信息"),
	}, []string{"success"})

	baseTool := tool.NewBaseTool(
		"redis",
		"Redis 数据库操作工具，支持字符串、哈希、列表、集合、有序集合等数据类型",
		inputSchema,
		outputSchema,
	)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisTool{
		BaseTool: baseTool,
		client:   client,
	}
}

// Invoke 执行 Redis 操作
func (r *RedisTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return r.errorResult("operation is required"), nil
	}

	key, ok := args["key"].(string)
	if !ok {
		return r.errorResult("key is required"), nil
	}

	switch operation {
	case "get":
		return r.get(ctx, key)
	case "set":
		value, _ := args["value"].(string)
		ttl, _ := args["ttl"].(float64)
		return r.set(ctx, key, value, time.Duration(ttl)*time.Second)
	case "del":
		return r.del(ctx, key)
	case "exists":
		return r.exists(ctx, key)
	case "keys":
		pattern, _ := args["pattern"].(string)
		if pattern == "" {
			pattern = "*"
		}
		return r.keys(ctx, pattern)
	case "hget":
		field, _ := args["field"].(string)
		return r.hget(ctx, key, field)
	case "hset":
		field, _ := args["field"].(string)
		value, _ := args["value"].(string)
		return r.hset(ctx, key, field, value)
	case "hdel":
		field, _ := args["field"].(string)
		return r.hdel(ctx, key, field)
	case "lpush":
		value, _ := args["value"].(string)
		return r.lpush(ctx, key, value)
	case "rpop":
		return r.rpop(ctx, key)
	case "sadd":
		value, _ := args["value"].(string)
		return r.sadd(ctx, key, value)
	case "srem":
		value, _ := args["value"].(string)
		return r.srem(ctx, key, value)
	case "zadd":
		value, _ := args["value"].(string)
		score, _ := args["score"].(float64)
		return r.zadd(ctx, key, score, value)
	case "zrange":
		start, _ := args["start"].(float64)
		stop, _ := args["stop"].(float64)
		return r.zrange(ctx, key, int64(start), int64(stop))
	default:
		return r.errorResult(fmt.Sprintf("unsupported operation: %s", operation)), nil
	}
}

// get 获取字符串值
func (r *RedisTool) get(ctx context.Context, key string) (map[string]any, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return map[string]any{
			"success": true,
			"result":  "Key not found",
			"value":   nil,
			"exists":  false,
		}, nil
	}
	if err != nil {
		return r.errorResult(fmt.Sprintf("failed to get key: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  "Key retrieved successfully",
		"value":   val,
		"exists":  true,
	}, nil
}

// set 设置字符串值
func (r *RedisTool) set(ctx context.Context, key, value string, ttl time.Duration) (map[string]any, error) {
	err := r.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return r.errorResult(fmt.Sprintf("failed to set key: %v", err)), nil
	}

	result := fmt.Sprintf("Key '%s' set successfully", key)
	if ttl > 0 {
		result += fmt.Sprintf(" with TTL %v", ttl)
	}

	return map[string]any{
		"success": true,
		"result":  result,
	}, nil
}

// del 删除键
func (r *RedisTool) del(ctx context.Context, key string) (map[string]any, error) {
	count, err := r.client.Del(ctx, key).Result()
	if err != nil {
		return r.errorResult(fmt.Sprintf("failed to delete key: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Deleted %d key(s)", count),
		"count":   count,
	}, nil
}

// exists 检查键是否存在
func (r *RedisTool) exists(ctx context.Context, key string) (map[string]any, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return r.errorResult(fmt.Sprintf("failed to check key existence: %v", err)), nil
	}

	exists := count > 0
	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Key '%s' exists: %t", key, exists),
		"exists":  exists,
	}, nil
}

// keys 获取匹配的键列表
func (r *RedisTool) keys(ctx context.Context, pattern string) (map[string]any, error) {
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return r.errorResult(fmt.Sprintf("failed to get keys: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Found %d keys matching pattern '%s'", len(keys), pattern),
		"values":  keys,
		"count":   len(keys),
	}, nil
}

// hget 获取哈希字段值
func (r *RedisTool) hget(ctx context.Context, key, field string) (map[string]any, error) {
	val, err := r.client.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return map[string]any{
			"success": true,
			"result":  "Field not found",
			"value":   nil,
			"exists":  false,
		}, nil
	}
	if err != nil {
		return r.errorResult(fmt.Sprintf("failed to get hash field: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  "Hash field retrieved successfully",
		"value":   val,
		"exists":  true,
	}, nil
}

// hset 设置哈希字段值
func (r *RedisTool) hset(ctx context.Context, key, field, value string) (map[string]any, error) {
	count, err := r.client.HSet(ctx, key, field, value).Result()
	if err != nil {
		return r.errorResult(fmt.Sprintf("failed to set hash field: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Hash field '%s' set in key '%s'", field, key),
		"count":   count,
	}, nil
}

// hdel 删除哈希字段
func (r *RedisTool) hdel(ctx context.Context, key, field string) (map[string]any, error) {
	count, err := r.client.HDel(ctx, key, field).Result()
	if err != nil {
		return r.errorResult(fmt.Sprintf("failed to delete hash field: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Deleted %d hash field(s)", count),
		"count":   count,
	}, nil
}

// lpush 向列表左侧推入元素
func (r *RedisTool) lpush(ctx context.Context, key, value string) (map[string]any, error) {
	count, err := r.client.LPush(ctx, key, value).Result()
	if err != nil {
		return r.errorResult(fmt.Sprintf("failed to push to list: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Pushed to list, new length: %d", count),
		"count":   count,
	}, nil
}

// rpop 从列表右侧弹出元素
func (r *RedisTool) rpop(ctx context.Context, key string) (map[string]any, error) {
	val, err := r.client.RPop(ctx, key).Result()
	if err == redis.Nil {
		return map[string]any{
			"success": true,
			"result":  "List is empty",
			"value":   nil,
		}, nil
	}
	if err != nil {
		return r.errorResult(fmt.Sprintf("failed to pop from list: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  "Popped from list successfully",
		"value":   val,
	}, nil
}

// sadd 向集合添加成员
func (r *RedisTool) sadd(ctx context.Context, key, value string) (map[string]any, error) {
	count, err := r.client.SAdd(ctx, key, value).Result()
	if err != nil {
		return r.errorResult(fmt.Sprintf("failed to add to set: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Added %d member(s) to set", count),
		"count":   count,
	}, nil
}

// srem 从集合删除成员
func (r *RedisTool) srem(ctx context.Context, key, value string) (map[string]any, error) {
	count, err := r.client.SRem(ctx, key, value).Result()
	if err != nil {
		return r.errorResult(fmt.Sprintf("failed to remove from set: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Removed %d member(s) from set", count),
		"count":   count,
	}, nil
}

// zadd 向有序集合添加成员
func (r *RedisTool) zadd(ctx context.Context, key string, score float64, value string) (map[string]any, error) {
	count, err := r.client.ZAdd(ctx, key, &redis.Z{
		Score:  score,
		Member: value,
	}).Result()
	if err != nil {
		return r.errorResult(fmt.Sprintf("failed to add to sorted set: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Added %d member(s) to sorted set", count),
		"count":   count,
	}, nil
}

// zrange 获取有序集合范围内的成员
func (r *RedisTool) zrange(ctx context.Context, key string, start, stop int64) (map[string]any, error) {
	members, err := r.client.ZRange(ctx, key, start, stop).Result()
	if err != nil {
		return r.errorResult(fmt.Sprintf("failed to get sorted set range: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Retrieved %d member(s) from sorted set", len(members)),
		"values":  members,
		"count":   len(members),
	}, nil
}

// errorResult 创建错误结果
func (r *RedisTool) errorResult(message string) map[string]any {
	return map[string]any{
		"success": false,
		"error":   message,
	}
}

// Close 关闭 Redis 连接
func (r *RedisTool) Close() error {
	return r.client.Close()
}

// Ping 测试 Redis 连接
func (r *RedisTool) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
