package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"openmanus-go/pkg/tool"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// ElasticsearchTool Elasticsearch 搜索引擎工具
type ElasticsearchTool struct {
	*tool.BaseTool
	client *elasticsearch.Client
}

// NewElasticsearchTool 创建 Elasticsearch 工具
func NewElasticsearchTool(addresses []string, username, password string) (*ElasticsearchTool, error) {
	inputSchema := tool.CreateJSONSchema("object", map[string]any{
		"operation": tool.StringProperty("操作类型：search, index, update, delete, create_index, delete_index, get, bulk, mapping"),
		"index":     tool.StringProperty("索引名称"),
		"doc_type":  tool.StringProperty("文档类型（可选，ES 7+ 已弃用）"),
		"doc_id":    tool.StringProperty("文档ID（用于 get, update, delete 操作）"),
		"query":     tool.ObjectProperty("查询条件（JSON格式）", nil),
		"document":  tool.ObjectProperty("文档内容（JSON格式）", nil),
		"mapping":   tool.ObjectProperty("索引映射配置（JSON格式）", nil),
		"settings":  tool.ObjectProperty("索引设置（JSON格式）", nil),
		"size":      tool.NumberProperty("返回结果数量限制（默认10）"),
		"from":      tool.NumberProperty("分页起始位置（默认0）"),
		"sort":      tool.ArrayProperty("排序规则", tool.ObjectProperty("排序字段", nil)),
		"refresh":   tool.StringProperty("刷新策略：true, false, wait_for（默认false）"),
	}, []string{"operation", "index"})

	outputSchema := tool.CreateJSONSchema("object", map[string]any{
		"success":      tool.BooleanProperty("操作是否成功"),
		"result":       tool.StringProperty("操作结果描述"),
		"hits":         tool.ArrayProperty("搜索结果", tool.ObjectProperty("文档", nil)),
		"total":        tool.NumberProperty("总结果数"),
		"took":         tool.NumberProperty("查询耗时（毫秒）"),
		"doc_id":       tool.StringProperty("文档ID"),
		"version":      tool.NumberProperty("文档版本"),
		"created":      tool.BooleanProperty("是否新创建"),
		"acknowledged": tool.BooleanProperty("索引操作是否被确认"),
		"error":        tool.StringProperty("错误信息"),
	}, []string{"success"})

	baseTool := tool.NewBaseTool(
		"elasticsearch",
		"Elasticsearch 搜索引擎工具，支持文档索引、搜索、更新、删除等操作",
		inputSchema,
		outputSchema,
	)

	// 配置 Elasticsearch 客户端
	cfg := elasticsearch.Config{
		Addresses: addresses,
	}

	// 如果提供了用户名和密码，添加基本认证
	if username != "" && password != "" {
		cfg.Username = username
		cfg.Password = password
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// 测试连接
	res, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Elasticsearch: %w", err)
	}
	res.Body.Close()

	return &ElasticsearchTool{
		BaseTool: baseTool,
		client:   client,
	}, nil
}

// Invoke 执行 Elasticsearch 操作
func (es *ElasticsearchTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return es.errorResult("operation is required"), nil
	}

	index, ok := args["index"].(string)
	if !ok {
		return es.errorResult("index is required"), nil
	}

	switch strings.ToLower(operation) {
	case "search":
		query := args["query"]
		size := es.getIntParam(args["size"], 10)
		from := es.getIntParam(args["from"], 0)
		sort := args["sort"]
		return es.search(ctx, index, query, size, from, sort)
	case "index":
		docID, _ := args["doc_id"].(string)
		document := args["document"]
		refresh, _ := args["refresh"].(string)
		return es.indexDocument(ctx, index, docID, document, refresh)
	case "update":
		docID, _ := args["doc_id"].(string)
		document := args["document"]
		refresh, _ := args["refresh"].(string)
		return es.updateDocument(ctx, index, docID, document, refresh)
	case "delete":
		docID, _ := args["doc_id"].(string)
		refresh, _ := args["refresh"].(string)
		return es.deleteDocument(ctx, index, docID, refresh)
	case "get":
		docID, _ := args["doc_id"].(string)
		return es.getDocument(ctx, index, docID)
	case "create_index":
		mapping := args["mapping"]
		settings := args["settings"]
		return es.createIndex(ctx, index, mapping, settings)
	case "delete_index":
		return es.deleteIndex(ctx, index)
	case "mapping":
		mapping := args["mapping"]
		return es.putMapping(ctx, index, mapping)
	case "bulk":
		documents := args["documents"]
		return es.bulkOperation(ctx, index, documents)
	default:
		return es.errorResult(fmt.Sprintf("unsupported operation: %s", operation)), nil
	}
}

