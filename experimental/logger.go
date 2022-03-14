package main

import (
	"fmt"
	"sync"

	"github.com/filecoin-project/bacalhau/internal/logger"
	"github.com/google/uuid"
)

var wg sync.WaitGroup

func Logger_Exp() {
	logger.Initialize()

	num_of_threads := 3
	fmt.Println("Running for loop…")

	wg.Add(num_of_threads)
	for i := 0; i < num_of_threads; i++ {
		fmt.Printf("Inside for loop... %d\n", i)
		go func(i int) {
			defer wg.Done()
			fmt.Printf("Inside thread: %d\n", i)
			log_breadth(i)
		}(i)
	}
	wg.Wait()
	fmt.Println("Done running")
}

func log_breadth(i int) {
	// subLogger := logger.LoggerWithNodeAndJobInfo(strconv.Itoa(i), string(uuid.NewString()))
	// subLogger.Trace().Msg(fmt.Sprintf("Trace: foo %s", "mank"))
	// subLogger.Debug().Msg(fmt.Sprintf("Debug: foo %s", "mank"))
	// subLogger.Info().Msg(fmt.Sprintf("Info: foo %s", "mank"))
	// subLogger.Warn().Msg(fmt.Sprintf("Warn: foo %s", "mank"))
	// subLogger.Error().Msg(fmt.Sprintf("Error: foo %s", "mank"))

	s := logger.LoggerWithRuntimeInfo(fmt.Sprintf("%d - %s", i, uuid.NewString()))

	s.Trace().Msgf("Trace: foo %s", "mank")
	s.Debug().Msgf("Debug: foo %s", "mank")
	s.Info().Msgf("Info: foo %s", "mank")
	s.Warn().Msgf("Warn: foo %s", "mank")
	s.Error().Msgf("Error: foo %s", "mank")

}
