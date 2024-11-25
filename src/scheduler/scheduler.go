package scheduler

import (
	"github.com/utkarsh5026/Orchestra/node"
	"github.com/utkarsh5026/Orchestra/task"
)

// Scheduler defines the interface for task scheduling algorithms.
// Implementations of this interface determine how tasks are assigned to worker nodes.
type Scheduler interface {
	// SelectCandidates filters the list of available nodes to those that are eligible
	// to run the given task. This allows implementations to exclude nodes that don't
	// meet basic requirements.
	//
	// Parameters:
	//   - t: The task to be scheduled
	//   - nodes: List of all available worker nodes
	//
	// Returns:
	//   - []*node.Node: List of candidate nodes that could potentially run the task
	SelectCandidates(t task.Task, nodes []*node.Node) []*node.Node

	// Score assigns a numerical score to each candidate node based on how suitable
	// it is for running the given task. Higher scores indicate better suitability.
	//
	// Parameters:
	//   - t: The task to be scheduled
	//   - nodes: List of candidate nodes to score
	//
	// Returns:
	//   - map[string]float64: Map of node names to their scores
	Score(t task.Task, nodes []*node.Node) map[string]float64

	// Pick selects the best node from the candidates based on their scores.
	//
	// Parameters:
	//   - scores: Map of node names to their scores from Score()
	//   - candidates: List of candidate nodes from SelectCandidates()
	//
	// Returns:
	//   - *node.Node: The selected node to run the task, or nil if no suitable node found
	Pick(scores map[string]float64, candidates []*node.Node) *node.Node
}

type SchedulerType uint

const (
	RoundRobinScheduler SchedulerType = iota
)

func NewScheduler(st SchedulerType) Scheduler {
	return &RoundRobin{}
}
