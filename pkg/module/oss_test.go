package module

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestOss(t *testing.T) {
	NewOssManager()
	objKey := "sd/test"
	// upload
	err := OssGlobal.UploadFileByByte(objKey, []byte("oss test"))
	assert.Nil(t, err)

	// download
	downloadFile := "downloadFile"
	err = OssGlobal.DownloadFile(objKey, downloadFile)
	assert.Nil(t, err)

	// delete
	err = OssGlobal.DeleteFile(objKey)
	assert.Nil(t, err)
	err = os.Remove(downloadFile)
	assert.Nil(t, err)
}
