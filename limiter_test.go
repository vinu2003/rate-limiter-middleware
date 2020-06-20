package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/tebeka/deque"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func Test_splitIpAddressWithEmptyStringAsInput(t *testing.T) {
	expectedOutput := ""
	outputString, err := splitIpAddress("")

	// check the response
	assert.Equal(t, expectedOutput, outputString)
	assert.Equal(t, err.Error(), "empty string is provided as remote address")
}

func Test_splitIpAddressWithValidStringAsInput(t *testing.T) {
	expectedOutput := "127.0.0.1"
	outputString, err := splitIpAddress("127.0.0.1:63291")

	// check response
	assert.Equal(t, nil, err)
	assert.Equal(t, expectedOutput, outputString)
}

func Test_rateLimiterMiddleWareSingleRequest(t *testing.T) {
	req, err := http.NewRequest("GET", "http://localhost:4000", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "127.0.0.1:54323"

	rr := httptest.NewRecorder()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	mData := middlewareData{
		RequestsAllowed: 1,
		WindowTime:      1,
	}
	handlerToTest := mData.rateLimiterMiddleWare(nextHandler)
	handlerToTest.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func Test_rateLimiterMiddleWareMultipleRequest(t *testing.T) {
	req1, err := http.NewRequest("GET", "http://localhost:4000", nil)
	if err != nil {
		t.Fatal(err)
	}
	req1.RemoteAddr = "127.0.0.1:54321"

	req2, err := http.NewRequest("GET", "http://localhost:4000", nil)
	if err != nil {
		t.Fatal(err)
	}
	req2.RemoteAddr = "127.0.0.1:54322"

	req3, err := http.NewRequest("GET", "http://localhost:4000", nil)
	if err != nil {
		t.Fatal(err)
	}
	req3.RemoteAddr = "127.0.0.1:54323"

	req4, err := http.NewRequest("GET", "http://localhost:4000", nil)
	if err != nil {
		t.Fatal(err)
	}
	req4.RemoteAddr = "127.0.0.1:54324"

	req5, err := http.NewRequest("GET", "http://localhost:4000", nil)
	if err != nil {
		t.Fatal(err)
	}
	req5.RemoteAddr = "127.0.0.1:54325"

	req6, err := http.NewRequest("GET", "http://localhost:4000", nil)
	if err != nil {
		t.Fatal(err)
	}
	req6.RemoteAddr = "127.0.0.1:54326"

	requests := [...]*http.Request{req1, req2, req3, req4, req5, req6}

	rr := httptest.NewRecorder()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	mData := middlewareData{
		RequestsAllowed: 1,
		WindowTime:int64(1),
	}
	handlerToTest := mData.rateLimiterMiddleWare(nextHandler)

	for _, req := range requests {
		handlerToTest.ServeHTTP(rr, req)
		// Check the status code is what we expect.
		if status := rr.Code; status != http.StatusTooManyRequests {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusTooManyRequests)
		}
	}
}

func Test_cleanupOlderTimeStampsWhichIsNil(t *testing.T) {
	currentTimeStamp := int64(100)
	ts := new(TimeStampsBucket)
	ts.timestamps = nil

	err := ts.cleanupOlderTimeStamps(currentTimeStamp)

	assert.Equal(t, "timestamp deque is nil", err.Error())
}

func Test_cleanupOlderTimeStampsWithOneEntry(t *testing.T) {
	currentTimeStamp := int64(100)
	ts := new(TimeStampsBucket)
	ts.timestamps = deque.New()
	ts.timestamps.Append(int64(99))

	err := ts.cleanupOlderTimeStamps(currentTimeStamp)

	assert.Equal(t, nil, err)
	// check length of queue
	assert.Equal(t, 0, ts.timestamps.Len())
}

func Test_cleanupOlderTimeStampsWithMixedEntries(t *testing.T) {
	currentTimeStamp := int64(100)
	ts := new(TimeStampsBucket)
	ts.timestamps = deque.New()
	ts.timestamps.Append(int64(99))
	ts.timestamps.Append(int64(100))

	err := ts.cleanupOlderTimeStamps(currentTimeStamp)

	assert.Equal(t, nil, err)
	// check length of queue
	assert.Equal(t, 1, ts.timestamps.Len())
}

func Test_ifIPAddrExistsEmptyRateLimiterMap(t *testing.T) {
	rl := new(SlidingWindowRateLimiter)
	rl.rlMutex = *new(sync.RWMutex)
	rl.rateLimiterMap = make(map[string]*TimeStampsBucket, 0)

	result := rl.ifIPAddrExists("127.0.0.1")

	assert.Equal(t, false, result)
}

func Test_ifIPAddrExistsNotHavingIPaddr(t *testing.T) {
	rl := new(SlidingWindowRateLimiter)
	rl.rlMutex = *new(sync.RWMutex)
	rl.rateLimiterMap = make(map[string]*TimeStampsBucket, 0)

	rl.rateLimiterMap["1.1.1.1"] = &TimeStampsBucket{timestamps: &deque.Deque{},
		requestsAllowed: 1,
		windowTimeInSecs: 1,
		tsMutex: sync.RWMutex{},
		}

	result := rl.ifIPAddrExists("127.0.0.1")

	assert.Equal(t, false, result)
}

func Test_ifIPAddrExistsHavingActualIpAddr(t *testing.T) {
	rl := new(SlidingWindowRateLimiter)
	rl.rlMutex = *new(sync.RWMutex)
	rl.rateLimiterMap = make(map[string]*TimeStampsBucket, 0)

	rl.rateLimiterMap["1.1.1.1"] = &TimeStampsBucket{timestamps: &deque.Deque{},
		requestsAllowed: 1,
		windowTimeInSecs: 1,
		tsMutex: sync.RWMutex{},
	}
	rl.rateLimiterMap["127.0.0.1"] = &TimeStampsBucket{timestamps: &deque.Deque{},
		requestsAllowed: 1,
		windowTimeInSecs: 1,
		tsMutex: sync.RWMutex{},
	}

	result := rl.ifIPAddrExists("127.0.0.1")

	assert.Equal(t, true, result)
}

func Test_AddIPAddressInvalidInput(t *testing.T) {
	rl := new(SlidingWindowRateLimiter)
	rl.rlMutex = *new(sync.RWMutex)
	rl.rateLimiterMap = make(map[string]*TimeStampsBucket, 0)

	err := rl.AddIPAddress("", 1, 1)

	assert.Equal(t, "invalid inputs provided", err.Error())
}

func Test_AddIPAddressValidInput(t *testing.T) {
	rl := new(SlidingWindowRateLimiter)
	rl.rlMutex = *new(sync.RWMutex)
	rl.rateLimiterMap = make(map[string]*TimeStampsBucket, 0)

	err := rl.AddIPAddress("127.0.0.1", 1, 1)

	assert.Equal(t, nil, err)

	assert.Contains(t, rl.rateLimiterMap, "127.0.0.1")
}

func Test_AllowAccessInvalidIPAddress(t *testing.T) {
	rl := new(SlidingWindowRateLimiter)
	rl.rlMutex = *new(sync.RWMutex)
	rl.rateLimiterMap = make(map[string]*TimeStampsBucket, 0)

	_ = rl.AddIPAddress("127.0.0.1", 1, 1)

	result := rl.AllowAccessToIPAddr("1.1.1.1")
	assert.Equal(t, false, result)
}

func Test_AllowAccessValidIPAddress(t *testing.T) {
	rl := new(SlidingWindowRateLimiter)
	rl.rlMutex = *new(sync.RWMutex)
	rl.rateLimiterMap = make(map[string]*TimeStampsBucket, 0)

	_ = rl.AddIPAddress("127.0.0.1", 1, 1)

	result := rl.AllowAccessToIPAddr("127.0.0.1")
	assert.Equal(t, true, result)
}

