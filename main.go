package main

import (
	"fmt"
	"github.com/ahmadateya/my-own-k8s/manager"
	"github.com/ahmadateya/my-own-k8s/worker"
)

func main() {
	whost := "localhost"
	wport := 5555

	mhost := "localhost"
	mport := 5556

	fmt.Println("Starting k8s worker")
	w1 := worker.New("worker-1", "persistent")
	// w1 := worker.New("worker-1", "memory")
	wapi1 := worker.Api{Address: whost, Port: wport, Worker: w1}

	w2 := worker.New("worker-2", "persistent")
	// w2 := worker.New("worker-2", "memory")
	wapi2 := worker.Api{Address: whost, Port: wport + 1, Worker: w2}

	w3 := worker.New("worker-3", "persistent")
	// w3 := worker.New("worker-3", "memory")
	wapi3 := worker.Api{Address: whost, Port: wport + 2, Worker: w3}

	go w1.RunTasks()
	go w1.UpdateTasks()
	go wapi1.Start()

	go w2.RunTasks()
	go w2.UpdateTasks()
	go wapi2.Start()

	go w3.RunTasks()
	go w3.UpdateTasks()
	go wapi3.Start()

	fmt.Println("Starting k8s manager")

	workers := []manager.WorkerAddress{
		manager.WorkerAddress(fmt.Sprintf("%s:%d", whost, wport)),
		manager.WorkerAddress(fmt.Sprintf("%s:%d", whost, wport+1)),
		manager.WorkerAddress(fmt.Sprintf("%s:%d", whost, wport+2)),
	}
	// m := manager.New(workers, "roundrobin")
	// m := manager.New(workers, "epvm", "memory")
	m := manager.New(workers, "epvm", "persistent")
	mapi := manager.Api{Address: mhost, Port: mport, Manager: m}

	go m.ProcessTasks()
	go m.UpdateTasks()
	go m.DoHealthChecks()
	//go m.UpdateNodeStats()

	mapi.Start()

}
