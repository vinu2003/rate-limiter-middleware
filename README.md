Rate Limiter implementation which monitors the number of requests per a window time a service agrees to allow. If the request count exceeds the number agreed by the service owner, the rate limiter blocks all the excess calls by return HTTP Status 429. This implementation considers Ip address of the machine where the user logged in or ip address of any machine where another service runs.

This is sliding window in-memory with multi thread implementation. It's concurrent safe designed to access from multiple go routines without the risk of race conditions. Its makes use of deque paired with RW mutex to store the timestamps in sliding window and new keys are added when services/users from new machine makes requests.

The module is implemented as shared functionality that you want to run for many (or even all) HTTP requests. So, it is set up as middle ware.

```
Router => Middleware Handler(rate-limiter) => Application Handler
```

The design makes sure the below requirements and it works perfectly User request rate limiting - restricts each user to a predetermined number of requests per time unit. Concurrent user request limiting - limits the number of concurrent user requests that can be inflight at any given point in time.

-- Space Complexity: O(Max requests seen in a window time) — Stores all the timestamps of the requests in a time window -- Time Complexity: O(Max requests seen in a window time)— Deleting a subset of timestamps

However, the Cons are: -- In case of multiple users with large window time - memory is utilized more to store all timstamps in window time. -- Cleanup old timestamps also takes has impact here in terms of time complexity.

**WORKAROUND for memory issue:**  
Counter based design - store only count of access in the entire window time and map them in different time slots within that window time. But this again has its own side effects.

**SETUP:**  
Local Machine I have used is macbookPro - Version 10.15.3

GoLang installment setup: go version - go version go1.14.3 darwin/amd64

**Additional packages/Libraries used:**  
package for deque: go get -u github.com/tebeka/deque packages for testing : testify golang.org/x/net/context

run the module and if you make enough requests in quick successions you can notice HTTP status 429. $curl -i localhost:4000

I have found a toolthat could be used for HTTP load test. brew install vegeta create configuration file with request and run the belwo command

vegeta attack -duration=10s -rate=100 -targets=vegeta.conf | vegeta report you can notice some requets with 200 and many with 429 returned like belwo

```
vinodhinis-mbp:rate-limiter-middleware vinodhinibalusamy$ vegeta attack -duration=10s -rate=100 -targets=vegeta.conf | 
vegeta report Requests [total, rate, throughput] 1000, 100.09, 0.10 
Duration [total, attack, wait] 9.992s, 9.991s, 540.884µs 
Latencies [min, mean, 50, 90, 95, 99, max] 218.756µs, 527.888µs, 489.501µs, 596.04µs, 660.138µs, 1.661ms, 9.022ms 
Bytes In [total, mean] 17984, 17.98 
Bytes Out [total, mean] 0, 0.00 
Success [ratio] 0.10% 
Status Codes [code:count] 200:1 429:999
Error Set: 429 Too Many Requests
```

**Note:**  
I have set MAX requests allowed and WindowTimeInSec via receiver type in the limiter.go var ( RequestsAllowed int = 100 WindowTime int64 = 60 )

However, the unit test makes requestsAllowed as 1 and WindowTimeInSec as 1 to verify and test locally.