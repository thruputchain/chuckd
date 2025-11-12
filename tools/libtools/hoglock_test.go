package libtools

import (
	"fmt"
	"testing"
)

func TestAcquireLock(t *testing.T) {
	AcquireLock(t.Context(), "TestAcquireLock", "hog6")
	fmt.Println("acquired")
}

func TestAcquireLockMany(t *testing.T) {
	AcquireLockMany(t.Context(), "TestAcquireLock", []string{"hog6", "hog7"})
	fmt.Println("acquired")
}
