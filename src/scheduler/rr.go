package scheduler

import (
	"github.com/utkarsh5026/Orchestra/node"
	"github.com/utkarsh5026/Orchestra/task"
)

type RoundRobin struct {
	Name       string
	LastWorker int
}

func (s *RoundRobin) SelectCandidates(t task.Task, nodes []*node.Node) []*node.Node {
	return nodes
}

func (s *RoundRobin) Score(t task.Task, nodes []*node.Node) map[string]float64 {
	scores := make(map[string]float64)
	var newWorker int
	if s.LastWorker < len(nodes) {
		newWorker = s.LastWorker + 1
		s.LastWorker++
	} else {
		newWorker = 0
		s.LastWorker = 0
	}

	for idx, n := range nodes {
		if idx == newWorker {
			scores[n.Name] = 1
		} else {
			scores[n.Name] = 0
		}
	}

	return scores
}

func (s *RoundRobin) Pick(scores map[string]float64, candidates []*node.Node) *node.Node {
	var bestNode *node.Node
	var bestScore float64
	for idx, n := range candidates {
		if idx == 0 {
			bestNode = n
			bestScore = scores[n.Name]
			continue
		}

		if scores[n.Name] > bestScore {
			bestNode = n
			bestScore = scores[n.Name]
		}
	}

	return bestNode
}
