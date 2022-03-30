package command

import (
	"fmt"
	"sync"
)

type Stats struct {
	all  int
	mAll *sync.Mutex

	done  int
	mDone *sync.Mutex

	err      int
	mErr     *sync.Mutex
	errStats map[string]error
}

func NewStats() *Stats {
	return &Stats{
		0,
		&sync.Mutex{},

		0,
		&sync.Mutex{},

		0,
		&sync.Mutex{},

		make(map[string]error),
	}
}

func (s *Stats) Add() {
	s.mAll.Lock()
	s.all += 1
	s.mAll.Unlock()
}

func (s *Stats) Done() {
	s.mDone.Lock()
	s.done += 1
	s.mDone.Unlock()
	s.PrintProgress()
}

func (s *Stats) Err(id string, err error) {
	s.mErr.Lock()
	s.err += 1
	s.errStats[id] = err
	s.mErr.Unlock()
	s.PrintProgress()
}

func (s *Stats) PrintProgress() {
	fmt.Printf(">>> Err: %d, done: %d, all: %d\n", s.err, s.done, s.all)
}

func (s *Stats) Print() {
	if len(s.errStats) == 0 {
		fmt.Printf("Finished without errors.")
		return
	}

	for key, val := range s.errStats {
		fmt.Printf("%s - %e\n", key, val)
	}
}
