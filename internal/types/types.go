package types

type Job struct {
	Id       string
	Commands []string
	Cpu      int
	Memory   int
	Disk     int
}
