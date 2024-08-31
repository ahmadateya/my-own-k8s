package main

import (
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/my-own-k8s/task"
	"github.com/my-own-k8s/worker"
	"log"
	"time"
)

func main() {
	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}
	api := worker.Api{Address: "localhost", Port: 5555, Worker: &w}

	go runTasks(&w)
	go w.CollectStats()
	api.Start()
}

func runTasks(w *worker.Worker) {
	for {
		if w.Queue.Len() != 0 {
			result := w.RunTask()
			if result.Error != nil {
				log.Printf("Error running task: %v\n", result.Error)
			}
		} else {
			log.Printf("No tasks to process currently.\n")
		}
		log.Println("Sleeping for 10 seconds.")
		time.Sleep(10 * time.Second)
	}
}
