package utils

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
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
	log.Println(hash)
	assert.Equal(t, 64, len(hash))
}

func TestTimestampMS(t *testing.T) {
	cur := TimestampMS()
	assert.Equal(t, len(fmt.Sprintf("%d", cur)), 13)
}
