package main

import (
	"fmt"
	"sync"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/google/uuid"
)

var wg sync.WaitGroup

func LoggerExp() {
	numOfThreads := 3
	fmt.Println("Running for loop…")

	wg.Add(numOfThreads)
	for i := 0; i < numOfThreads; i++ {
		fmt.Printf("Inside for loop... %d\n", i)
		go func(i int) {
			defer wg.Done()
			fmt.Printf("Inside thread: %d\n", i)
			LogBreadth(i)
		}(i)
	}
	wg.Wait()
	fmt.Println("Done running")
}

func LogBreadth(i int) {
	s := logger.LoggerWithRuntimeInfo(fmt.Sprintf("%d - %s", i, uuid.NewString()))

	s.Trace().Msgf("Trace: foo %s", "mank")
	s.Debug().Msgf("Debug: foo %s", "mank")
	s.Info().Msgf("Info: foo %s", "mank")
	s.Warn().Msgf("Warn: foo %s", "mank")
	s.Error().Msgf("Error: foo %s", "mank")
}
