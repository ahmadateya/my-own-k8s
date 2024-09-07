package store

import (
	"encoding/json"
	"fmt"
	"github.com/ahmadateya/my-own-k8s/task"
	"github.com/boltdb/bolt"
	"log"
	"os"
)

type InMemoryTaskEventStore struct {
	Db map[string]*task.Event
}

func NewInMemoryTaskEventStore() *InMemoryTaskEventStore {
	return &InMemoryTaskEventStore{
		Db: make(map[string]*task.Event),
	}
}

func (i *InMemoryTaskEventStore) Put(key string, value interface{}) error {
	e, ok := value.(*task.Event)
	if !ok {
		return fmt.Errorf("value %v is not a task.TaskEvent type", value)
	}
	i.Db[key] = e
	return nil
}

func (i *InMemoryTaskEventStore) Get(key string) (interface{}, error) {
	e, ok := i.Db[key]
	if !ok {
		return nil, fmt.Errorf("task event with key %s does not exist", key)
	}

	return e, nil
}

func (i *InMemoryTaskEventStore) List() (interface{}, error) {
	var events []*task.Event
	for _, e := range i.Db {
		events = append(events, e)
	}
	return events, nil
}

func (i *InMemoryTaskEventStore) Count() (int, error) {
	return len(i.Db), nil
}

type EventStore struct {
	DbFile   string
	FileMode os.FileMode
	Db       *bolt.DB
	Bucket   string
}

func NewEventStore(file string, mode os.FileMode, bucket string) (*EventStore, error) {
	db, err := bolt.Open(file, mode, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to open %v", file)
	}
	e := EventStore{
		DbFile:   file,
		FileMode: mode,
		Db:       db,
		Bucket:   bucket,
	}

	err = e.CreateBucket()
	if err != nil {
		log.Printf("bucket already exists, will use it instead of creating new one")
	}

	return &e, nil
}

func (e *EventStore) Close() {
	e.Db.Close()
}

func (e *EventStore) CreateBucket() error {
	return e.Db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte(e.Bucket))
		if err != nil {
			return fmt.Errorf("create bucket %s: %s", e.Bucket, err)
		}
		return nil
	})
}

func (e *EventStore) Count() (int, error) {
	eventCount := 0
	err := e.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(e.Bucket))
		b.ForEach(func(k, v []byte) error {
			eventCount++
			return nil
		})
		return nil
	})
	if err != nil {
		return -1, err
	}

	return eventCount, nil
}

func (e *EventStore) Put(key string, value interface{}) error {
	return e.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(e.Bucket))

		buf, err := json.Marshal(value.(*task.Event))
		if err != nil {
			return err
		}

		err = b.Put([]byte(key), buf)
		if err != nil {
			log.Printf("unable to save item %s", key)
			return err
		}
		return nil
	})
}

func (e *EventStore) Get(key string) (interface{}, error) {
	var event task.Event
	err := e.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(e.Bucket))
		t := b.Get([]byte(key))
		if t == nil {
			return fmt.Errorf("event %v not found", key)
		}
		err := json.Unmarshal(t, &event)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (e *EventStore) List() (interface{}, error) {
	var events []*task.Event
	err := e.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(e.Bucket))
		b.ForEach(func(k, v []byte) error {
			var event task.Event
			err := json.Unmarshal(v, &event)
			if err != nil {
				return err
			}
			events = append(events, &event)
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return events, nil
}
