# Task Related Notes and Thoughts

## How it works
### Main Components


---
## TODOs and Spin-offs

### K8s manager
- Kubernetes doesn’t have a singular name for this component, but it is often referred to as the control plane.
- The control plane is responsible for managing the cluster and consists of multiple components:
  - API server, controller manager, etcd, scheduler.

#### Control plane vs Data plane In Networking
- this separation is coming from the SDN networking world.
- the **network control-plane**: controls how data moves from point A to point B, responsible for things like
  - Routing tables: which are determined by different protocols, such as 
    - the Border Gateway Protocol (BGP) 
    - and the Open Shortest Path First (OSPF) protocol.
  - Network management protocols (SNMP)
  - Application layer protocols (HTTP and FTP)
- while the **network data-plane** is responsible for actually moving the data from point A to point B.
  - Some examples of data planes include: Ethernet networks, Wi-Fi networks, Cellular networks

### Multiple managers and Consensus
- The current implementation is a single manager, which means the manager is a single point of failure.

#### Dist-Sys, Consensus, and Raft
- The manager should be able to elect a new manager if the current one fails.
- The manager should be able to replicate its state to other managers.
- For this we can use a consensus algorithm like Raft and apply other distributed systems concepts.
  - Check k8s's [etcd](https://etcd.io/) for a distributed key-value store used by k8s to store its state.