package main

import (
	"github.com/tebeka/deque"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// *************************************************************
// In memory implementation - single machine with multiple users
// *************************************************************


// **********************************************************
// Timestamp bucket for each IPAddress
//
// deque implement double ended queue for storing timestamps.
// supports below functions
// Append - appends an item to the right of deque
// AppendLeft -  appends an item to left of deque
// Pop - pop removes & returns the rightmost element
// PopLeft - removed & returns leftmost element
//
// requests - max requests allowed for each user
//
// windowTimeInSec - sliding window time allowed
//
// mutex is lock for concurrency in multi threaded system
// **********************************************************
type TimeStampsBucket struct {
	timestamps *deque.Deque
	requestsAllowed int
	windowTimeInSecs int64
	tsMutex sync.RWMutex
}

// cleanup older timestamps older than the given time for the user
// Called for each request to server made
// TODO - improve by calling this periodically after 'n'mins or when a certain length is reached
func (ts *TimeStampsBucket) cleanupOlderTimeStamps(currentTimeStamp int64) {
	if ts.timestamps == nil {
		log.Println("Nothing to cleanup in the timestamp bucket.")
		return
	}

		for {
			val, _ := ts.timestamps.Get(0)
			if ts.timestamps.Len() != 0 && (currentTimeStamp - val.(int64)) > ts.windowTimeInSecs {
				// just cleanup from front of queue - bound check is done earlier
				_, _ = ts.timestamps.PopLeft()
			} else {
				break
			}
		}
}

// ********************************************************************
// Maintains map of Ip Addr and its corresponding timestamp bucket list
// ********************************************************************
type SlidingWindowRateLimiter struct {
	rlMutex sync.RWMutex
	rateLimiterMap map[string]*TimeStampsBucket
}

// check if IP address exists in sliding window
func (rl *SlidingWindowRateLimiter) ifIPAddrExists(ipAddress string) bool {
	rl.rlMutex.Lock()
	defer rl.rlMutex.Unlock()

	if _, ok := rl.rateLimiterMap[ipAddress]; ok == true {
		return true
	}

	return false
}

// Add New IpAddress in sliding window
func (rl * SlidingWindowRateLimiter) AddIPAddress(ipAddress string, requestsAllowed int, windowTimeInSecs int64) {
	rl.rlMutex.Lock()
	defer rl.rlMutex.Unlock()

	rl.rateLimiterMap = make(map[string]*TimeStampsBucket)

	rl.rateLimiterMap[ipAddress] = &TimeStampsBucket{
		requestsAllowed: requestsAllowed,
		windowTimeInSecs: windowTimeInSecs,
	}
}

// remove IPAddress for cleanup purpose
func (rl *SlidingWindowRateLimiter) removeIpAddress(ipAddress string) {
	rl.rlMutex.Lock()
	defer rl.rlMutex.Unlock()

	// TODO check if ip address exists

	if rl.ifIPAddrExists(ipAddress) {
		delete(rl.rateLimiterMap,ipAddress)
	}
}


// Allow Access to ipAddress
// before allowing access make sure the tokenBucket for that ipAddr is cleaned up (older timestamps removed)
// append the current timestamp when the request is made and then verify
func (rl *SlidingWindowRateLimiter) AllowAccessToIPAddr(ipAddress string) bool {
	rl.rlMutex.Lock()
	defer rl.rlMutex.Unlock()

	ipaddAccess := rl.rateLimiterMap[ipAddress]
	ipaddAccess.tsMutex.Lock()
	defer ipaddAccess.tsMutex.Unlock()

	//cleanup existing older timestamps for the ipAddress
	currentTimeStamp := time.Now().Unix()
	ipaddAccess.cleanupOlderTimeStamps(currentTimeStamp)

	// append new timestamp in deque front
	if ipaddAccess.timestamps == nil {
		ipaddAccess.timestamps = deque.New()
	}
	ipaddAccess.timestamps.Append(currentTimeStamp)

	// if length of timeStamp bucket is greater than requests allowed then do not give access
	if ipaddAccess.timestamps.Len() > ipaddAccess.requestsAllowed {
		return false
	}

	return true
}

// This global - TODO needs to be improved.
var slidingWindowLimiter = new(SlidingWindowRateLimiter)

// middle ware method
func rateLimiterMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// check if IP address exists
		if slidingWindowLimiter.ifIPAddrExists(ip) == false {
			log.Println("ALWAYS ADDING....")
			slidingWindowLimiter.AddIPAddress(ip, 3, 1)
		}

		if slidingWindowLimiter.AllowAccessToIPAddr(ip) == false {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}