package watcher

import (
	"errors"
	"sync"
	"time"
)

// Watcher is a cached watcher for a pair of (any, error) value.
type Watcher[T any] struct {
	mutex sync.RWMutex
	value value[T]
	last  time.Time

	fetch func() (T, error)
	age   time.Duration
	opts  WatcherOpts
}

type value[T any] struct {
	v T
	e error
	t time.Time
}

type WatcherOpts uint8

const (
	_ WatcherOpts = iota
	// WatchAllowStale allows Get to return a stale value while renewing the
	// value in the background.
	WatchAllowStale
)

// Watch creates a new Watcher that transparently fetches and caches the return
// value of renew.
func Watch[T any](age time.Duration, opts WatcherOpts, fetch func() (T, error)) *Watcher[T] {
	return &Watcher[T]{
		fetch: fetch,
		age:   age,
		opts:  opts,
	}
}

// Get gets the value, its error, and the time that it was last fetched.
func (w *Watcher[T]) Get() (T, error, time.Time) {
	v := w.get()
	return v.v, v.e, v.t
}

func (w *Watcher[T]) unfetchedValue() value[T] {
	return value[T]{
		e: errors.New("watcher value unfetched"),
		t: time.Now(),
	}
}

func (w *Watcher[T]) get() value[T] {
	if w.opts&WatchAllowStale != 0 {
		if !w.mutex.TryRLock() {
			return w.unfetchedValue()
		}
	} else {
		w.mutex.RLock()
	}

	old := w.value
	ok := w.isValid()
	w.mutex.RUnlock()

	if ok {
		return old
	}

	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.isValid() {
		return w.value
	}

	if w.opts&WatchAllowStale != 0 {
		w.last = time.Now()

		go func() {
			w.mutex.Lock()
			w.renew()
			w.mutex.Unlock()
		}()

		if old.t.IsZero() {
			return w.unfetchedValue()
		}

		return old
	} else {
		w.renew()
		return w.value
	}
}

func (w *Watcher[T]) renew() {
	v, err := w.fetch()
	w.value = value[T]{
		v: v,
		e: err,
		t: time.Now(),
	}

	w.last = w.value.t
}

func (w *Watcher[T]) isValid() bool {
	return w.last.Add(w.age).After(time.Now())
}
