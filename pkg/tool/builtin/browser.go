package builtin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"openmanus-go/pkg/tool"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// BrowserTool 浏览器自动化工具
type BrowserTool struct {
	*tool.BaseTool
	browser     *rod.Browser
	currentPage *rod.Page
	headless    bool
	timeout     time.Duration
}

// NewBrowserTool 创建浏览器工具
func NewBrowserTool(headless bool, timeout time.Duration) (*BrowserTool, error) {
	inputSchema := tool.CreateJSONSchema("object", map[string]any{
		"operation": tool.StringProperty("操作类型：navigate, click, type, get_text, get_html, screenshot, wait_for_element, get_attribute, execute_js"),
		"url":       tool.StringProperty("要导航的 URL"),
		"selector":  tool.StringProperty("CSS 选择器"),
		"text":      tool.StringProperty("要输入的文本"),
		"attribute": tool.StringProperty("要获取的属性名"),
		"script":    tool.StringProperty("要执行的 JavaScript 代码"),
		"timeout":   tool.NumberProperty("操作超时时间（秒）"),
		"wait_time": tool.NumberProperty("等待时间（秒）"),
	}, []string{"operation"})

	outputSchema := tool.CreateJSONSchema("object", map[string]any{
		"success":    tool.BooleanProperty("操作是否成功"),
		"result":     tool.StringProperty("操作结果"),
		"text":       tool.StringProperty("获取的文本内容"),
		"html":       tool.StringProperty("获取的 HTML 内容"),
		"url":        tool.StringProperty("当前页面 URL"),
		"title":      tool.StringProperty("页面标题"),
		"screenshot": tool.StringProperty("截图文件路径"),
		"value":      tool.StringProperty("获取的属性值或脚本执行结果"),
		"error":      tool.StringProperty("错误信息"),
	}, []string{"success"})

	baseTool := tool.NewBaseTool(
		"browser",
		"浏览器自动化工具，支持页面导航、元素操作、内容提取等功能",
		inputSchema,
		outputSchema,
	)

	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// 启动浏览器
	var browser *rod.Browser
	if headless {
		browser = rod.New().MustConnect()
	} else {
		l := launcher.New().Headless(false)
		browser = rod.New().ControlURL(l.MustLaunch()).MustConnect()
	}

	return &BrowserTool{
		BaseTool:    baseTool,
		browser:     browser,
		currentPage: nil,
		headless:    headless,
		timeout:     timeout,
	}, nil
}

// Invoke 执行浏览器操作
func (b *BrowserTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return b.errorResult("operation is required"), nil
	}

	// 设置操作超时
	timeout := b.timeout
	if timeoutSec, ok := args["timeout"].(float64); ok && timeoutSec > 0 {
		timeout = time.Duration(timeoutSec) * time.Second
	}

	// 创建带超时的上下文
	opCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch strings.ToLower(operation) {
	case "navigate":
		url, _ := args["url"].(string)
		return b.navigate(opCtx, url)
	case "click":
		selector, _ := args["selector"].(string)
		return b.click(opCtx, selector)
	case "type":
		selector, _ := args["selector"].(string)
		text, _ := args["text"].(string)
		return b.typeText(opCtx, selector, text)
	case "get_text":
		selector, _ := args["selector"].(string)
		return b.getText(opCtx, selector)
	case "get_html":
		selector, _ := args["selector"].(string)
		return b.getHTML(opCtx, selector)
	case "screenshot":
		return b.screenshot(opCtx)
	case "wait_for_element":
		selector, _ := args["selector"].(string)
		waitTime, _ := args["wait_time"].(float64)
		return b.waitForElement(opCtx, selector, time.Duration(waitTime)*time.Second)
	case "get_attribute":
		selector, _ := args["selector"].(string)
		attribute, _ := args["attribute"].(string)
		return b.getAttribute(opCtx, selector, attribute)
	case "execute_js":
		script, _ := args["script"].(string)
		return b.executeJS(opCtx, script)
	default:
		return b.errorResult(fmt.Sprintf("unsupported operation: %s", operation)), nil
	}
}

