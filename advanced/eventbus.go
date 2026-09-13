package advanced

import (
	"sync"
)

type Event struct {
	Type    string
	Payload any
}

type Subscriber struct {
	ID     string
	Ch     chan Event
	Closed bool
}

type EventBus struct {
	subscribers map[string]map[string]*Subscriber
	mu          sync.RWMutex
	nextID      int
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string]map[string]*Subscriber),
	}
}

func (eb *EventBus) Subscribe(topic string, bufferSize int) *Subscriber {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.subscribers[topic] == nil {
		eb.subscribers[topic] = make(map[string]*Subscriber)
	}

	eb.nextID++
	id := string(rune(eb.nextID))

	sub := &Subscriber{
		ID:  id,
		Ch:  make(chan Event, bufferSize),
	}

	eb.subscribers[topic][id] = sub
	return sub
}

func (eb *EventBus) Unsubscribe(topic string, sub *Subscriber) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if subs, ok := eb.subscribers[topic]; ok {
		if s, ok := subs[sub.ID]; ok {
			if !s.Closed {
				close(s.Ch)
				s.Closed = true
			}
			delete(subs, sub.ID)
		}
	}
}

func (eb *EventBus) Publish(topic string, event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if subs, ok := eb.subscribers[topic]; ok {
		for _, sub := range subs {
			select {
			case sub.Ch <- event:
			default:
			}
		}
	}
}

func (eb *EventBus) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for topic, subs := range eb.subscribers {
		for _, sub := range subs {
			if !sub.Closed {
				close(sub.Ch)
				sub.Closed = true
			}
		}
		delete(eb.subscribers, topic)
	}
}
