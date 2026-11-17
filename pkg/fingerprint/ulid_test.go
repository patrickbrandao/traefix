package fingerprint

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewULID(t *testing.T) {
	// The time part of the spec example: 1469918176385 ms is 01ARYZ6S41.
	id := newULID(time.UnixMilli(1469918176385))

	assert.Len(t, id, 26)
	assert.Equal(t, "01ARYZ6S41", id[:10])
	assert.NotEqual(t, id, newULID(time.UnixMilli(1469918176385)))
	assert.Less(t, newULID(time.UnixMilli(1)), newULID(time.UnixMilli(2)))
}
