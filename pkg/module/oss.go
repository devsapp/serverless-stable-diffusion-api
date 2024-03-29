package module

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	expiredInSec = 3 * 60 * 60 * 1000
)

type OssOp interface {
	UploadFile(ossKey, localFile string) error
	UploadFileByByte(ossKey string, body []byte) error
	DownloadFile(ossKey, localFile string) error
	DeleteFile(ossKey string) error
	DownloadFileToBase64(ossPath string) (*string, error)
	GetUrl(ossPath []string) ([]string, error)
}

// OssGlobal oss manager
var OssGlobal OssOp

func NewOssManager() error {
	switch config.ConfigGlobal.OssMode {
	case config.LOCAL:
		// read/write with disk
		OssGlobal = new(OssManagerLocal)
	case config.REMOTE:
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

func (o *OssManagerRemote) GetUrl(ossKeys []string) ([]string, error) {
	ossUrl := make([]string, 0, len(ossKeys))
	for _, key := range ossKeys {
		url, err := o.bucket.SignURL(key, oss.HTTPGet, expiredInSec)
		if err != nil {
			return nil, errors.New("get")
		}
		ossUrl = append(ossUrl, url)
	}
	return ossUrl, nil
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

func (o *OssManagerRemote) DownloadFileToBase64(ossKey string) (*string, error) {
	// get image from oss
	body, err := o.bucket.GetObject(ossKey)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(body)
	body.Close()
	if err != nil {
		return nil, err
	}
	// image to base64
	imageBase64 := base64.StdEncoding.EncodeToString(data)
	return &imageBase64, nil
}

type OssManagerLocal struct {
}

func (o *OssManagerLocal) GetUrl(ossKey []string) ([]string, error) {
	return nil, errors.New("not support")
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
	cmd := exec.Command("cp", destFile, localFile)
	err := cmd.Run()
	return err
}
func (o *OssManagerLocal) DeleteFile(ossKey string) error {
	destFile := fmt.Sprintf("%s/%s", config.ConfigGlobal.OssPath, ossKey)
	_, err := utils.DeleteLocalFile(destFile)
	return err
}

// DownloadFileToBase64 : support png/jpg/jpeg
func (o *OssManagerLocal) DownloadFileToBase64(ossKey string) (*string, error) {
	destFile := fmt.Sprintf("%s/%s", config.ConfigGlobal.OssPath, ossKey)
	if !utils.FileExists(destFile) {
		return nil, fmt.Errorf("ossKey:%s not exist", ossKey)
	}
	data, err := ioutil.ReadFile(destFile)
	if err != nil {
		return nil, err
	}
	imageBase64 := base64.StdEncoding.EncodeToString(data)
	return &imageBase64, nil
}
