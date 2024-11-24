package scheduler

type Scheduler interface {
	SelectCandidates()
	Score()
	Pick()
}
