package nettools

import (
    "testing"
    "time"
)

func TestRateLimiter(t *testing.T) {
    throttler := NewThrottler(1, 3)
    for i := 0; i < 60; i++ {
        time.Sleep(3 * time.Second)
        if throttler.CheckBlock("8.8.8.8") != true {
            t.Errorf("incorrect result, expected true, got false")
        }
    }

    if throttler.CheckBlock("8.8.8.8") != false {
        t.Errorf("incorrect result, expected false, got true")
    }
}