// getIntParam 获取整数参数，带默认值
func (es *ElasticsearchTool) getIntParam(value any, defaultValue int) int {
	if value == nil {
		return defaultValue
	}
	if v, ok := value.(float64); ok {
		return int(v)
	}
	if v, ok := value.(int); ok {
		return v
	}
	return defaultValue
}

// search 执行搜索操作
func (es *ElasticsearchTool) search(ctx context.Context, index string, query any, size, from int, sort any) (map[string]any, error) {
	// 构建搜索请求体
	searchBody := map[string]any{
		"size": size,
		"from": from,
	}

	if query != nil {
		searchBody["query"] = query
	} else {
		// 如果没有指定查询，使用 match_all
		searchBody["query"] = map[string]any{
			"match_all": map[string]any{},
		}
	}

	if sort != nil {
		searchBody["sort"] = sort
	}

	// 序列化请求体
	body, err := json.Marshal(searchBody)
	if err != nil {
		return es.errorResult(fmt.Sprintf("failed to marshal search body: %v", err)), nil
	}

	// 执行搜索
	req := esapi.SearchRequest{
		Index: []string{index},
		Body:  strings.NewReader(string(body)),
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return es.errorResult(fmt.Sprintf("search failed: %v", err)), nil
	}
	defer res.Body.Close()

	if res.IsError() {
		return es.errorResult(fmt.Sprintf("search error: %s", res.String())), nil
	}

	// 解析响应
	var response map[string]any
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return es.errorResult(fmt.Sprintf("failed to decode search response: %v", err)), nil
	}

	// 提取结果
	hits := make([]map[string]any, 0)
	total := 0
	took := 0

	if hitsData, ok := response["hits"].(map[string]any); ok {
		if totalData, ok := hitsData["total"].(map[string]any); ok {
			if value, ok := totalData["value"].(float64); ok {
				total = int(value)
			}
		}
		if hitsList, ok := hitsData["hits"].([]any); ok {
			for _, hit := range hitsList {
				if hitMap, ok := hit.(map[string]any); ok {
					hits = append(hits, hitMap)
				}
			}
		}
	}

	if tookValue, ok := response["took"].(float64); ok {
		took = int(tookValue)
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Search completed, found %d documents", total),
		"hits":    hits,
		"total":   total,
		"took":    took,
	}, nil
}

// indexDocument 索引文档
func (es *ElasticsearchTool) indexDocument(ctx context.Context, index, docID string, document any, refresh string) (map[string]any, error) {
	if document == nil {
		return es.errorResult("document is required for index operation"), nil
	}

	// 序列化文档
	body, err := json.Marshal(document)
	if err != nil {
		return es.errorResult(fmt.Sprintf("failed to marshal document: %v", err)), nil
	}

	// 构建请求
	req := esapi.IndexRequest{
		Index:   index,
		Body:    strings.NewReader(string(body)),
		Refresh: refresh,
	}

	if docID != "" {
		req.DocumentID = docID
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return es.errorResult(fmt.Sprintf("index failed: %v", err)), nil
	}
	defer res.Body.Close()

	if res.IsError() {
		return es.errorResult(fmt.Sprintf("index error: %s", res.String())), nil
	}

	// 解析响应
	var response map[string]any
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return es.errorResult(fmt.Sprintf("failed to decode index response: %v", err)), nil
	}

	result := map[string]any{
		"success": true,
		"result":  "Document indexed successfully",
	}

	if id, ok := response["_id"].(string); ok {
		result["doc_id"] = id
	}
	if version, ok := response["_version"].(float64); ok {
		result["version"] = int(version)
	}
	if created, ok := response["result"].(string); ok {
		result["created"] = created == "created"
	}

	return result, nil
}

