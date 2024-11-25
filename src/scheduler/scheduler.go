package scheduler

import (
	"github.com/utkarsh5026/Orchestra/node"
	"github.com/utkarsh5026/Orchestra/task"
)

type Scheduler interface {
	SelectCandidates(t task.Task, nodes []*node.Node) []*node.Node
	Score(t task.Task, nodes []*node.Node) map[string]float64
	Pick(scores map[string]float64, candidates []*node.Node) *node.Node
}
