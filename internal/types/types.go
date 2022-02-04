package types

type Job struct {
	Id       string
	Cids     []string
	Commands []string
	Cpu      int
	Memory   int
	Disk     int
}
