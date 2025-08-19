package builtin

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"openmanus-go/pkg/tool"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
)

// CrawlerTool 网页爬虫工具
type CrawlerTool struct {
	*tool.BaseTool
	userAgent      string
	allowedDomains []string
	blockedDomains []string
}

// NewCrawlerTool 创建爬虫工具
func NewCrawlerTool(userAgent string, allowedDomains, blockedDomains []string) *CrawlerTool {
	inputSchema := tool.CreateJSONSchema("object", map[string]any{
		"operation":      tool.StringProperty("操作类型：scrape, crawl, extract_links, extract_text, extract_images"),
		"url":            tool.StringProperty("要爬取的 URL"),
		"urls":           tool.ArrayProperty("要爬取的 URL 列表", tool.StringProperty("")),
		"selector":       tool.StringProperty("CSS 选择器"),
		"attribute":      tool.StringProperty("要提取的属性名"),
		"depth":          tool.NumberProperty("爬取深度（仅用于 crawl 操作）"),
		"delay":          tool.NumberProperty("请求延迟（秒）"),
		"max_pages":      tool.NumberProperty("最大页面数"),
		"follow_links":   tool.BooleanProperty("是否跟随链接"),
		"respect_robots": tool.BooleanProperty("是否遵守 robots.txt"),
	}, []string{"operation", "url"})

	outputSchema := tool.CreateJSONSchema("object", map[string]any{
		"success":     tool.BooleanProperty("操作是否成功"),
		"result":      tool.StringProperty("操作结果"),
		"data":        tool.ArrayProperty("提取的数据", tool.ObjectProperty("数据项", nil)),
		"links":       tool.ArrayProperty("提取的链接", tool.StringProperty("")),
		"images":      tool.ArrayProperty("提取的图片", tool.StringProperty("")),
		"text":        tool.StringProperty("提取的文本"),
		"pages_count": tool.NumberProperty("爬取的页面数"),
		"error":       tool.StringProperty("错误信息"),
	}, []string{"success"})

	baseTool := tool.NewBaseTool(
		"crawler",
		"网页爬虫工具，支持单页抓取、批量爬取、内容提取等功能",
		inputSchema,
		outputSchema,
	)

	if userAgent == "" {
		userAgent = "OpenManus-Go-Crawler/1.0"
	}

	return &CrawlerTool{
		BaseTool:       baseTool,
		userAgent:      userAgent,
		allowedDomains: allowedDomains,
		blockedDomains: blockedDomains,
	}
}

// Invoke 执行爬虫操作
func (c *CrawlerTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return c.errorResult("operation is required"), nil
	}

	switch strings.ToLower(operation) {
	case "scrape":
		url, _ := args["url"].(string)
		selector, _ := args["selector"].(string)
		attribute, _ := args["attribute"].(string)
		return c.scrape(ctx, url, selector, attribute)
	case "crawl":
		startURL, _ := args["url"].(string)
		depth, _ := args["depth"].(float64)
		maxPages, _ := args["max_pages"].(float64)
		followLinks, _ := args["follow_links"].(bool)
		return c.crawl(ctx, startURL, int(depth), int(maxPages), followLinks)
	case "extract_links":
		url, _ := args["url"].(string)
		return c.extractLinks(ctx, url)
	case "extract_text":
		url, _ := args["url"].(string)
		selector, _ := args["selector"].(string)
		return c.extractText(ctx, url, selector)
	case "extract_images":
		url, _ := args["url"].(string)
		return c.extractImages(ctx, url)
	default:
		return c.errorResult(fmt.Sprintf("unsupported operation: %s", operation)), nil
	}
}

// scrape 抓取单个页面的指定内容
func (c *CrawlerTool) scrape(ctx context.Context, targetURL, selector, attribute string) (map[string]any, error) {
	if targetURL == "" {
		return c.errorResult("url is required"), nil
	}

	// 检查域名限制
	if err := c.checkDomain(targetURL); err != nil {
		return c.errorResult(err.Error()), nil
	}

	collector := c.createCollector()
	var data []map[string]any
	var scrapedText strings.Builder

	// 设置回调
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		if selector != "" {
			e.ForEach(selector, func(i int, el *colly.HTMLElement) {
				item := make(map[string]any)
				item["index"] = i
				item["text"] = strings.TrimSpace(el.Text)

				if attribute != "" {
					item["attribute"] = el.Attr(attribute)
				} else {
					item["html"] = el.Text
				}

				data = append(data, item)
			})
		} else {
			// 如果没有选择器，提取所有文本
			scrapedText.WriteString(strings.TrimSpace(e.Text))
		}
	})

	collector.OnError(func(r *colly.Response, err error) {
		// 错误将在外部处理
	})

	// 访问页面
	err := collector.Visit(targetURL)
	if err != nil {
		return c.errorResult(fmt.Sprintf("failed to visit URL: %v", err)), nil
	}

	collector.Wait()

	result := map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Successfully scraped %s", targetURL),
		"url":     targetURL,
	}

	if len(data) > 0 {
		result["data"] = data
		result["count"] = len(data)
	} else if scrapedText.Len() > 0 {
		result["text"] = scrapedText.String()
	}

	return result, nil
}

