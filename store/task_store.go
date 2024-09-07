package store

import (
	"encoding/json"
	"fmt"
	"github.com/ahmadateya/my-own-k8s/task"
	"github.com/boltdb/bolt"
	"log"
	"os"
)

type TaskStore struct {
	Db       *bolt.DB
	DbFile   string
	FileMode os.FileMode
	Bucket   string
}

type InMemoryTaskStore struct {
	Db map[string]*task.Task
}

func NewInMemoryTaskStore() *InMemoryTaskStore {
	return &InMemoryTaskStore{
		Db: make(map[string]*task.Task),
	}
}

func (i *InMemoryTaskStore) Put(key string, value interface{}) error {
	t, ok := value.(*task.Task)
	if !ok {
		return fmt.Errorf("value %v is not a task.Task type", value)
	}
	i.Db[key] = t
	return nil
}

func (i *InMemoryTaskStore) Get(key string) (interface{}, error) {
	t, ok := i.Db[key]
	if !ok {
		return nil, fmt.Errorf("task with key %s does not exist", key)
	}

	return t, nil
}

func (i *InMemoryTaskStore) List() (interface{}, error) {
	var tasks []*task.Task
	for _, t := range i.Db {
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (i *InMemoryTaskStore) Count() (int, error) {
	return len(i.Db), nil
}

func NewTaskStore(file string, mode os.FileMode, bucket string) (*TaskStore, error) {
	db, err := bolt.Open(file, mode, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to open %v", file)
	}
	t := TaskStore{
		DbFile:   file,
		FileMode: mode,
		Db:       db,
		Bucket:   bucket,
	}

	err = t.CreateBucket()
	if err != nil {
		log.Printf("bucket already exists, will use it instead of creating new one")
	}

	return &t, nil

}

func (t *TaskStore) Close() {
	t.Db.Close()
}

func (t *TaskStore) Count() (int, error) {
	taskCount := 0
	err := t.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tasks"))
		b.ForEach(func(k, v []byte) error {
			taskCount++
			return nil
		})
		return nil
	})
	if err != nil {
		return -1, err
	}

	return taskCount, nil

}

func (t *TaskStore) CreateBucket() error {
	return t.Db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte(t.Bucket))
		if err != nil {
			return fmt.Errorf("create bucket %s: %s", t.Bucket, err)
		}
		return nil
	})

}

func (t *TaskStore) Put(key string, value interface{}) error {
	return t.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(t.Bucket))

		buf, err := json.Marshal(value.(*task.Task))
		if err != nil {
			return err
		}

		err = b.Put([]byte(key), buf)
		if err != nil {
			return err
		}
		return nil
	})
}

func (t *TaskStore) Get(key string) (interface{}, error) {
	var task task.Task
	err := t.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(t.Bucket))
		t := b.Get([]byte(key))
		if t == nil {
			return fmt.Errorf("task %v not found", key)
		}
		err := json.Unmarshal(t, &task)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &task, nil

}

func (t *TaskStore) List() (interface{}, error) {
	var tasks []*task.Task
	err := t.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(t.Bucket))
		b.ForEach(func(k, v []byte) error {
			var task task.Task
			err := json.Unmarshal(v, &task)
			if err != nil {
				return err
			}
			tasks = append(tasks, &task)
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return tasks, nil
}
