package main

import (
	"fmt"
	"github.com/ahmadateya/my-own-k8s/manager"
	"github.com/ahmadateya/my-own-k8s/task"
	"github.com/ahmadateya/my-own-k8s/worker"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

func main() {
	whost := "localhost"
	wport := 5555

	mhost := "localhost"
	mport := 5556

	fmt.Println("Starting k8s worker")

	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}
	wapi := worker.Api{Address: whost, Port: wport, Worker: &w}

	go w.RunTasks()
	go w.CollectStats()
	go w.UpdateTasks()
	go wapi.Start()

	workers := []manager.WorkerAddress{
		manager.WorkerAddress(fmt.Sprintf("%s:%d", whost, wport)),
	}
	m := manager.New(workers, "epvm")
	mapi := manager.Api{Address: mhost, Port: mport, Manager: m}

	go m.ProcessTasks()
	go m.UpdateTasks()
	go m.DoHealthChecks()

	mapi.Start()

}
