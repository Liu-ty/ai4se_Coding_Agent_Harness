package credentials

import (
	"context"
	"fmt"
	"testing"
)

func TestServiceReleasesUnusedReferenceLocks(t *testing.T) {
	service := NewService(NewMemoryStore(), nil)
	for i := 0; i < 64; i++ {
		ref := Ref{Provider: "openai", Host: fmt.Sprintf("api-%d.example.test", i)}
		if err := service.Add(context.Background(), ref, []byte("test-key")); err != nil {
			t.Fatalf("Add(%d): %v", i, err)
		}
	}
	service.locksMu.Lock()
	defer service.locksMu.Unlock()
	if got := len(service.locks); got != 0 {
		t.Fatalf("retained %d unused reference locks", got)
	}
}
