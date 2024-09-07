package node

import (
	"encoding/json"
	"fmt"
	"github.com/ahmadateya/my-own-k8s/stats"
	"github.com/ahmadateya/my-own-k8s/utils"
	"io/ioutil"
	"log"
	"net/http"
)

type Node struct {
	Name            string
	Ip              string
	Api             string
	Cores           int
	Memory          uint64
	MemoryAllocated uint64
	Stats           stats.Stats
	Disk            uint64
	DiskAllocated   uint64
	Role            string
	TaskCount       uint64
}

func New(name string, api string, role string) *Node {
	return &Node{
		Name: name,
		Api:  api,
		Role: role,
	}
}

func (n *Node) GetStats() (*stats.Stats, error) {
	var resp *http.Response
	var err error

	url := fmt.Sprintf("%s/stats", n.Api)
	resp, err = utils.HTTPWithRetry(http.Get, url)
	if err != nil {
		msg := fmt.Sprintf("Unable to connect to %v. Permanent failure.\n", n.Api)
		log.Println(msg)
		return nil, fmt.Errorf(msg)
	}

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("Error retrieving stats from %v: %v", n.Api, err)
		log.Println(msg)
		return nil, fmt.Errorf(msg)
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var stats stats.Stats
	err = json.Unmarshal(body, &stats)
	if err != nil {
		msg := fmt.Sprintf("error decoding message while getting stats for node %s", n.Name)
		log.Println(msg)
		return nil, fmt.Errorf(msg)
	}

	if stats.MemStats == nil || stats.DiskStats == nil {
		return nil, fmt.Errorf("error getting stats from node %s", n.Name)
	}

	n.Memory = stats.MemTotalKb()
	n.Disk = stats.DiskTotal()
	n.Stats = stats

	return &n.Stats, nil
}
