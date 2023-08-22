package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRandStr(t *testing.T) {
	length := 10
	randStr := RandStr(length)
	assert.Equal(t, length, len(randStr))
}

func TestHash(t *testing.T) {
	s := "dddddd"
	hash := Hash(s)
	assert.Equal(t, 32, len(hash))
}
