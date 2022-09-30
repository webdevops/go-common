package collector

import (
	"sync"
)

var (
	lock sync.RWMutex
)

func Lock() sync.RWMutex {
	return lock
}
