package core

import "sync"

// EventQueue is an unbounded event channel for engines. Push never blocks, so the
// network library's goroutines are never held up by a slow reader; events are
// delivered in order on Events until Close.
type EventQueue struct {
	mu      sync.Mutex
	pending []Event
	closed  bool
	wake    chan struct{}
	out     chan Event
	done    chan struct{}
}

// NewEventQueue creates a queue and starts its delivery goroutine.
func NewEventQueue() *EventQueue {
	q := &EventQueue{
		wake: make(chan struct{}, 1),
		out:  make(chan Event),
		done: make(chan struct{}),
	}
	go q.run()
	return q
}

// Push enqueues an event. Events pushed after Close are dropped.
func (q *EventQueue) Push(e Event) {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return
	}
	q.pending = append(q.pending, e)
	q.mu.Unlock()
	q.signal()
}

// Events returns the delivery channel. It is closed shortly after Close.
func (q *EventQueue) Events() <-chan Event { return q.out }

// Close stops accepting events and closes the channel. Pending events that nobody
// reads are discarded. It is safe to call more than once.
func (q *EventQueue) Close() {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return
	}
	q.closed = true
	q.mu.Unlock()
	close(q.done)
}

func (q *EventQueue) signal() {
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

func (q *EventQueue) run() {
	defer close(q.out)
	for {
		q.mu.Lock()
		batch := q.pending
		q.pending = nil
		q.mu.Unlock()

		for _, e := range batch {
			select {
			case q.out <- e:
			case <-q.done:
				return
			}
		}
		if len(batch) > 0 {
			continue
		}
		select {
		case <-q.wake:
		case <-q.done:
			return
		}
	}
}
