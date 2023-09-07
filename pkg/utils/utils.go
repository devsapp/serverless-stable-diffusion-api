package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"gocv.io/x/gocv"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"strings"
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

func DeleteLocalModelFile(localFile string) (bool, error) {
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

func ImageToBase64(path string, ext gocv.FileExt) (*string, error) {
	imageMat := gocv.IMRead(path, gocv.IMReadColor)
	imageBytes, err := gocv.IMEncode(ext, imageMat)
	if err != nil {
		return String(""), err
	}
	imageBase64 := base64.StdEncoding.EncodeToString(imageBytes.GetBytes())
	return &imageBase64, nil
}

func ImageType(imageFn string) (gocv.FileExt, error) {
	fileExt := gocv.PNGFileExt
	imgTypeSlice := strings.Split(imageFn, ".")
	switch imgTypeSlice[len(imgTypeSlice)-1] {
	case "png":
		fileExt = gocv.PNGFileExt
	case "jpg", "jpeg":
		fileExt = gocv.JPEGFileExt
	default:
		return "", errors.New("img type not support")
	}
	return fileExt, nil
}

func ListFile(path string) []string {
	fileSlice := make([]string, 0)
	files, _ := ioutil.ReadDir(path)
	for _, f := range files {
		fileSlice = append(fileSlice, f.Name())

	}
	return fileSlice
}
