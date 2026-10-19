package gamefx

import (
	"os"
	"path/filepath"
)

// csvPath resolves a CSV filename by searching the working directory and parent directories.
func csvPath(filename string) string {
	candidates := []string{
		filepath.Join("data", filename),
		filepath.Join("backend", "data", filename),
		filepath.Join("cmd", "server", "data", filename),
		filepath.Join("backend", "cmd", "server", "data", filename),
		filepath.Join("..", "data", filename),
		filepath.Join("..", "..", "data", filename),
		filepath.Join("..", "cmd", "server", "data", filename),
		filename,
		filepath.Join("..", filename),
		filepath.Join("..", "..", filename),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filename
}
