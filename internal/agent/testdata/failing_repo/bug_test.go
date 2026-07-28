package failingrepo

import "testing"

func TestValue(t *testing.T) {
	if Value() != 2 {
		t.Fatalf("Value() = %d, want 2", Value())
	}
}
