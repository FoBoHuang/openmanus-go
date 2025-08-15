
package store

import (
	"encoding/json"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketRuns   = []byte("runs")
	bucketEvents = []byte("events")
)

type Store struct{ db *bolt.DB }

func Open(path string) (*Store, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil { return nil, err }
	err = db.Update(func(tx *bolt.Tx) error {
		if _, e := tx.CreateBucketIfNotExists(bucketRuns); e != nil { return e }
		if _, e := tx.CreateBucketIfNotExists(bucketEvents); e != nil { return e }
		return nil
	})
	if err != nil { return nil, err }
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) PutRun(id string, v any) error {
	b, _ := json.Marshal(v)
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketRuns).Put([]byte(id), b)
	})
}

func (s *Store) PutEvent(ts int64, v any) error {
	key := []byte(time.Unix(0, ts).Format(time.RFC3339Nano))
	b, _ := json.Marshal(v)
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketEvents).Put(key, b)
	})
}