// crawl 爬取多个页面
func (c *CrawlerTool) crawl(ctx context.Context, startURL string, depth, maxPages int, followLinks bool) (map[string]any, error) {
	if startURL == "" {
		return c.errorResult("url is required"), nil
	}

	if depth <= 0 {
		depth = 1
	}
	if maxPages <= 0 {
		maxPages = 10
	}

	// 检查域名限制
	if err := c.checkDomain(startURL); err != nil {
		return c.errorResult(err.Error()), nil
	}

	collector := c.createCollector()
	collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       1 * time.Second,
	})

	var pages []map[string]any
	var allLinks []string
	visitedCount := 0

	// 页面访问回调
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		if visitedCount >= maxPages {
			return
		}

		page := map[string]any{
			"url":   e.Request.URL.String(),
			"title": e.ChildText("title"),
			"text":  strings.TrimSpace(e.Text),
		}

		// 提取链接
		var pageLinks []string
		e.ForEach("a[href]", func(i int, el *colly.HTMLElement) {
			link := el.Attr("href")
			if link != "" {
				absoluteURL := e.Request.AbsoluteURL(link)
				pageLinks = append(pageLinks, absoluteURL)
				allLinks = append(allLinks, absoluteURL)
			}
		})
		page["links"] = pageLinks

		pages = append(pages, page)
		visitedCount++

		// 如果需要跟随链接且未达到深度限制
		if followLinks && len(pageLinks) > 0 && visitedCount < maxPages {
			for _, link := range pageLinks {
				if c.checkDomain(link) == nil {
					e.Request.Visit(link)
				}
			}
		}
	})

	collector.OnError(func(r *colly.Response, err error) {
		// 记录错误但继续执行
	})

	// 开始爬取
	err := collector.Visit(startURL)
	if err != nil {
		return c.errorResult(fmt.Sprintf("failed to start crawling: %v", err)), nil
	}

	collector.Wait()

	return map[string]any{
		"success":     true,
		"result":      fmt.Sprintf("Crawled %d pages starting from %s", len(pages), startURL),
		"pages":       pages,
		"pages_count": len(pages),
		"all_links":   allLinks,
	}, nil
}

// extractLinks 提取页面中的所有链接
func (c *CrawlerTool) extractLinks(ctx context.Context, targetURL string) (map[string]any, error) {
	if targetURL == "" {
		return c.errorResult("url is required"), nil
	}

	if err := c.checkDomain(targetURL); err != nil {
		return c.errorResult(err.Error()), nil
	}

	collector := c.createCollector()
	var links []string

	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		if link != "" {
			absoluteURL := e.Request.AbsoluteURL(link)
			links = append(links, absoluteURL)
		}
	})

	err := collector.Visit(targetURL)
	if err != nil {
		return c.errorResult(fmt.Sprintf("failed to visit URL: %v", err)), nil
	}

	collector.Wait()

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Extracted %d links from %s", len(links), targetURL),
		"links":   links,
		"count":   len(links),
	}, nil
}

// extractText 提取页面文本内容
func (c *CrawlerTool) extractText(ctx context.Context, targetURL, selector string) (map[string]any, error) {
	if targetURL == "" {
		return c.errorResult("url is required"), nil
	}

	if err := c.checkDomain(targetURL); err != nil {
		return c.errorResult(err.Error()), nil
	}

	collector := c.createCollector()
	var text strings.Builder

	if selector != "" {
		collector.OnHTML(selector, func(e *colly.HTMLElement) {
			text.WriteString(strings.TrimSpace(e.Text))
			text.WriteString("\n")
		})
	} else {
		collector.OnHTML("body", func(e *colly.HTMLElement) {
			text.WriteString(strings.TrimSpace(e.Text))
		})
	}

	err := collector.Visit(targetURL)
	if err != nil {
		return c.errorResult(fmt.Sprintf("failed to visit URL: %v", err)), nil
	}

	collector.Wait()

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Extracted text from %s", targetURL),
		"text":    text.String(),
		"length":  text.Len(),
	}, nil
}

// extractImages 提取页面中的图片链接
func (c *CrawlerTool) extractImages(ctx context.Context, targetURL string) (map[string]any, error) {
	if targetURL == "" {
		return c.errorResult("url is required"), nil
	}

	if err := c.checkDomain(targetURL); err != nil {
		return c.errorResult(err.Error()), nil
	}

	collector := c.createCollector()
	var images []string

	collector.OnHTML("img[src]", func(e *colly.HTMLElement) {
		src := e.Attr("src")
		if src != "" {
			absoluteURL := e.Request.AbsoluteURL(src)
			images = append(images, absoluteURL)
		}
	})

	err := collector.Visit(targetURL)
	if err != nil {
		return c.errorResult(fmt.Sprintf("failed to visit URL: %v", err)), nil
	}

	collector.Wait()

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Extracted %d images from %s", len(images), targetURL),
		"images":  images,
		"count":   len(images),
	}, nil
}

// createCollector 创建爬虫收集器
func (c *CrawlerTool) createCollector() *colly.Collector {
	collector := colly.NewCollector(
		colly.Debugger(&debug.LogDebugger{}),
	)

	// 设置用户代理
	collector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", c.userAgent)
	})

	// 设置允许的域名
	if len(c.allowedDomains) > 0 {
		collector.AllowedDomains = c.allowedDomains
	}

	// 设置请求延迟
	collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       1 * time.Second,
	})

	return collector
}

// checkDomain 检查域名是否被允许
func (c *CrawlerTool) checkDomain(targetURL string) error {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	hostname := parsedURL.Hostname()

	// 检查被禁止的域名
	for _, blocked := range c.blockedDomains {
		if strings.Contains(hostname, blocked) {
			return fmt.Errorf("domain %s is blocked", hostname)
		}
	}

	// 如果设置了允许的域名，检查是否在列表中
	if len(c.allowedDomains) > 0 {
		allowed := false
		for _, allowedDomain := range c.allowedDomains {
			if strings.Contains(hostname, allowedDomain) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("domain %s is not in allowed domains", hostname)
		}
	}

	return nil
}

// errorResult 创建错误结果
func (c *CrawlerTool) errorResult(message string) map[string]any {
	return map[string]any{
		"success": false,
		"error":   message,
	}
}
