package datastore

import "fmt"

type DatastoreFactory struct{}

func (f *DatastoreFactory) New(cfg *Config) (Datastore, error) {
	switch cfg.Type {
	case SQLite:
		return NewSQLiteDatastore(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported datastore type: %d", cfg.Type)
	}
}
