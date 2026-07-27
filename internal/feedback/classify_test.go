package feedback

import (
	"fmt"
	"testing"
)

func TestClassifierPatternCacheIsBounded(t *testing.T) {
	original := classifierPatternCache
	classifierPatternCache = newClassifierPatternCache(classifierPatternCacheLimit)
	t.Cleanup(func() {
		classifierPatternCache = original
	})

	for i := 0; i < classifierPatternCacheLimit*2; i++ {
		cachedClassifierRegexp(fmt.Sprintf("pattern-%d", i))
	}
	if got := classifierPatternCache.len(); got > classifierPatternCacheLimit {
		t.Fatalf("classifier cache size = %d, want <= %d", got, classifierPatternCacheLimit)
	}
}
