package handler

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/module"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"io/ioutil"
	"os"
)

const (
	taskIdLength     = 10
	userKey          = "username"
	requestType      = "Request-Type"
	taskKey          = "taskId"
	FcAsyncKey       = "X-Fc-Invocation-Type"
	versionKey       = "version"
	requestOk        = 200
	asyncSuccessCode = 202
	syncSuccessCode  = 200
)

func getBindResult(c *gin.Context, in interface{}) error {
	if err := binding.JSON.Bind(c.Request, in); err != nil {
		return err
	}
	return nil
}

func outputImage(fileName, base64Str *string) error {
	decode, err := base64.StdEncoding.DecodeString(*base64Str)
	if err != nil {
		return fmt.Errorf("base64 decode err=%s", err.Error())
	}
	if err := ioutil.WriteFile(*fileName, decode, 0666); err != nil {
		return fmt.Errorf("writer image err=%s", err.Error())
	}
	return nil
}

func downloadModelsFromOss(modelsType, ossPath, modelName string) (string, error) {
	path := ""
	switch modelsType {
	case config.SD_MODEL:
		path = fmt.Sprintf("%s/models/%s/%s", config.ConfigGlobal.SdPath, "Stable-diffusion", modelName)
	case config.SD_VAE:
		path = fmt.Sprintf("%s/models/%s/%s", config.ConfigGlobal.SdPath, "VAE", modelName)
	case config.LORA_MODEL:
		path = fmt.Sprintf("%s/models/%s/%s", config.ConfigGlobal.SdPath, "Lora", modelName)
	case config.CONTORLNET_MODEL:
		path = fmt.Sprintf("%s/models/%s/%s", config.ConfigGlobal.SdPath, "ControlNet", modelName)
	default:
		return "", fmt.Errorf("modeltype: %s not support", modelsType)
	}
	if err := module.OssGlobal.DownloadFile(ossPath, path); err != nil {
		return "", err
	}
	return path, nil
}

func uploadImages(ossPath, imageBody *string) error {
	decode, err := base64.StdEncoding.DecodeString(*imageBody)
	if err != nil {
		return fmt.Errorf("base64 decode err=%s", err.Error())
	}
	return module.OssGlobal.UploadFileByByte(*ossPath, decode)
}

// delete local file
func deleteLocalModelFile(localFile string) (bool, error) {
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

func handleError(c *gin.Context, code int, err string) {
	c.JSON(code, gin.H{"message": err})
}
