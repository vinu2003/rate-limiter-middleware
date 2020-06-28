package main

import (
	"fmt"
	"log"
	"sync"
	"net/http"
)

func FinalHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func main() {

	mux := http.NewServeMux()
	mData := new(middlewareData)

	mData.slidingWindowLimiter = &SlidingWindowRateLimiter{
		rlMutex: *new(sync.RWMutex),
		rateLimiterMap: make(map[string]*TimeStampsBucket),
	}

	limitHandler := http.HandlerFunc(FinalHandler)
	mux.Handle("/", mData.rateLimiterMiddleWare(limitHandler))

	log.Println("Listening on port : 4000...")
	err := http.ListenAndServe(":4000", mux)
	if err != nil {
		fmt.Println("", err.Error())
	}
}