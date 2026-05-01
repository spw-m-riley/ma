package dashboard

import "sync"

type eventDispatcher struct {
	root   string
	events chan RunEvent
}

var eventDispatchers sync.Map

func dispatchEvent(root string, event RunEvent) {
	if root == "" {
		return
	}

	dispatcher := getEventDispatcher(root)
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
	}
	actual, loaded := eventDispatchers.LoadOrStore(root, dispatcher)
	if loaded {
		return actual.(*eventDispatcher)
	}

	go dispatcher.run()
	return dispatcher
}

func (d *eventDispatcher) run() {
	for event := range d.events {
		if err := publishEvent(d.root, event); err != nil {
			_ = recordDiagnostic(d.root, "event delivery failed for "+event.Kind+" event: "+err.Error())
		}
	}
}
