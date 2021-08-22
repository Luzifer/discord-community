package main

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/pkg/errors"
)

type metaStore struct {
	ModuleAttributes map[string]moduleAttributeStore `json:"module_attributes"`

	filename string
	lock     sync.RWMutex
}

func newMetaStoreFromDisk(filename string) (*metaStore, error) {
	out := &metaStore{
		ModuleAttributes: map[string]moduleAttributeStore{},
		filename:         filename,
	}

	s, err := os.Stat(filename)
	switch {
	case err == nil:
		// This is fine

	case os.IsNotExist(err):
		// No store yet, return empty store
		return out, nil

	default:
		return nil, errors.Wrap(err, "getting file stats for store")
	}

	if s.IsDir() {
		// A directory was provided
		return nil, errors.New("store location is directory")
	}

	if s.Size() == 0 {
		// An empty file was created, we don't care and will overwrite on save
		return out, nil
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "opening store")
	}
	defer f.Close()

	return out, errors.Wrap(
		json.NewDecoder(f).Decode(out),
		"decoding store",
	)
}

func (m *metaStore) Save() error {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.save()
}

func (m *metaStore) save() error {
	f, err := os.Create(m.filename)
	if err != nil {
		return errors.Wrap(err, "creating storage file")
	}
	defer f.Close()

	return errors.Wrap(
		json.NewEncoder(f).Encode(m),
		"encoding storage file",
	)
}

func (m *metaStore) Set(moduleID, key string, value interface{}) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.ModuleAttributes[moduleID] == nil {
		m.ModuleAttributes[moduleID] = make(moduleAttributeStore)
	}

	m.ModuleAttributes[moduleID][key] = value

	return errors.Wrap(m.save(), "saving store")
}

func (m *metaStore) ReadWithLock(moduleID string, fn func(m moduleAttributeStore) error) error {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.ModuleAttributes[moduleID] == nil {
		return fn(moduleAttributeStore{})
	}

	return fn(m.ModuleAttributes[moduleID])
}
