package autoinc

import (
	"sync"
	"testing"
)

func TestAutoIncConcurrency(t *testing.T) {
	var wg sync.WaitGroup
	const goroutines = 1000

	ai := &AutoInc[int]{}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ai.Get()
		}()
	}

	wg.Wait()
	if int(ai.last) != goroutines {
		t.Errorf("expected last to be %d, got %d", goroutines, ai.last)
	}
}

func TestAutoIncOverflow(t *testing.T) {
	ai := &AutoInc[uint8]{last: 254}

	if got := ai.Get(); got != 255 {
		t.Errorf("expected last to be 255, got %d", got)
	}

	if got := ai.Get(); got != 0 {
		t.Errorf("expected last to overflow to 0, got %d", got)
	}
}
