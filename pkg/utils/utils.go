package utils

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"time"
)

// RandStr product random string
func RandStr(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	rand.Seed(time.Now().UnixNano() + int64(rand.Intn(100)))
	for i := 0; i < length; i++ {
		result = append(result, bytes[rand.Intn(len(bytes))])
	}
	return string(result)
}

func Hash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs[:32])
}

func TimestampS() int64 {
	return time.Now().Unix()
}

func TimestampMS() int64 {
	return time.Now().UnixNano() / 1e6
}

func String(s string) *string {
	return &s
}

func Int32(v int32) *int32 {
	return &v
}

func Float32(v float32) *float32 {
	return &v
}

func Bool(v bool) *bool {
	return &v
}
