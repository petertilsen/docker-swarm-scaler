package main

import (
	s "github.com/petertilsen/docker-swarm-scaler"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	s.Scaler()
}
