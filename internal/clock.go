package internal

import "time"

type Clock interface {
	Since() <-chan time.Duration
}

type clock time.Time

func NewClock() Clock {
	return clock(time.Now())
}
func (c clock) Since() <-chan time.Duration {
	durChan := make(chan time.Duration, 1)
	durChan <- time.Since(time.Time(c))
	return durChan
}
