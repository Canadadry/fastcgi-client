package autoinc

import "sync"

type Integer interface {
	Signed | Unsigned
}
type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}
type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}
type AutoInc[I Integer] struct {
	last I
	mu   sync.Mutex
}

func (ai *AutoInc[I]) Get() I {
	ai.mu.Lock()
	defer ai.mu.Unlock()
	ai.last += 1
	return ai.last
}
