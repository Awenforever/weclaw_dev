package ilink

import (
	"sync"
	"time"
)

const defaultSendMessageMinGap = 1200 * time.Millisecond

type sendMessagePacer struct {
	mu   sync.Mutex
	gap  time.Duration
	last time.Time
}

func newSendMessagePacer(gap time.Duration) *sendMessagePacer {
	p := &sendMessagePacer{}
	p.setGap(gap)
	return p
}

func (p *sendMessagePacer) setGap(gap time.Duration) {
	if p == nil {
		return
	}
	if gap < 0 {
		gap = 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gap = gap
}

func (p *sendMessagePacer) do(send func() error) error {
	if p == nil {
		return send()
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.last.IsZero() {
		if wait := p.gap - time.Since(p.last); wait > 0 {
			time.Sleep(wait)
		}
	}
	err := send()
	p.last = time.Now()
	return err
}
