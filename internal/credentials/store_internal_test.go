package credentials

import (
	"bytes"
	"context"
	"sync"
	"testing"
)

func TestMemoryStoreGetCopiesWhileReadLockIsHeld(t *testing.T) {
	store := NewMemoryStore()
	ref := Ref{Provider: "openai", Host: "api.openai.com"}
	first := bytes.Repeat([]byte{'A'}, 4096)
	second := bytes.Repeat([]byte{'B'}, 4096)
	if err := store.Set(context.Background(), ref, first); err != nil {
		t.Fatal(err)
	}

	var wait sync.WaitGroup
	wait.Add(2)
	errs := make(chan string, 1)
	go func() {
		defer wait.Done()
		for i := 0; i < 2000; i++ {
			value := first
			if i%2 == 1 {
				value = second
			}
			if err := store.Set(context.Background(), ref, value); err != nil {
				select {
				case errs <- err.Error():
				default:
				}
				return
			}
		}
	}()
	go func() {
		defer wait.Done()
		for i := 0; i < 2000; i++ {
			value, err := store.Get(context.Background(), ref)
			if err != nil {
				select {
				case errs <- err.Error():
				default:
				}
				return
			}
			if !bytes.Equal(value, first) && !bytes.Equal(value, second) {
				select {
				case errs <- "read observed a cleared or partially replaced credential":
				default:
				}
				return
			}
		}
	}()
	wait.Wait()
	select {
	case err := <-errs:
		t.Fatal(err)
	default:
	}
}
