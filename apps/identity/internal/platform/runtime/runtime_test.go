package runtime

import (
	"testing"
	"time"
)

func TestClockReturnsUTC(t *testing.T) {
	if (Clock{}).Now().Location() != time.UTC {
		t.Fatal("clock is not UTC")
	}
}