// navigate 导航到指定 URL
func (b *BrowserTool) navigate(ctx context.Context, url string) (map[string]any, error) {
	if url == "" {
		return b.errorResult("url is required for navigate operation"), nil
	}

	// 如果没有当前页面，创建一个新的
	if b.currentPage == nil {
		b.currentPage = b.browser.MustPage()
	}

	err := b.currentPage.Navigate(url)
	if err != nil {
		return b.errorResult(fmt.Sprintf("navigation failed: %v", err)), nil
	}

	// 等待页面加载
	b.currentPage.MustWaitLoad()

	title := b.currentPage.MustInfo().Title
	currentURL := b.currentPage.MustInfo().URL

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Successfully navigated to %s", url),
		"url":     currentURL,
		"title":   title,
	}, nil
}

// click 点击元素
func (b *BrowserTool) click(ctx context.Context, selector string) (map[string]any, error) {
	if selector == "" {
		return b.errorResult("selector is required for click operation"), nil
	}

	page := b.browser.MustPage()
	defer page.Close()

	element, err := page.Element(selector)
	if err != nil {
		return b.errorResult(fmt.Sprintf("element not found: %v", err)), nil
	}

	element.MustClick()

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Successfully clicked element with selector: %s", selector),
	}, nil
}

// typeText 在元素中输入文本
func (b *BrowserTool) typeText(ctx context.Context, selector, text string) (map[string]any, error) {
	if selector == "" {
		return b.errorResult("selector is required for type operation"), nil
	}

	page := b.browser.MustPage()
	defer page.Close()

	element, err := page.Element(selector)
	if err != nil {
		return b.errorResult(fmt.Sprintf("element not found: %v", err)), nil
	}

	err = element.Input(text)
	if err != nil {
		return b.errorResult(fmt.Sprintf("input failed: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Successfully typed text into element with selector: %s", selector),
	}, nil
}

// getText 获取元素文本
func (b *BrowserTool) getText(ctx context.Context, selector string) (map[string]any, error) {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return b.errorResult(fmt.Sprintf("operation canceled: %v", ctx.Err())), nil
	default:
	}

	page := b.browser.MustPage()
	defer page.Close()

	// 等待页面加载完成
	err := page.WaitLoad()
	if err != nil {
		return b.errorResult(fmt.Sprintf("page load timeout: %v", err)), nil
	}

	// 额外等待一下确保页面完全渲染
	select {
	case <-ctx.Done():
		return b.errorResult(fmt.Sprintf("operation canceled: %v", ctx.Err())), nil
	case <-time.After(2 * time.Second):
	}

	var text string

	if selector == "" {
		// 获取整个页面的文本
		result, err := page.Eval("() => document.body.innerText")
		if err != nil {
			return b.errorResult(fmt.Sprintf("failed to get page text: %v", err)), nil
		}
		text = result.Value.Str()
	} else {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return b.errorResult(fmt.Sprintf("operation canceled: %v", ctx.Err())), nil
		default:
		}

		// 尝试等待元素出现
		element, elemErr := page.Element(selector)
		if elemErr != nil {
			// 如果元素没找到，尝试等待一下再重试
			select {
			case <-ctx.Done():
				return b.errorResult(fmt.Sprintf("operation canceled: %v", ctx.Err())), nil
			case <-time.After(3 * time.Second):
			}

			element, elemErr = page.Element(selector)
			if elemErr != nil {
				return b.errorResult(fmt.Sprintf("element not found after retry: %v", elemErr)), nil
			}
		}

		// 再次检查上下文是否已取消
		select {
		case <-ctx.Done():
			return b.errorResult(fmt.Sprintf("operation canceled: %v", ctx.Err())), nil
		default:
		}

		text, err = element.Text()
		if err != nil {
			return b.errorResult(fmt.Sprintf("failed to get text: %v", err)), nil
		}
	}

	return map[string]any{
		"success": true,
		"result":  "Text retrieved successfully",
		"text":    text,
	}, nil
}

