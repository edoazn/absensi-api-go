package services

import (
	"sync"
	"time"
)

type Blacklist struct {
	mu      sync.Mutex
	entries map[string]time.Time
}

func NewBlacklist() *Blacklist {
	return &Blacklist{entries: make(map[string]time.Time)}
}

func (b *Blacklist) Add(jti string, until time.Time) {
	if until.Before(time.Now()) {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries[jti] = until
}

func (b *Blacklist) IsBlacklisted(jti string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	until, ok := b.entries[jti]
	if !ok {
		return false
	}
	if until.Before(time.Now()) {
		delete(b.entries, jti)
		return false
	}
	return true
}
