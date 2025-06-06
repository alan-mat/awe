// Copyright 2025 Alan Matykiewicz
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the
// Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package registry

import (
	"sync"
)

// Entry is a helper struct to pass key-value pairs to the variadic
// Register function.
type Entry[K comparable, V any] struct {
	Key   K
	Value V
}

// Registry is a generic, thread-safe implementation of
// the Registry pattern.
type Registry[K comparable, V any] struct {
	sync.RWMutex
	data map[K]V
}

// New creates and returns a new, initialized Registry.
func New[K comparable, V any]() *Registry[K, V] {
	return &Registry[K, V]{
		data: make(map[K]V),
	}
}

// Register adds a single key-value pair to the registry.
func (r *Registry[K, V]) Register(key K, val V) {
	r.Lock()
	defer r.Unlock()

	r.data[key] = val
}

// RegisterMany adds one or more key-value pairs to the registry.
// If a key already exists, its value will be overwritten.
func (r *Registry[K, V]) RegisterMany(entries ...Entry[K, V]) {
	r.Lock()
	defer r.Unlock()

	for _, entry := range entries {
		r.data[entry.Key] = entry.Value
	}
}

// Exists checks if a key is present in the registry.
func (r *Registry[K, V]) Exists(key K) bool {
	r.RLock()
	defer r.RUnlock()

	_, ok := r.data[key]
	return ok
}

// Get retrieves a value by its key.
// It returns the value and a boolean indicating if the key was found.
func (r *Registry[K, V]) Get(key K) (V, bool) {
	r.RLock()
	defer r.RUnlock()

	val, ok := r.data[key]
	return val, ok
}

// Delete removes one or more keys from the registry.
// It is safe to call with keys that do not exist.
func (r *Registry[K, V]) Delete(keys ...K) {
	r.Lock()
	defer r.Unlock()

	for _, key := range keys {
		delete(r.data, key)
	}
}

// List returns a slice containing all of the keys currently
// in the registry. The order of the keys is not guaranteed.
func (r *Registry[K, V]) List() []K {
	r.RLock()
	defer r.RUnlock()

	keys := make([]K, 0, len(r.data))
	for k := range r.data {
		keys = append(keys, k)
	}
	return keys
}
