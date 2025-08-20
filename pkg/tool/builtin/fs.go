package builtin

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"openmanus-go/pkg/tool"
)

// FileSystemTool 文件系统工具
type FileSystemTool struct {
	*tool.BaseTool
	allowedPaths []string // 允许访问的路径（安全限制）
	blockedPaths []string // 禁止访问的路径
}

// NewFileSystemTool 创建文件系统工具
func NewFileSystemTool(allowedPaths, blockedPaths []string) *FileSystemTool {
	inputSchema := tool.CreateJSONSchema("object", map[string]any{
		"operation": tool.StringProperty("操作类型：read, write, list, delete, mkdir, exists, stat"),
		"path":      tool.StringProperty("文件或目录路径"),
		"content":   tool.StringProperty("写入内容（仅用于 write 操作）"),
		"recursive": tool.BooleanProperty("是否递归操作（用于 list, mkdir 等）"),
	}, []string{"operation", "path"})

	outputSchema := tool.CreateJSONSchema("object", map[string]any{
		"success": tool.BooleanProperty("操作是否成功"),
		"result":  tool.StringProperty("操作结果或内容"),
		"error":   tool.StringProperty("错误信息"),
		"files":   tool.ArrayProperty("文件列表（用于 list 操作）", tool.StringProperty("")),
		"size":    tool.NumberProperty("文件大小（字节）"),
		"is_dir":  tool.BooleanProperty("是否为目录"),
	}, []string{"success"})

	baseTool := tool.NewBaseTool(
		"fs",
		"文件系统操作工具，支持读取、写入、列表、删除等操作",
		inputSchema,
		outputSchema,
	)

	return &FileSystemTool{
		BaseTool:     baseTool,
		allowedPaths: allowedPaths,
		blockedPaths: blockedPaths,
	}
}

// Invoke 执行文件系统操作
func (fs *FileSystemTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return fs.errorResult("operation is required"), nil
	}

	path, ok := args["path"].(string)
	if !ok {
		return fs.errorResult("path is required"), nil
	}

	// 安全检查
	if err := fs.checkPath(path); err != nil {
		return fs.errorResult(err.Error()), nil
	}

	switch strings.ToLower(operation) {
	case "read":
		return fs.readFile(path)
	case "write":
		content, _ := args["content"].(string)
		return fs.writeFile(path, content)
	case "list":
		recursive, _ := args["recursive"].(bool)
		return fs.listDir(path, recursive)
	case "delete":
		return fs.deleteFile(path)
	case "mkdir":
		recursive, _ := args["recursive"].(bool)
		return fs.makeDir(path, recursive)
	case "exists":
		return fs.fileExists(path)
	case "stat":
		return fs.fileStat(path)
	default:
		return fs.errorResult(fmt.Sprintf("unsupported operation: %s", operation)), nil
	}
}

// checkPath 检查路径是否被允许访问
func (fs *FileSystemTool) checkPath(path string) error {
	// 获取绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// 检查是否在被禁止的路径中
	for _, blocked := range fs.blockedPaths {
		if strings.HasPrefix(absPath, blocked) {
			return fmt.Errorf("access denied: path %s is blocked", path)
		}
	}

	// 如果设置了允许路径，检查是否在允许的路径中
	if len(fs.allowedPaths) > 0 {
		allowed := false
		for _, allowedPath := range fs.allowedPaths {
			// 将允许的路径也转换为绝对路径进行比较
			absAllowedPath, err := filepath.Abs(allowedPath)
			if err != nil {
				continue
			}
			if strings.HasPrefix(absPath, absAllowedPath) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("access denied: path %s is not in allowed paths", path)
		}
	}

	return nil
}

// readFile 读取文件
func (fs *FileSystemTool) readFile(path string) (map[string]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return fs.errorResult(fmt.Sprintf("failed to read file: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  string(content),
		"size":    len(content),
	}, nil
}

// writeFile 写入文件
func (fs *FileSystemTool) writeFile(path, content string) (map[string]any, error) {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fs.errorResult(fmt.Sprintf("failed to create directory: %v", err)), nil
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fs.errorResult(fmt.Sprintf("failed to write file: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path),
		"size":    len(content),
	}, nil
}

// listDir 列出目录内容
func (fs *FileSystemTool) listDir(path string, recursive bool) (map[string]any, error) {
	var files []string

	if recursive {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			relPath, _ := filepath.Rel(path, filePath)
			if relPath != "." {
				files = append(files, relPath)
			}
			return nil
		})
		if err != nil {
			return fs.errorResult(fmt.Sprintf("failed to walk directory: %v", err)), nil
		}
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return fs.errorResult(fmt.Sprintf("failed to read directory: %v", err)), nil
		}

		for _, entry := range entries {
			files = append(files, entry.Name())
		}
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Listed %d items in %s", len(files), path),
		"files":   files,
	}, nil
}

