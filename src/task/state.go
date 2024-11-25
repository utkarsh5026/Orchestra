package task

import "slices"

type State uint

const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)

var stateTransitionMap = map[State][]State{
	Pending:   {Scheduled},
	Scheduled: {Scheduled, Running, Failed},
	Running:   {Running, Completed, Failed},
	Completed: {},
	Failed:    {},
}

func (s State) CanTransitionTo(next State) bool {
	transitions, ok := stateTransitionMap[s]
	if !ok {
		return false
	}
	return slices.Contains(transitions, next)
}
