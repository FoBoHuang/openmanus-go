package main

import (
	"context"
	"fmt"
	"log"

	"openmanus-go/pkg/config"
	"openmanus-go/pkg/tool/builtin"
)

func main() {
	// 加载配置
	cfg, err := config.Load("../../configs/config.toml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 为了演示，我们手动设置 Elasticsearch 配置
	cfg.Tools.Database.Elasticsearch.Addresses = []string{"http://localhost:9200"}
	cfg.Tools.Database.Elasticsearch.Username = ""
	cfg.Tools.Database.Elasticsearch.Password = ""

	// 创建 Elasticsearch 工具
	esTool, err := builtin.NewElasticsearchTool(
		cfg.Tools.Database.Elasticsearch.Addresses,
		cfg.Tools.Database.Elasticsearch.Username,
		cfg.Tools.Database.Elasticsearch.Password,
	)
	if err != nil {
		log.Fatalf("Failed to create Elasticsearch tool: %v", err)
	}

	ctx := context.Background()

	fmt.Println("=== Elasticsearch Tool Demo ===")

	// 1. 创建索引
	fmt.Println("\n1. Creating index 'demo_index'...")
	result, err := esTool.Invoke(ctx, map[string]any{
		"operation": "create_index",
		"index":     "demo_index",
		"mapping": map[string]any{
			"properties": map[string]any{
				"title": map[string]any{
					"type": "text",
				},
				"content": map[string]any{
					"type": "text",
				},
				"tags": map[string]any{
					"type": "keyword",
				},
				"created_at": map[string]any{
					"type": "date",
				},
			},
		},
		"settings": map[string]any{
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
	})
	if err != nil {
		log.Printf("Error creating index: %v", err)
	} else {
		fmt.Printf("Result: %+v\n", result)
	}

	// 2. 索引文档
	fmt.Println("\n2. Indexing documents...")
	documents := []map[string]any{
		{
			"title":      "Getting Started with Elasticsearch",
			"content":    "Elasticsearch is a distributed search and analytics engine built on Apache Lucene.",
			"tags":       []string{"elasticsearch", "search", "tutorial"},
			"created_at": "2024-01-15T10:00:00Z",
		},
		{
			"title":      "Advanced Search Queries",
			"content":    "Learn how to build complex search queries using Elasticsearch Query DSL.",
			"tags":       []string{"elasticsearch", "query", "advanced"},
			"created_at": "2024-01-16T14:30:00Z",
		},
		{
			"title":      "Elasticsearch Performance Tuning",
			"content":    "Tips and tricks for optimizing Elasticsearch performance in production.",
			"tags":       []string{"elasticsearch", "performance", "optimization"},
			"created_at": "2024-01-17T09:15:00Z",
		},
	}

	for i, doc := range documents {
		result, err := esTool.Invoke(ctx, map[string]any{
			"operation": "index",
			"index":     "demo_index",
			"doc_id":    fmt.Sprintf("doc_%d", i+1),
			"document":  doc,
			"refresh":   "true",
		})
		if err != nil {
			log.Printf("Error indexing document %d: %v", i+1, err)
		} else {
			fmt.Printf("Indexed document %d: %+v\n", i+1, result)
		}
	}

	// 3. 搜索文档
	fmt.Println("\n3. Searching documents...")

	// 简单的匹配查询
	result, err = esTool.Invoke(ctx, map[string]any{
		"operation": "search",
		"index":     "demo_index",
		"query": map[string]any{
			"match": map[string]any{
				"content": "search",
			},
		},
		"size": 10,
	})
	if err != nil {
		log.Printf("Error searching: %v", err)
	} else {
		fmt.Printf("Search results: %+v\n", result)
	}

	// 4. 使用过滤器的复杂查询
	fmt.Println("\n4. Complex query with filters...")
	result, err = esTool.Invoke(ctx, map[string]any{
		"operation": "search",
		"index":     "demo_index",
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"match": map[string]any{
							"content": "elasticsearch",
						},
					},
				},
				"filter": []map[string]any{
					{
						"terms": map[string]any{
							"tags": []string{"tutorial", "advanced"},
						},
					},
				},
			},
		},
		"sort": []map[string]any{
			{
				"created_at": map[string]any{
					"order": "desc",
				},
			},
		},
		"size": 5,
	})
	if err != nil {
		log.Printf("Error in complex search: %v", err)
	} else {
		fmt.Printf("Complex search results: %+v\n", result)
	}

	// 5. 获取特定文档
	fmt.Println("\n5. Getting specific document...")
	result, err = esTool.Invoke(ctx, map[string]any{
		"operation": "get",
		"index":     "demo_index",
		"doc_id":    "doc_1",
	})
	if err != nil {
		log.Printf("Error getting document: %v", err)
	} else {
		fmt.Printf("Document: %+v\n", result)
	}

	// 6. 更新文档
	fmt.Println("\n6. Updating document...")
	result, err = esTool.Invoke(ctx, map[string]any{
		"operation": "update",
		"index":     "demo_index",
		"doc_id":    "doc_1",
		"document": map[string]any{
			"title":   "Getting Started with Elasticsearch - Updated",
			"updated": true,
		},
		"refresh": "true",
	})
	if err != nil {
		log.Printf("Error updating document: %v", err)
	} else {
		fmt.Printf("Update result: %+v\n", result)
	}

	// 7. 批量操作
	fmt.Println("\n7. Bulk operations...")
	bulkDocs := []map[string]any{
		{
			"title":      "Bulk Document 1",
			"content":    "This is the first bulk document",
			"tags":       []string{"bulk", "demo"},
			"created_at": "2024-01-18T10:00:00Z",
		},
		{
			"title":      "Bulk Document 2",
			"content":    "This is the second bulk document",
			"tags":       []string{"bulk", "demo"},
			"created_at": "2024-01-18T11:00:00Z",
		},
	}

	result, err = esTool.Invoke(ctx, map[string]any{
		"operation": "bulk",
		"index":     "demo_index",
		"documents": bulkDocs,
	})
	if err != nil {
		log.Printf("Error in bulk operation: %v", err)
	} else {
		fmt.Printf("Bulk operation result: %+v\n", result)
	}

	// 8. 删除文档
	fmt.Println("\n8. Deleting document...")
	result, err = esTool.Invoke(ctx, map[string]any{
		"operation": "delete",
		"index":     "demo_index",
		"doc_id":    "doc_3",
		"refresh":   "true",
	})
	if err != nil {
		log.Printf("Error deleting document: %v", err)
	} else {
		fmt.Printf("Delete result: %+v\n", result)
	}

	// 9. 最终搜索以查看剩余文档
	fmt.Println("\n9. Final search to see remaining documents...")
	result, err = esTool.Invoke(ctx, map[string]any{
		"operation": "search",
		"index":     "demo_index",
		"query": map[string]any{
			"match_all": map[string]any{},
		},
		"size": 20,
	})
	if err != nil {
		log.Printf("Error in final search: %v", err)
	} else {
		fmt.Printf("Final search results: %+v\n", result)
	}

	// 10. 清理：删除索引
	fmt.Println("\n10. Cleaning up: deleting index...")
	result, err = esTool.Invoke(ctx, map[string]any{
		"operation": "delete_index",
		"index":     "demo_index",
	})
	if err != nil {
		log.Printf("Error deleting index: %v", err)
	} else {
		fmt.Printf("Index deletion result: %+v\n", result)
	}

	fmt.Println("\n=== Demo completed ===")
}
