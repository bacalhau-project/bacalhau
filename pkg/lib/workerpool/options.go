package workerpool

type OptionFn func(*WorkerPoolConfig)

// WithWorkerCount
func WithWorkerCount(count int) OptionFn {
	return func(cfg *WorkerPoolConfig) {
		cfg.workerCount = count
	}
}

// WithInputChannelSize sets the size of the input queue used by the worker
// pool. By default it is unbuffered (a size of 1) but this can be changed
// to any larger value to make it use a buffered queue. When choosing an
// input queue size, you should pay attention to the number of workers to
// ensure you don't end up accidentally applying back pressure to the
// process adding items.
func WithInputChannelSize(length int) OptionFn {
	return func(cfg *WorkerPoolConfig) {
		cfg.inputChannelSize = length
	}
}