// updateDocument 更新文档
func (es *ElasticsearchTool) updateDocument(ctx context.Context, index, docID string, document any, refresh string) (map[string]any, error) {
	if docID == "" {
		return es.errorResult("doc_id is required for update operation"), nil
	}
	if document == nil {
		return es.errorResult("document is required for update operation"), nil
	}

	// 构建更新请求体
	updateBody := map[string]any{
		"doc": document,
	}

	body, err := json.Marshal(updateBody)
	if err != nil {
		return es.errorResult(fmt.Sprintf("failed to marshal update body: %v", err)), nil
	}

	req := esapi.UpdateRequest{
		Index:      index,
		DocumentID: docID,
		Body:       strings.NewReader(string(body)),
		Refresh:    refresh,
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return es.errorResult(fmt.Sprintf("update failed: %v", err)), nil
	}
	defer res.Body.Close()

	if res.IsError() {
		return es.errorResult(fmt.Sprintf("update error: %s", res.String())), nil
	}

	var response map[string]any
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return es.errorResult(fmt.Sprintf("failed to decode update response: %v", err)), nil
	}

	result := map[string]any{
		"success": true,
		"result":  "Document updated successfully",
	}

	if version, ok := response["_version"].(float64); ok {
		result["version"] = int(version)
	}

	return result, nil
}

// deleteDocument 删除文档
func (es *ElasticsearchTool) deleteDocument(ctx context.Context, index, docID string, refresh string) (map[string]any, error) {
	if docID == "" {
		return es.errorResult("doc_id is required for delete operation"), nil
	}

	req := esapi.DeleteRequest{
		Index:      index,
		DocumentID: docID,
		Refresh:    refresh,
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return es.errorResult(fmt.Sprintf("delete failed: %v", err)), nil
	}
	defer res.Body.Close()

	if res.IsError() {
		return es.errorResult(fmt.Sprintf("delete error: %s", res.String())), nil
	}

	return map[string]any{
		"success": true,
		"result":  "Document deleted successfully",
	}, nil
}

// getDocument 获取文档
func (es *ElasticsearchTool) getDocument(ctx context.Context, index, docID string) (map[string]any, error) {
	if docID == "" {
		return es.errorResult("doc_id is required for get operation"), nil
	}

	req := esapi.GetRequest{
		Index:      index,
		DocumentID: docID,
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return es.errorResult(fmt.Sprintf("get failed: %v", err)), nil
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == 404 {
			return map[string]any{
				"success": true,
				"result":  "Document not found",
				"found":   false,
			}, nil
		}
		return es.errorResult(fmt.Sprintf("get error: %s", res.String())), nil
	}

	var response map[string]any
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return es.errorResult(fmt.Sprintf("failed to decode get response: %v", err)), nil
	}

	result := map[string]any{
		"success":  true,
		"result":   "Document retrieved successfully",
		"document": response,
	}

	if found, ok := response["found"].(bool); ok {
		result["found"] = found
	}

	return result, nil
}

// createIndex 创建索引
func (es *ElasticsearchTool) createIndex(ctx context.Context, index string, mapping, settings any) (map[string]any, error) {
	requestBody := make(map[string]any)

	if mapping != nil {
		requestBody["mappings"] = mapping
	}
	if settings != nil {
		requestBody["settings"] = settings
	}

	var body strings.Reader
	if len(requestBody) > 0 {
		bodyBytes, err := json.Marshal(requestBody)
		if err != nil {
			return es.errorResult(fmt.Sprintf("failed to marshal index body: %v", err)), nil
		}
		body = *strings.NewReader(string(bodyBytes))
	}

	req := esapi.IndicesCreateRequest{
		Index: index,
		Body:  &body,
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return es.errorResult(fmt.Sprintf("create index failed: %v", err)), nil
	}
	defer res.Body.Close()

	if res.IsError() {
		return es.errorResult(fmt.Sprintf("create index error: %s", res.String())), nil
	}

	var response map[string]any
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return es.errorResult(fmt.Sprintf("failed to decode create index response: %v", err)), nil
	}

	result := map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Index '%s' created successfully", index),
	}

	if acknowledged, ok := response["acknowledged"].(bool); ok {
		result["acknowledged"] = acknowledged
	}

	return result, nil
}