// getHTML 获取元素 HTML
func (b *BrowserTool) getHTML(ctx context.Context, selector string) (map[string]any, error) {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return b.errorResult(fmt.Sprintf("operation canceled: %v", ctx.Err())), nil
	default:
	}

	page := b.browser.MustPage()
	defer page.Close()

	var html string
	var err error

	if selector == "" {
		// 获取整个页面的 HTML
		html, err = page.HTML()
	} else {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return b.errorResult(fmt.Sprintf("operation canceled: %v", ctx.Err())), nil
		default:
		}

		element, elemErr := page.Element(selector)
		if elemErr != nil {
			return b.errorResult(fmt.Sprintf("element not found: %v", elemErr)), nil
		}

		// 再次检查上下文是否已取消
		select {
		case <-ctx.Done():
			return b.errorResult(fmt.Sprintf("operation canceled: %v", ctx.Err())), nil
		default:
		}

		html, err = element.HTML()
	}

	if err != nil {
		return b.errorResult(fmt.Sprintf("failed to get HTML: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  "HTML retrieved successfully",
		"html":    html,
	}, nil
}

// screenshot 截图
func (b *BrowserTool) screenshot(ctx context.Context) (map[string]any, error) {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return b.errorResult(fmt.Sprintf("operation canceled: %v", ctx.Err())), nil
	default:
	}

	page := b.browser.MustPage()
	defer page.Close()

	// 生成截图文件名
	filename := fmt.Sprintf("screenshot_%d.png", time.Now().Unix())

	data, err := page.Screenshot(true, nil)
	if err != nil {
		return b.errorResult(fmt.Sprintf("screenshot failed: %v", err)), nil
	}

	// 这里可以保存到文件或返回 base64 数据
	// 为简化，这里只返回数据长度
	return map[string]any{
		"success":    true,
		"result":     "Screenshot taken successfully",
		"screenshot": filename,
		"size":       len(data),
	}, nil
}

// waitForElement 等待元素出现
func (b *BrowserTool) waitForElement(ctx context.Context, selector string, waitTime time.Duration) (map[string]any, error) {
	if selector == "" {
		return b.errorResult("selector is required for wait_for_element operation"), nil
	}

	if waitTime == 0 {
		waitTime = 10 * time.Second
	}

	page := b.browser.MustPage()
	defer page.Close()

	// 创建等待上下文
	waitCtx, cancel := context.WithTimeout(ctx, waitTime)
	defer cancel()

	element, err := page.Context(waitCtx).Element(selector)
	if err != nil {
		return b.errorResult(fmt.Sprintf("element did not appear within timeout: %v", err)), nil
	}

	// 检查元素是否可见
	visible, err := element.Visible()
	if err != nil {
		return b.errorResult(fmt.Sprintf("failed to check element visibility: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Element found with selector: %s", selector),
		"visible": visible,
	}, nil
}

// getAttribute 获取元素属性
func (b *BrowserTool) getAttribute(ctx context.Context, selector, attribute string) (map[string]any, error) {
	if selector == "" {
		return b.errorResult("selector is required for get_attribute operation"), nil
	}
	if attribute == "" {
		return b.errorResult("attribute is required for get_attribute operation"), nil
	}

	page := b.browser.MustPage()
	defer page.Close()

	element, err := page.Element(selector)
	if err != nil {
		return b.errorResult(fmt.Sprintf("element not found: %v", err)), nil
	}

	value, err := element.Attribute(attribute)
	if err != nil {
		return b.errorResult(fmt.Sprintf("failed to get attribute: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Attribute '%s' retrieved successfully", attribute),
		"value":   *value,
	}, nil
}

// executeJS 执行 JavaScript
func (b *BrowserTool) executeJS(ctx context.Context, script string) (map[string]any, error) {
	if script == "" {
		return b.errorResult("script is required for execute_js operation"), nil
	}

	page := b.browser.MustPage()
	defer page.Close()

	result, err := page.Eval(script)
	if err != nil {
		return b.errorResult(fmt.Sprintf("script execution failed: %v", err)), nil
	}

	value := result.Value.Str()

	return map[string]any{
		"success": true,
		"result":  "JavaScript executed successfully",
		"value":   value,
	}, nil
}

// errorResult 创建错误结果
func (b *BrowserTool) errorResult(message string) map[string]any {
	return map[string]any{
		"success": false,
		"error":   message,
	}
}

// Close 关闭浏览器
func (b *BrowserTool) Close() error {
	if b.browser != nil {
		return b.browser.Close()
	}
	return nil
}
