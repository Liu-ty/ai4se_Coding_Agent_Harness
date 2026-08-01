//go:build integration

package demo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContainerDeliveryFilesExist(t *testing.T) {
	for _, path := range []string{"Dockerfile", ".dockerignore", filepath.Join("deploy", "compose.yml"), filepath.Join("deploy", "Caddyfile")} {
		if _, err := os.Stat(filepath.Join("..", "..", path)); err != nil {
			t.Fatalf("required delivery file %s: %v", path, err)
		}
	}
}
