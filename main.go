package main

import (
	s "./scaler"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	s.Scaler()
}