// deleteFile 删除文件或目录
func (fs *FileSystemTool) deleteFile(path string) (map[string]any, error) {
	err := os.RemoveAll(path)
	if err != nil {
		return fs.errorResult(fmt.Sprintf("failed to delete: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Successfully deleted %s", path),
	}, nil
}

// makeDir 创建目录
func (fs *FileSystemTool) makeDir(path string, recursive bool) (map[string]any, error) {
	var err error
	if recursive {
		err = os.MkdirAll(path, 0755)
	} else {
		err = os.Mkdir(path, 0755)
	}

	if err != nil {
		return fs.errorResult(fmt.Sprintf("failed to create directory: %v", err)), nil
	}

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("Successfully created directory %s", path),
	}, nil
}

// fileExists 检查文件是否存在
func (fs *FileSystemTool) fileExists(path string) (map[string]any, error) {
	_, err := os.Stat(path)
	exists := err == nil

	return map[string]any{
		"success": true,
		"result":  fmt.Sprintf("File %s exists: %t", path, exists),
		"exists":  exists,
	}, nil
}

// fileStat 获取文件信息
func (fs *FileSystemTool) fileStat(path string) (map[string]any, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fs.errorResult(fmt.Sprintf("failed to get file info: %v", err)), nil
	}

	return map[string]any{
		"success":  true,
		"result":   fmt.Sprintf("File info for %s", path),
		"size":     info.Size(),
		"is_dir":   info.IsDir(),
		"mod_time": info.ModTime().Format("2006-01-02 15:04:05"),
		"mode":     info.Mode().String(),
	}, nil
}

// errorResult 创建错误结果
func (fs *FileSystemTool) errorResult(message string) map[string]any {
	return map[string]any{
		"success": false,
		"error":   message,
	}
}

// FileCopyTool 文件复制工具
type FileCopyTool struct {
	*tool.BaseTool
}

// NewFileCopyTool 创建文件复制工具
func NewFileCopyTool() *FileCopyTool {
	inputSchema := tool.CreateJSONSchema("object", map[string]any{
		"source":      tool.StringProperty("源文件路径"),
		"destination": tool.StringProperty("目标文件路径"),
		"overwrite":   tool.BooleanProperty("是否覆盖已存在的文件"),
	}, []string{"source", "destination"})

	outputSchema := tool.CreateJSONSchema("object", map[string]any{
		"success":      tool.BooleanProperty("操作是否成功"),
		"result":       tool.StringProperty("操作结果"),
		"error":        tool.StringProperty("错误信息"),
		"bytes_copied": tool.NumberProperty("复制的字节数"),
	}, []string{"success"})

	baseTool := tool.NewBaseTool(
		"file_copy",
		"文件复制工具，支持单文件和目录复制",
		inputSchema,
		outputSchema,
	)

	return &FileCopyTool{
		BaseTool: baseTool,
	}
}

// Invoke 执行文件复制
func (fc *FileCopyTool) Invoke(ctx context.Context, args map[string]any) (map[string]any, error) {
	source, ok := args["source"].(string)
	if !ok {
		return fc.errorResult("source is required"), nil
	}

	destination, ok := args["destination"].(string)
	if !ok {
		return fc.errorResult("destination is required"), nil
	}

	overwrite, _ := args["overwrite"].(bool)

	// 检查源文件是否存在
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return fc.errorResult(fmt.Sprintf("source file does not exist: %v", err)), nil
	}

	// 检查目标文件是否存在
	if !overwrite {
		if _, err := os.Stat(destination); err == nil {
			return fc.errorResult("destination file exists and overwrite is false"), nil
		}
	}

	var bytesCopied int64

	if sourceInfo.IsDir() {
		// 复制目录
		bytesCopied, err = fc.copyDir(source, destination)
	} else {
		// 复制文件
		bytesCopied, err = fc.copyFile(source, destination)
	}

	if err != nil {
		return fc.errorResult(fmt.Sprintf("copy failed: %v", err)), nil
	}

	return map[string]any{
		"success":      true,
		"result":       fmt.Sprintf("Successfully copied %s to %s", source, destination),
		"bytes_copied": bytesCopied,
	}, nil
}

// copyFile 复制单个文件
func (fc *FileCopyTool) copyFile(source, destination string) (int64, error) {
	// 确保目标目录存在
	destDir := filepath.Dir(destination)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return 0, err
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return 0, err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destination)
	if err != nil {
		return 0, err
	}
	defer destFile.Close()

	return io.Copy(destFile, sourceFile)
}

// copyDir 复制目录
func (fc *FileCopyTool) copyDir(source, destination string) (int64, error) {
	var totalBytes int64

	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destination, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		bytes, err := fc.copyFile(path, destPath)
		if err != nil {
			return err
		}
		totalBytes += bytes

		return nil
	})

	return totalBytes, err
}

// errorResult 创建错误结果
func (fc *FileCopyTool) errorResult(message string) map[string]any {
	return map[string]any{
		"success": false,
		"error":   message,
	}
}
