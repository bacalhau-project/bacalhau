package main

type Job struct {
	Id string
	// these are comamnds that are essentially "build image commands"
	// in our use case - it's because we haven't got volume mounts of CIDs yet
	// so need to get some data from somewhere - i.e. before the actual "job" kicks in
	// which we will monitor
	BuildCommands []string
	// these are commands that are part of the actual job that we monitor
	Commands []string
	Cpu      int
	Memory   int
	Disk     int
}
