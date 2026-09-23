package stats

/**
 * store.go - stats storage and getter
 *
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */

import (
	"sync"
)

/**
 * Handlers Store
 */
var Store = struct {
	sync.RWMutex
	handlers map[string]*Handler
}{handlers: make(map[string]*Handler)}

/**
 * Get stats for the server
 */
func GetStats(name string) interface{} {

	Store.RLock()
	defer Store.RUnlock()

	handler, ok := Store.handlers[name]
	if !ok {
		return nil
	}
	return handler.Snapshot()
}

/**
 * SnapshotAll returns a consistent point-in-time copy of every server's
 * statistics. Worker processes use it for periodic IPC reports.
 */
func SnapshotAll() map[string]Stats {
	Store.RLock()
	handlers := make(map[string]*Handler, len(Store.handlers))
	for name, handler := range Store.handlers {
		handlers[name] = handler
	}
	Store.RUnlock()

	result := make(map[string]Stats, len(handlers))
	for name, handler := range handlers {
		result[name] = handler.Snapshot()
	}
	return result
}
