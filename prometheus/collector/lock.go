package collector

import (
	"net/http"
	"sync"
)

var (
	lock sync.RWMutex
)

func Lock() *sync.RWMutex {
	return &lock
}

func HttpWaitForRlock(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lock.RLock()
		defer lock.RUnlock()
		handler.ServeHTTP(w, r)
	})
}
