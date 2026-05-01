package dashboard

import (
	"sync"
	"time"
)

type eventDispatcher struct {
	root      string
	events    chan RunEvent
	done      chan struct{}
	closeOnce sync.Once
}

var eventDispatchers sync.Map

func dispatchEvent(root string, event RunEvent) {
	if root == "" {
		return
	}

	dispatcher := getEventDispatcher(root)

	// Recover from send-on-closed-channel if FlushEvents has already shut
	// this dispatcher down. Event delivery is best-effort.
	defer func() { _ = recover() }()

	select {
	case dispatcher.events <- event:
	default:
		_ = recordDiagnostic(root, "event queue full: dropping "+event.Kind+" event")
	}
}

func getEventDispatcher(root string) *eventDispatcher {
	if existing, ok := eventDispatchers.Load(root); ok {
		return existing.(*eventDispatcher)
	}

	dispatcher := &eventDispatcher{
		root:   root,
		events: make(chan RunEvent, 64),
		done:   make(chan struct{}),
	}
	actual, loaded := eventDispatchers.LoadOrStore(root, dispatcher)
	if loaded {
		return actual.(*eventDispatcher)
	}

	go dispatcher.run()
	return dispatcher
}

func (d *eventDispatcher) run() {
	defer close(d.done)
	for event := range d.events {
		if err := publishEvent(d.root, event); err != nil {
			_ = recordDiagnostic(d.root, "event delivery failed for "+event.Kind+" event: "+err.Error())
		}
	}
}

// FlushEvents closes all dispatcher channels and waits for background
// goroutines to finish delivering queued events. Safe to call multiple
// times. A 5-second timeout prevents blocking forever if the dashboard
// is unreachable (each HTTP attempt already has a 2-second timeout).
func FlushEvents() {
	eventDispatchers.Range(func(key, value any) bool {
		d := value.(*eventDispatcher)
		d.closeOnce.Do(func() { close(d.events) })

		select {
		case <-d.done:
		case <-time.After(5 * time.Second):
		}

		eventDispatchers.Delete(key)
		return true
	})
}
