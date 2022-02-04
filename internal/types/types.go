package types

type Job struct {
	Id       string
	Cids     []string
	Commands []string
	Cpu      int
	Memory   int
	Disk     int
}

// a message from a peer on the network updating about a job on a node
type Update struct {
	JobId  string
	NodeId string
	State  string
	Status string
	Output string
}
