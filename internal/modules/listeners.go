package modules

import (
	"fmt"

	"github.com/keyvie/minecrafter/internal/core"
)

func AddListener[S any](container *core.Container[S], listener *core.Listener[S]) error {
	key := listener.Key()
	if _, exists := container.Listeners[key]; exists {
		return fmt.Errorf("listener for event %s with index %d already exists", key.Name, key.Index)
	}
	container.Listeners[key] = listener
	return nil
}

func AddListeners[S any](container *core.Container[S], listeners ...*core.Listener[S]) error {
	for _, listener := range listeners {
		if err := AddListener(container, listener); err != nil {
			return err
		}
	}
	return nil
}

func RemoveListener[S any](container *core.Container[S], key core.ListenerKey) error {
	if _, exists := container.Listeners[key]; !exists {
		return fmt.Errorf("listener for event %s with index %d does not exist", key.Name, key.Index)
	}
	delete(container.Listeners, key)
	return nil
}

func GetListener[S any](container *core.Container[S], key core.ListenerKey) (*core.Listener[S], error) {
	if listener, exists := container.Listeners[key]; exists {
		return listener, nil
	}
	return nil, fmt.Errorf("listener for event %s with index %d does not exist", key.Name, key.Index)
}

func ListListeners[S any](container *core.Container[S]) []*core.Listener[S] {
	listeners := make([]*core.Listener[S], 0, len(container.Listeners))
	for _, listener := range container.Listeners {
		listeners = append(listeners, listener)
	}
	return listeners
}
