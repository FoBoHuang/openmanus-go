# Elasticsearch Tool Example

这个示例演示了如何使用 OpenManus-Go 的 Elasticsearch 工具进行搜索引擎操作。

## 功能特性

Elasticsearch 工具支持以下操作：

- **索引管理**：创建、删除索引，设置映射和配置
- **文档操作**：索引、获取、更新、删除文档
- **搜索功能**：支持复杂查询、过滤、排序、分页
- **批量操作**：高效处理大量文档
- **映射管理**：动态设置字段映射

## 前置条件

1. **安装 Elasticsearch**：
   ```bash
   # 使用 Docker 运行 Elasticsearch
   docker run -d \
     --name elasticsearch \
     -p 9200:9200 \
     -p 9300:9300 \
     -e "discovery.type=single-node" \
     -e "ES_JAVA_OPTS=-Xms512m -Xmx512m" \
     elasticsearch:8.11.0
   ```

2. **配置连接**：
   在 `configs/config.toml` 中配置 Elasticsearch 连接：
   ```toml
   [tools.database.elasticsearch]
   addresses = ["http://localhost:9200"]
   username = ""  # 如果需要认证
   password = ""  # 如果需要认证
   ```

## 运行示例

```bash
cd examples/05-elasticsearch
go run main.go
```

## 支持的操作类型

### 1. 索引管理

**创建索引**：
```go
result, err := esTool.Invoke(ctx, map[string]any{
    "operation": "create_index",
    "index":     "my_index",
    "mapping": map[string]any{
        "properties": map[string]any{
            "title": map[string]any{"type": "text"},
            "tags":  map[string]any{"type": "keyword"},
        },
    },
})
```

**删除索引**：
```go
result, err := esTool.Invoke(ctx, map[string]any{
    "operation": "delete_index",
    "index":     "my_index",
})
```

### 2. 文档操作

**索引文档**：
```go
result, err := esTool.Invoke(ctx, map[string]any{
    "operation": "index",
    "index":     "my_index",
    "doc_id":    "doc_1",  // 可选，不提供则自动生成
    "document": map[string]any{
        "title":   "My Document",
        "content": "Document content here",
    },
    "refresh": "true",  // 立即刷新使文档可搜索
})
```

**获取文档**：
```go
result, err := esTool.Invoke(ctx, map[string]any{
    "operation": "get",
    "index":     "my_index",
    "doc_id":    "doc_1",
})
```

**更新文档**：
```go
result, err := esTool.Invoke(ctx, map[string]any{
    "operation": "update",
    "index":     "my_index",
    "doc_id":    "doc_1",
    "document": map[string]any{
        "title": "Updated Title",
    },
})
```

**删除文档**：
```go
result, err := esTool.Invoke(ctx, map[string]any{
    "operation": "delete",
    "index":     "my_index",
    "doc_id":    "doc_1",
})
```

### 3. 搜索功能

**简单搜索**：
```go
result, err := esTool.Invoke(ctx, map[string]any{
    "operation": "search",
    "index":     "my_index",
    "query": map[string]any{
        "match": map[string]any{
            "content": "search term",
        },
    },
    "size": 10,
    "from": 0,
})
```

**复杂查询**：
```go
result, err := esTool.Invoke(ctx, map[string]any{
    "operation": "search",
    "index":     "my_index",
    "query": map[string]any{
        "bool": map[string]any{
            "must": []map[string]any{
                {"match": map[string]any{"title": "important"}},
            },
            "filter": []map[string]any{
                {"term": map[string]any{"status": "published"}},
            },
        },
    },
    "sort": []map[string]any{
        {"created_at": map[string]any{"order": "desc"}},
    },
})
```

### 4. 批量操作

```go
result, err := esTool.Invoke(ctx, map[string]any{
    "operation": "bulk",
    "index":     "my_index",
    "documents": []map[string]any{
        {"title": "Doc 1", "content": "Content 1"},
        {"title": "Doc 2", "content": "Content 2"},
    },
})
```

### 5. 映射管理

```go
result, err := esTool.Invoke(ctx, map[string]any{
    "operation": "mapping",
    "index":     "my_index",
    "mapping": map[string]any{
        "properties": map[string]any{
            "new_field": map[string]any{
                "type": "keyword",
            },
        },
    },
})
```

## 错误处理

工具返回的结果包含 `success` 字段指示操作是否成功：

```go
result, err := esTool.Invoke(ctx, args)
if err != nil {
    log.Printf("Tool error: %v", err)
    return
}

if success, ok := result["success"].(bool); !ok || !success {
    if errMsg, ok := result["error"].(string); ok {
        log.Printf("Operation failed: %s", errMsg)
    }
    return
}

// 操作成功，处理结果
fmt.Printf("Operation result: %+v\n", result)
```

## 性能优化建议

1. **批量操作**：对于大量文档，使用 `bulk` 操作而不是单独索引每个文档
2. **刷新策略**：除非需要立即搜索，否则不要使用 `refresh=true`
3. **分页**：使用 `size` 和 `from` 参数进行分页，避免一次返回太多结果
4. **索引映射**：提前定义好字段映射，避免动态映射的性能开销

## 故障排除

1. **连接失败**：检查 Elasticsearch 是否运行在指定地址
2. **认证错误**：确认用户名和密码配置正确
3. **索引不存在**：在搜索前确保索引已创建
4. **映射冲突**：检查字段类型是否与现有映射兼容

## 更多信息

- [Elasticsearch 官方文档](https://www.elastic.co/guide/en/elasticsearch/reference/current/index.html)
- [Elasticsearch Go 客户端](https://github.com/elastic/go-elasticsearch)
- [Query DSL 参考](https://www.elastic.co/guide/en/elasticsearch/reference/current/query-dsl.html)
