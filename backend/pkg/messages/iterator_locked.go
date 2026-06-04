package messages

import "sync"

// LockedIterator wraps a MessageIterator so Iterate is serialized. Use when
// multiple queue consumers share one iterator (e.g. two Kafka topics → one handler).
type LockedIterator struct {
	mu    sync.Mutex
	inner MessageIterator
}

func NewLockedIterator(inner MessageIterator) MessageIterator {
	return &LockedIterator{inner: inner}
}

func (l *LockedIterator) Iterate(batchData []byte, batchInfo *BatchInfo) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.inner.Iterate(batchData, batchInfo)
}
