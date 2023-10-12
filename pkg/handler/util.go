package handler

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/models"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/module"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

const (
	taskIdLength     = 10
	userKey          = "username"
	requestType      = "Request-Type"
	taskKey          = "taskId"
	FcAsyncKey       = "X-Fc-Invocation-Type"
	versionKey       = "version"
	requestOk        = 200
	requestFail      = 422
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

func isImgPath(str string) bool {
	return strings.HasSuffix(str, ".png") || strings.HasSuffix(str, ".jpg") ||
		strings.HasSuffix(str, ".jpeg")
}

func listModelFile(path, modelType string) (modelAttrs []*models.ModelAttributes) {
	files := utils.ListFile(path)
	for _, name := range files {
		if strings.HasSuffix(name, ".pt") || strings.HasSuffix(name, ".ckpt") ||
			strings.HasSuffix(name, ".safetensors") || strings.HasSuffix(name, ".pth") {
			modelAttrs = append(modelAttrs, &models.ModelAttributes{
				Type:   modelType,
				Name:   name,
				Status: config.MODEL_LOADED,
			})
		}
	}
	return
}

// Stat cost code
func Stat() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next()
		endTime := time.Now()
		latencyTime := endTime.Sub(startTime)
		reqMethod := c.Request.Method
		reqUri := c.Request.RequestURI
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		logrus.Infof("%s | %3d | %13v | %15s | %s | %s | %s",
			config.ConfigGlobal.ServerName,
			statusCode,
			latencyTime,
			clientIP,
			reqMethod,
			reqUri,
			func() string {
				if taskId := c.Writer.Header().Get("taskId"); taskId != "" {
					return fmt.Sprintf("taskId=%s", taskId)
				} else {
					return ""
				}
			}(),
		)
	}
}
