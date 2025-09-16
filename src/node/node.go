package node

type Node struct {
	Name      string
	Ip        string
	Role      string
	TaskCount int
	Api       string
	stats     Stats
}

func NewNode(name string, api string, role string) *Node {
	return &Node{
		Name:  name,
		Api:   api,
		Role:  role,
		stats: *GetStats(),
	}
}
