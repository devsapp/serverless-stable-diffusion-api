package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
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

func FileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	defer file.Close()
	if err != nil {
		return "", err
	}
	hash := md5.New()
	_, _ = io.Copy(hash, file)
	return hex.EncodeToString(hash.Sum(nil)), nil
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

// PortCheck port usable
func PortCheck(port string, timeout int) bool {
	if port == "" {
		return false
	}
	timeoutChan := time.After(time.Duration(timeout) * time.Millisecond)
	for {
		select {
		case <-timeoutChan:
			return false
		default:
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%s", port),
				time.Duration(10)*time.Millisecond)
			if err == nil && conn != nil {
				conn.Close()
				return true
			}
		}
	}
	return false
}

func DeleteLocalFile(localFile string) (bool, error) {
	_, err := os.Stat(localFile)
	if err == nil {
		if err := os.Remove(localFile); err == nil {
			return true, nil
		} else {
			return false, errors.New("delete model fail")
		}
	}
	if os.IsNotExist(err) {
		return false, errors.New("model not exist")
	}
	return false, err
}

// FileExists check file exist
func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func ListFile(path string) []string {
	fileSlice := make([]string, 0)
	files, _ := ioutil.ReadDir(path)
	for _, f := range files {
		fileSlice = append(fileSlice, f.Name())

	}
	return fileSlice
}

// DiffSet check a == b ? return diff
func DiffSet(old, new map[string]struct{}) ([]string, []string) {
	del := make([]string, 0)
	add := make([]string, 0)
	for k := range old {
		if _, ok := new[k]; !ok {
			del = append(del, k)
		}
	}
	for k := range new {
		if _, ok := old[k]; !ok {
			add = append(add, k)
		}
	}
	return add, del
}

func IsSame(key string, a, b interface{}) bool {
	switch a.(type) {
	case []interface{}:
		aT := a.([]interface{})
		bT := b.([]interface{})
		if len(aT) != len(bT) {
			return false
		}
		for i, _ := range aT {
			if aT[i] != bT[i] {
				return false
			}
		}
		return true
	case int64, int32, int, int16, int8, string, float64, float32, bool:
		return a == b
	default:
		logrus.Info(key, a, b)
		logrus.Fatal("type not support")
	}
	return true
}

// MapToStruct map to struct
func MapToStruct(m map[string]interface{}, s interface{}) error {
	jsonData, err := json.Marshal(m)
	if err != nil {
		return err
	}
	logrus.Info(string(jsonData))
	err = json.Unmarshal(jsonData, s)
	if err != nil {
		return err
	}

	return nil
}
