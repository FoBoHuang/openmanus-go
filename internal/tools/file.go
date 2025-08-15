
package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileReadTool struct{ BaseDir string }

func (t *FileReadTool) Name() string { return "file_read" }
func (t *FileReadTool) Desc() string { return "Read a file relative to BaseDir. Inputs: path (string)" }
func (t *FileReadTool) Schema() Schema { return Schema{Name: t.Name(), Desc: t.Desc(), Inputs: map[string]string{"path":"string"}} }

func secureJoin(base, p string) (string, error) {
	if p == "" { return "", errors.New("empty path") }
	clean := filepath.Clean("/" + p)
	clean = strings.TrimPrefix(clean, "/")
	full := filepath.Join(base, clean)
	if !strings.HasPrefix(filepath.Clean(full), filepath.Clean(base)) {
		return "", fmt.Errorf("path escapes base dir")
	}
	return full, nil
}

func (t *FileReadTool) Run(ctx context.Context, in Input) (Output, error) {
	rel, _ := in["path"].(string)
	full, err := secureJoin(t.BaseDir, rel); if err != nil { return nil, err }
	b, err := os.ReadFile(full); if err != nil { return nil, err }
	return Output{"path": rel, "size": len(b), "content": string(b)}, nil
}
