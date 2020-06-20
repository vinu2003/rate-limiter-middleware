package main

import (
	"errors"
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

type middlewareData struct {
	RequestsAllowed int
	WindowTime int64
}

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
func (ts *TimeStampsBucket) cleanupOlderTimeStamps(currentTimeStamp int64) error {
	if ts.timestamps == nil {
		log.Println("Nothing to cleanup in the timestamp bucket.")
		return errors.New("timestamp deque is nil")
	}

	for {
		val, _ := ts.timestamps.Get(0)
		if ts.timestamps.Len() != 0 && (currentTimeStamp-val.(int64)) > ts.windowTimeInSecs {
			// just cleanup from front of queue - bound check is done earlier
			_, _ = ts.timestamps.PopLeft()
		} else {
			break
		}
	}

	return nil
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
func (rl * SlidingWindowRateLimiter) AddIPAddress(ipAddress string, requestsAllowed int, windowTimeInSecs int64) error {
	if ipAddress == "" || requestsAllowed == 0 || windowTimeInSecs == 0 {
		log.Println(" Invalid input provided")
		return errors.New("invalid inputs provided")
	}

	rl.rlMutex.Lock()
	defer rl.rlMutex.Unlock()

	rl.rateLimiterMap[ipAddress] = &TimeStampsBucket{
		timestamps: deque.New(),
		requestsAllowed: requestsAllowed,
		windowTimeInSecs: windowTimeInSecs,
		tsMutex: *new(sync.RWMutex),
	}

	return nil
}

// remove IPAddress for cleanup purpose
func (rl *SlidingWindowRateLimiter) removeIpAddress(ipAddress string) {
	rl.rlMutex.Lock()
	defer rl.rlMutex.Unlock()

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

	if _, ok := rl.rateLimiterMap[ipAddress]; ok == false {
		log.Println("Unable to get IP address key")
		return false
	}
	ipaddAccess := rl.rateLimiterMap[ipAddress]
	ipaddAccess.tsMutex.Lock()
	defer ipaddAccess.tsMutex.Unlock()

	//cleanup existing older timestamps for the ipAddress
	currentTimeStamp := time.Now().Unix()
	err := ipaddAccess.cleanupOlderTimeStamps(currentTimeStamp)
	if err != nil {
		log.Println("encountered problem in cleaning up the timeStamps queue", err.Error())
	}

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

// split the ip address from port from http request
func splitIpAddress(remoteAddress string) (string, error) {
	if remoteAddress == "" {
		return "", errors.New("empty string is provided as remote address")
	}

	ip, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		return "", errors.New("internal server error")
	}

	return ip, nil
}

// This global - TODO needs to be improved.
var slidingWindowLimiter = SlidingWindowRateLimiter{
	rlMutex:        *new(sync.RWMutex),
	rateLimiterMap: make(map[string]*TimeStampsBucket),
}

// middle ware method
func (mData *middlewareData) rateLimiterMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, err := splitIpAddress(r.RemoteAddr)
		if err != nil {
			log.Println("Error: Error in processing the SplitHostPort function - ", err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if mData.RequestsAllowed == 0 || mData.WindowTime == 0 {
			mData.RequestsAllowed = 100
			mData.WindowTime = int64(60)
		}

		// check if IP address exists
		if slidingWindowLimiter.ifIPAddrExists(ip) == false {
			_ = slidingWindowLimiter.AddIPAddress(ip, mData.RequestsAllowed, mData.WindowTime)
		}

		if slidingWindowLimiter.AllowAccessToIPAddr(ip) == false {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}