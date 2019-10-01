package nettools

import (
    "container/list"
    "github.com/youtube/vitess/go/cache"
    "time"
)

const (
    HOUR   = 60
    MINUTE = 60
)

type CacheValue struct {
    slidingWindow *list.List
}

func (cv CacheValue) Size() int {
    return 1
}

// record how many requests the DHT has received for each minute
type Entry struct {
    tsMinute int64 // unix timestamp / 60 seconds
    reqCount int
}

// NewThrottle creates a new client throttler that blocks spammy clients.
func NewThrottler(maxPerMinute int, maxHosts int64) *ClientThrottle {
    r := ClientThrottle{
        maxPerHour: maxPerMinute * HOUR,
        c:          cache.NewLRUCache(maxHosts),
    }
    return &r
}

// ClientThrottle identifies and blocks hosts that are too spammy. It only
// cares about the number of operations per hour.
type ClientThrottle struct {
    maxPerHour int
    c          *cache.LRUCache // Rate limiter.
}

func (r *ClientThrottle) Stop() {
    r.c.Clear()
}

/*
1. If the host is not in c, add it to c.
   If c is full, this operation will evict the least recently seen host in c

2. If the host is in c
  1) get the current timestamp, remove outdated entry from the list
  2) if total reqCount + 1 > maxPerHour, return false; otherwise return true
*/
func (r *ClientThrottle) CheckBlock(host string) bool {
    currTSMinute := int64(time.Now().Unix() / MINUTE)
    v, ok := r.c.Get(host)
    if !ok {
        l := list.New()
        entry := &Entry{
            tsMinute: currTSMinute,
            reqCount: 1,
        }
        l.PushBack(entry)
        cacheValue := CacheValue{slidingWindow: l}
        r.c.Set(host, cacheValue)
        return true
    }

    cv := v.(CacheValue)
    totalReqCount := 0 // total requests within one hour

    for e := cv.slidingWindow.Front(); e != nil; {
        if currTSMinute-e.Value.(*Entry).tsMinute >= 60 {
            next := e.Next()
            cv.slidingWindow.Remove(e)
            e = next
        } else {
            totalReqCount += e.Value.(*Entry).reqCount
            e = e.Next()
        }
    }

    if totalReqCount+1 > r.maxPerHour {
        return false
    }

    if e := cv.slidingWindow.Back(); e == nil {
        entry := &Entry{
            tsMinute: currTSMinute,
            reqCount: 1,
        }
        cv.slidingWindow.PushBack(entry)
    } else {
        if e.Value.(*Entry).tsMinute == currTSMinute {
            e.Value.(*Entry).reqCount++
        } else {
            entry := &Entry{
                tsMinute: int64(currTSMinute),
                reqCount: 1,
            }
            cv.slidingWindow.PushBack(entry)
        }
    }
    return true
}
