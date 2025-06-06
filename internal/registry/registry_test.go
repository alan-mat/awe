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

package registry_test

import (
	"testing"

	"github.com/alan-mat/awe/internal/registry"
)

func TestRegistryRegister(t *testing.T) {
	r := registry.New[string, any]()

	r.Register("one", 1)
	r.Register("two", "zwei")
	r.Register("three", 3.145)

	for _, k := range []string{"one", "two", "three"} {
		if exists := r.Exists(k); !exists {
			t.Errorf("key '%s' not found in registry", k)
		}
	}
}

func TestRegistryRegisterMany(t *testing.T) {
	r := registry.New[string, any]()
	entries := []registry.Entry[string, any]{
		{"test1", "a"},
		{"test2", "b"},
		{"test3", 3},
		{"test4", true},
	}
	r.RegisterMany(entries...)

	for _, e := range entries {
		if exists := r.Exists(e.Key); !exists {
			t.Errorf("key '%s' not found in registry", e.Key)
		}
	}

	r.RegisterMany(registry.Entry[string, any]{"another", new([]string)})
	if !r.Exists("another") {
		t.Errorf("key '%s' not found in registry", "another")
	}
}

func TestRegistryRegisterOverwrite(t *testing.T) {
	r := registry.New[string, any]()
	key := "test4"
	originalEntry := registry.Entry[string, any]{key, "original"}
	entries := []registry.Entry[string, any]{
		{"test1", "a"},
		{"test2", "b"},
		{"test3", 3},
		originalEntry,
	}
	r.RegisterMany(entries...)

	newEntry := registry.Entry[string, any]{key, "new"}
	r.RegisterMany(newEntry)
	val, ok := r.Get(key)
	if !ok {
		t.Error("key doesn't exist")
	}
	if val != newEntry.Value {
		t.Errorf("got '%s', expected '%s'", val, newEntry.Value)
	}
}

func TestRegistryExists(t *testing.T) {
	r := registry.New[string, any]()
	key := "new"
	r.Register(key, 200)
	if !r.Exists(key) {
		t.Errorf("key '%s' does not exist", key)
	}

	if r.Exists("some-key") {
		t.Error("unregistered key exists")
	}
}

func TestRegistryGet(t *testing.T) {
	r := registry.New[string, any]()
	key := "test-key"
	val := 3.1415
	r.Register(key, val)

	got, ok := r.Get(key)
	if !ok {
		t.Error("registered entry not found")
	}
	if got != val {
		t.Errorf("got %v, expected %v", got, val)
	}

	_, ok = r.Get("some-key")
	if ok {
		t.Error("got unregistered key")
	}
}

func TestRegistryDelete(t *testing.T) {
	r := registry.New[string, any]()
	entries := []registry.Entry[string, any]{
		{"test1", "a"},
		{"test2", "b"},
		{"test3", 3},
		{"test4", true},
		{"test5", 3.1415},
	}
	r.RegisterMany(entries...)

	r.Delete(entries[0].Key, entries[1].Key, entries[2].Key)
	for i := range 3 {
		if r.Exists(entries[i].Key) {
			t.Errorf("found deleted entry with key '%s'", entries[i].Key)
		}
	}

	r.Delete(entries[3].Key)
	if r.Exists(entries[3].Key) {
		t.Errorf("found deleted entry with key '%s'", entries[3].Key)
	}

	if !r.Exists(entries[4].Key) {
		t.Errorf("undeleted key '%s' not found", entries[4].Key)
	}
}

func TestRegistryList(t *testing.T) {
	r := registry.New[string, any]()
	entries := []registry.Entry[string, any]{
		{"test1", "a"},
		{"test2", "b"},
		{"test3", 3},
		{"test4", true},
		{"test5", 3.1415},
	}
	r.RegisterMany(entries...)

	keys := r.List()
	if len(keys) != 5 {
		t.Error("length of keys does not match")
	}

	gotMap := make(map[string]bool)
	for _, key := range keys {
		gotMap[key] = true
	}

	for _, entry := range entries {
		if _, ok := gotMap[entry.Key]; !ok {
			t.Errorf("expected key '%s' not found", entry.Key)
		}
	}

	r.Delete(keys...)
	newKeys := r.List()
	if len(newKeys) != 0 {
		t.Errorf("length of keys got '%d', expected 0", len(newKeys))
	}
}
