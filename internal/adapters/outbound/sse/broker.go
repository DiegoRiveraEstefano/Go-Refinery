// Package sse implements an in-memory Pub/Sub broker for Server-Sent Events (SSE).
package sse

import (
	"context"
	"sync"

	"github.com/diego/go-refinery/internal/domain/job"
)

// Broker fans out job.ProgressEvent updates to subscribed HTTP SSE clients.
type Broker struct {
	mu   sync.RWMutex
	subs map[job.ID]map[chan job.ProgressEvent]struct{}
}

// NewBroker initializes an in-memory SSE Broker.
func NewBroker() *Broker {
	return &Broker{
		subs: make(map[job.ID]map[chan job.ProgressEvent]struct{}),
	}
}

// Publish sends a ProgressEvent to all active subscribers of event.JobID without blocking.
func (b *Broker) Publish(event job.ProgressEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	clients, ok := b.subs[event.JobID]
	if !ok {
		return
	}
	for ch := range clients {
		select {
		case ch <- event:
		default:
		}
	}
}

// Subscribe registers a listener channel for a specific jobID and returns an unsubscribe closure.
func (b *Broker) Subscribe(ctx context.Context, jobID job.ID) (<-chan job.ProgressEvent, func()) {
	ch := make(chan job.ProgressEvent, 16)

	b.mu.Lock()
	if _, ok := b.subs[jobID]; !ok {
		b.subs[jobID] = make(map[chan job.ProgressEvent]struct{})
	}
	b.subs[jobID][ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			b.mu.Lock()
			if clients, ok := b.subs[jobID]; ok {
				delete(clients, ch)
				if len(clients) == 0 {
					delete(b.subs, jobID)
				}
			}
			b.mu.Unlock()
			close(ch)
		})
	}

	go func() {
		<-ctx.Done()
		unsubscribe()
	}()

	return ch, unsubscribe
}
