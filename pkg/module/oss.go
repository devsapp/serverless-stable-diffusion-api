package module

import (
	"bytes"
	"encoding/base64"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"io/ioutil"
)

// OssGlobal oss manager
var OssGlobal *OssManager

type OssManager struct {
	bucket *oss.Bucket
}

func NewOssManager() error {
	client, err := oss.New(config.ConfigGlobal.OssEndpoint, config.ConfigGlobal.AccessKeyId,
		config.ConfigGlobal.AccessKeySecret, oss.SecurityToken(config.ConfigGlobal.AccessKeyToken))
	if err != nil {
		return err
	}
	bucket, err := client.Bucket(config.ConfigGlobal.Bucket)
	if err != nil {
		return err
	}
	OssGlobal = &OssManager{
		bucket: bucket,
	}
	return nil
}

// UploadFile upload file to oss
func (o *OssManager) UploadFile(ossKey, localFile string) error {
	return o.bucket.PutObjectFromFile(ossKey, localFile)
}

// UploadFileByByte UploadFile upload file to oss
func (o *OssManager) UploadFileByByte(ossKey string, body []byte) error {
	return o.bucket.PutObject(ossKey, bytes.NewReader(body))
}

// DownloadFile download file from oss
func (o *OssManager) DownloadFile(ossKey, localFile string) error {
	return o.bucket.GetObjectToFile(ossKey, localFile)
}

// DeleteFile delete file from oss
func (o *OssManager) DeleteFile(ossKey string) error {
	return o.bucket.DeleteObject(ossKey)
}

// DownloadFileToBase64 Download the object into ReadCloser(). The body needs to be closed
func (o *OssManager) DownloadFileToBase64(ossKey string) (string, error) {
	body, err := o.bucket.GetObject(ossKey)
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadAll(body)
	body.Close()
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}