// deleteIndex 删除索引
func (es *ElasticsearchTool) deleteIndex(ctx context.Context, index string) (map[string]any, error) {
	req := esapi.IndicesDeleteRequest{
		Index: []string{index},
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return es.errorResult(fmt.Sprintf("delete index failed: %v", err)), nil
	}
	defer res.Body.Close()

	if res.IsError() {
		return es.errorResult(fmt.Sprintf("delete index error: %s", res.String())), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Index '%s' deleted successfully", index),
	}, nil
}

// putMapping 设置索引映射
func (es *ElasticsearchTool) putMapping(ctx context.Context, index string, mapping any) (map[string]any, error) {
	if mapping == nil {
		return es.errorResult("mapping is required for mapping operation"), nil
	}

	body, err := json.Marshal(mapping)
	if err != nil {
		return es.errorResult(fmt.Sprintf("failed to marshal mapping: %v", err)), nil
	}

	req := esapi.IndicesPutMappingRequest{
		Index: []string{index},
		Body:  strings.NewReader(string(body)),
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return es.errorResult(fmt.Sprintf("put mapping failed: %v", err)), nil
	}
	defer res.Body.Close()

	if res.IsError() {
		return es.errorResult(fmt.Sprintf("put mapping error: %s", res.String())), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Mapping for index '%s' updated successfully", index),
	}, nil
}

// bulkOperation 批量操作
func (es *ElasticsearchTool) bulkOperation(ctx context.Context, index string, documents any) (map[string]any, error) {
	if documents == nil {
		return es.errorResult("documents is required for bulk operation"), nil
	}

	docsList, ok := documents.([]any)
	if !ok {
		return es.errorResult("documents must be an array"), nil
	}

	// 构建批量操作请求体
	var bulkBody strings.Builder
	for _, doc := range docsList {
		// 索引操作头
		indexAction := map[string]any{
			"index": map[string]any{
				"_index": index,
			},
		}
		indexHeader, _ := json.Marshal(indexAction)
		bulkBody.Write(indexHeader)
		bulkBody.WriteString("\n")

		// 文档内容
		docBody, _ := json.Marshal(doc)
		bulkBody.Write(docBody)
		bulkBody.WriteString("\n")
	}

	req := esapi.BulkRequest{
		Body: strings.NewReader(bulkBody.String()),
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return es.errorResult(fmt.Sprintf("bulk operation failed: %v", err)), nil
	}
	defer res.Body.Close()

	if res.IsError() {
		return es.errorResult(fmt.Sprintf("bulk operation error: %s", res.String())), nil
	}

	var response map[string]any
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return es.errorResult(fmt.Sprintf("failed to decode bulk response: %v", err)), nil
	}

	took := 0
	if tookValue, ok := response["took"].(float64); ok {
		took = int(tookValue)
	}

	errors := false
	if errorsValue, ok := response["errors"].(bool); ok {
		errors = errorsValue
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Bulk operation completed, processed %d documents", len(docsList)),
		"took":    took,
		"errors":  errors,
	}, nil
}

// errorResult 创建错误结果
func (es *ElasticsearchTool) errorResult(message string) map[string]any {
	return map[string]any{
		"success": false,
		"error":   message,
	}
}

// Close 关闭 Elasticsearch 连接
func (es *ElasticsearchTool) Close() error {
	// Elasticsearch Go 客户端不需要显式关闭
	return nil
}

// Ping 测试 Elasticsearch 连接
func (es *ElasticsearchTool) Ping(ctx context.Context) error {
	res, err := es.client.Info()
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch ping failed: %s", res.String())
	}

	return nil
}
