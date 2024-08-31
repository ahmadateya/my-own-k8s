package task

type State int

const (
	Pending   State = iota // The initial state, the starting point, for every task.
	Scheduled              // Once the manager has scheduled it onto a worker.
	Running                // When a worker successfully starts the task (i.e., starts the container).
	Completed              // When a task completes its work in a normal way (i.e., it does not fail).
	Failed                 // If a task fails, it moves to this state.
)

var stateTransitionMap = map[State][]State{
	Pending:   []State{Scheduled},
	Scheduled: []State{Scheduled, Running, Failed},
	Running:   []State{Running, Completed, Failed},
	Completed: []State{},
	Failed:    []State{},
}

func Contains(states []State, state State) bool {
	for _, s := range states {
		if s == state {
			return true
		}
	}
	return false
}

func ValidStateTransition(from State, to State) bool {
	return Contains(stateTransitionMap[from], to)
}
