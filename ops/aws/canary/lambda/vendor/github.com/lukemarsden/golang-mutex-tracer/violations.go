package muxtracer

type ViolationType int

const (
	ViolationDefault  ViolationType = iota // 0, unused
	ViolationLock                          // 1, time to lock (awaiting)
	ViolationCritical                      // 2, time spent in the lock
)

func (v ViolationType) String() string {
	switch v {
	case ViolationLock:
		return "LOCK"
	case ViolationCritical:
		return "CRITICAL"
	case ViolationDefault:
		fallthrough
	default:
		panic("should never happen")
	}
}
