package to

import (
	"testing"
	"time"
)

func Test_UnixTime(t *testing.T) {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		t.Errorf("Error occurred during loading location UTC: %v", err)
	}
	timestamp := time.Date(2020, 01, 01, 0, 0, 0, 0, loc)

	val := UnixTime(timestamp)
	if val != 1577836800 {
		t.Errorf("Expected time: %v  Actual time: %v", 1577836800, val)
	}
}
