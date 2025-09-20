package modules

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/Luzifer/discord-community/pkg/attributestore"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// MetaStore holds the data stored by modules and serializes it
type MetaStore struct {
	ModuleAttributes map[string]attributestore.ModuleAttributeStore `json:"module_attributes"`

	filename string
	lock     sync.RWMutex
}

// NewMetaStoreFromDisk reads the stored data from disk and returns
// a new MetaStore instance with that data
func NewMetaStoreFromDisk(filename string) (*MetaStore, error) {
	out := &MetaStore{
		ModuleAttributes: map[string]attributestore.ModuleAttributeStore{},
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

	f, err := os.Open(filename) //#nosec:G304 // Intended to open store location
	if err != nil {
		return nil, errors.Wrap(err, "opening store")
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.WithError(err).Error("closing store (read)")
		}
	}()

	return out, errors.Wrap(
		json.NewDecoder(f).Decode(out),
		"decoding store",
	)
}

// Save stores the data to disk
func (m *MetaStore) Save() error {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.save()
}

func (m *MetaStore) save() error { //revive:disable-line:confusing-naming
	f, err := os.Create(m.filename)
	if err != nil {
		return errors.Wrap(err, "creating storage file")
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.WithError(err).Error("closing store (write)")
		}
	}()

	return errors.Wrap(
		json.NewEncoder(f).Encode(m),
		"encoding storage file",
	)
}

// Set stores the given value for the given key and module ID
func (m *MetaStore) Set(moduleID, key string, value interface{}) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.ModuleAttributes[moduleID] == nil {
		m.ModuleAttributes[moduleID] = make(attributestore.ModuleAttributeStore)
	}

	m.ModuleAttributes[moduleID][key] = value

	return errors.Wrap(m.save(), "saving store")
}

// ReadWithLock returns the ModuleAttributeStore for the given module ID
// and locks the MetaStore while the returned store is used
func (m *MetaStore) ReadWithLock(moduleID string, fn func(m attributestore.ModuleAttributeStore) error) error {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.ModuleAttributes[moduleID] == nil {
		return fn(attributestore.ModuleAttributeStore{})
	}

	return fn(m.ModuleAttributes[moduleID])
}
