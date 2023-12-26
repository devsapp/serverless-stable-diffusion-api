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
	log.Println(find([]int{5, 6, 7, 8, 9, 10}, 8))
	log.Println(find([]int{5, 6, 7, 8, 9, 10}, 11))
	log.Println(find([]int{5, 6, 7, 8, 9, 10}, 5))
	log.Println(find([]int{5, 6, 7, 8, 9, 10}, 3))
}

func find(L []int, val int) int {
	low := 0
	high := len(L) - 1
	for low <= high {
		mid := (low + high) / 2
		if L[mid] < val {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	return low
}
