package bus

import "sync"

type Event struct {
	Topic   string
	Payload any
}

type Bus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan Event
}

func New() *Bus {
	return &Bus{
		subscribers: make(map[string][]chan Event),
	}
}

func (b *Bus) Subscribe(topic string, buffer int) <-chan Event {
	ch := make(chan Event, buffer)

	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[topic] = append(b.subscribers[topic], ch)

	return ch
}

func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subscribers[event.Topic] {
		select {
		case ch <- event:
		default:
		}
	}
}
