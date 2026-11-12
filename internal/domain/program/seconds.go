package program

import (
	"fmt"
	"time"
)

const (
	MinSeconds = 1
	MaxSeconds = 120
)

type Seconds time.Duration

func (s Seconds) Int() int {
	return int(time.Duration(s).Seconds())
}

func (s Seconds) Duration() time.Duration {
	return time.Duration(s)
}

func ParseSeconds(i int) (Seconds, error) {
	if i < MinSeconds || i > MaxSeconds {
		return 0, InvalidSecondsError{seconds: i}
	}
	return HydrateSeconds(i), nil
}

func HydrateSeconds(i int) Seconds {
	return Seconds(time.Duration(i) * time.Second)
}

type InvalidSecondsError struct {
	seconds int
}

func (i InvalidSecondsError) Error() string {
	return fmt.Sprintf("seconds must be between %d and %d, got %d", MinSeconds, MaxSeconds, i.seconds)
}
