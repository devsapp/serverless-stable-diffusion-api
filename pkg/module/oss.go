package module

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	LOCAL  = "local"
	REMOTE = "remote"
)

type OssOp interface {
	UploadFile(ossKey, localFile string) error
	UploadFileByByte(ossKey string, body []byte) error
	DownloadFile(ossKey, localFile string) error
	DeleteFile(ossKey string) error
}

// OssGlobal oss manager
var OssGlobal OssOp

func NewOssManager() error {
	switch config.ConfigGlobal.OssMode {
	case LOCAL:
		// read/write with disk
		OssGlobal = new(OssManagerLocal)
	case REMOTE:
		client, err := oss.New(config.ConfigGlobal.OssEndpoint, config.ConfigGlobal.AccessKeyId,
			config.ConfigGlobal.AccessKeySecret, oss.SecurityToken(config.ConfigGlobal.AccessKeyToken))
		if err != nil {
			return err
		}
		bucket, err := client.Bucket(config.ConfigGlobal.Bucket)
		if err != nil {
			return err
		}
		OssGlobal = &OssManagerRemote{
			bucket: bucket,
		}
	default:
		log.Fatal("oss mode err")
	}
	return nil
}

type OssManagerRemote struct {
	bucket *oss.Bucket
}

// UploadFile upload file to oss
func (o *OssManagerRemote) UploadFile(ossKey, localFile string) error {
	// mode: remote
	return o.bucket.PutObjectFromFile(ossKey, localFile)
}

// UploadFileByByte UploadFile upload file to oss
func (o *OssManagerRemote) UploadFileByByte(ossKey string, body []byte) error {
	return o.bucket.PutObject(ossKey, bytes.NewReader(body))
}

// DownloadFile download file from oss
func (o *OssManagerRemote) DownloadFile(ossKey, localFile string) error {
	return o.bucket.GetObjectToFile(ossKey, localFile)
}

// DeleteFile delete file from oss
func (o *OssManagerRemote) DeleteFile(ossKey string) error {
	return o.bucket.DeleteObject(ossKey)
}

type OssManagerLocal struct {
}

func (o *OssManagerLocal) UploadFile(ossKey, localFile string) error {
	destFile := fmt.Sprintf("%s/%s", config.ConfigGlobal.OssPath, ossKey)
	cmd := exec.Command(fmt.Sprintf("cp %s %s", localFile, destFile))
	err := cmd.Run()
	return err
}
func (o *OssManagerLocal) UploadFileByByte(ossKey string, body []byte) error {
	destFile := fmt.Sprintf("%s/%s", config.ConfigGlobal.OssPath, ossKey)
	pathSlice := strings.Split(destFile, "/")
	path := strings.Join(pathSlice[:len(pathSlice)-1], "/")
	if !utils.FileExists(path) {
		os.MkdirAll(path, os.ModePerm)
	}
	fn, err := os.OpenFile(destFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return errors.New(fmt.Sprintf("upload fail because open file error, err=%s", err.Error()))
	}
	defer fn.Close()
	_, err = fn.Write(body)
	if err != nil {
		return errors.New("upload fail because write file error")
	}
	return nil
}
func (o *OssManagerLocal) DownloadFile(ossKey, localFile string) error {
	destFile := fmt.Sprintf("%s/%s", config.ConfigGlobal.OssPath, ossKey)
	cmd := exec.Command(fmt.Sprintf("cp %s %s", destFile, localFile))
	err := cmd.Run()
	return err
}
func (o *OssManagerLocal) DeleteFile(ossKey string) error {
	destFile := fmt.Sprintf("%s/%s", config.ConfigGlobal.OssPath, ossKey)
	_, err := utils.DeleteLocalModelFile(destFile)
	return err
}
