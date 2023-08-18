package module

import (
	"bytes"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
)

// OssGlobal oss manager
var OssGlobal *OssManager

type OssManager struct {
	bucket *oss.Bucket
}

func NewOssManager() error {
	client, err := oss.New(config.ConfigGlobal.OssEndpoint, config.ConfigGlobal.AccessKeyId,
		config.ConfigGlobal.AccessKeySecret)
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
