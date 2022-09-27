# golang-mutex-tracer
Helps you debug slow lock/unlocks for golang sync Mutex/RWMutex 

Example usage
----
Import with an alias for minimal impact. 

Enable per lock:
```
import sync "github.com/RobinUS2/golang-mutex-tracer"

l := sync.Mutex{}
l.EnableTracer()
l.Lock()
l.Unlock()
```

Enable with customer settings per lock:
```
import sync "github.com/RobinUS2/golang-mutex-tracer"

l := sync.Mutex{}
l.EnableTracerWithOpts(sync.Opts{
    Threshold: 10 * time.Millisecond,
})
l.Lock()
l.Unlock()
```

Enable with customer name for lock:
```
import sync "github.com/RobinUS2/golang-mutex-tracer"

l := sync.Mutex{}
l.EnableTracerWithOpts(sync.Opts{
    Threshold: 10 * time.Millisecond,
    Id: "myLock",
})
l.Lock()
l.Unlock()
```

Enable for all locks (that use the import):
```
import sync "github.com/RobinUS2/golang-mutex-tracer"

l := sync.Mutex{}
sync.SetGlobalOpts(sync.Opts{
    Threshold: 100 * time.Millisecond,
    Enabled:   true,
})
l.Lock()
l.Unlock()
```

Example output
----
```
2019/02/20 13:32:04 testLock violation CRITICAL section took 23.477ms 23477000 (threshold 10ms)
```

Benchmark
----
Yes, there is performance impact. This is in the order of 1000 nanoseconds which is 0.001 milliseconds.
However, the purpose of this project is to debug long blocked locks / contention, it should not be used
continuously during production. 
```
goos: darwin
goarch: amd64
BenchmarkRWNativeLock-8                          	 5000000	       381 ns/op
BenchmarkRWTracerLockDisabled-8                  	 5000000	       377 ns/op
BenchmarkRWTracerLockEnabled-8                   	 5000000	       385 ns/op
BenchmarkRWNativeLockWithConcurrency-8           	  200000	      6536 ns/op
BenchmarkRWTracerLockDisabledWithConcurrency-8   	  200000	      6504 ns/op
BenchmarkRWTracerLockEnabledWithConcurrency-8    	  200000	      6435 ns/op
BenchmarkNativeLock-8                            	100000000	        15.0 ns/op
BenchmarkTracerLockDisabled-8                    	 5000000	       366 ns/op
BenchmarkTracerLockEnabled-8                     	 5000000	       373 ns/op
BenchmarkNativeLockWithConcurrency-8             	 1000000	      1019 ns/op
BenchmarkTracerLockDisabledWithConcurrency-8     	  200000	      6431 ns/op
BenchmarkTracerLockEnabledWithConcurrency-8      	  200000	      6396 ns/op
```