package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/models"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/module"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
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
	base64MinLen     = 2048
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
		logrus.Infof("%s | %3d | %13v | %15s | %s | %s | %s | %s",
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
			func() string {
				if model := c.Writer.Header().Get("model"); model != "" {
					return fmt.Sprintf("model=%s", model)
				} else {
					return ""
				}
			}(),
		)
	}
}

func convertImgToBase64(body []byte) ([]byte, error) {
	var request map[string]interface{}
	if err := json.Unmarshal(body, &request); err != nil {
		return body, err
	}
	parseMap(request, "", "", nil)
	if newRequest, err := json.Marshal(request); err != nil {
		return body, err
	} else {
		return newRequest, nil
	}
}

func convertBase64ToImg(body []byte, taskId, user string) ([]byte, error) {
	idx := 1
	var request map[string]interface{}
	if err := json.Unmarshal(body, &request); err == nil {
		newRequest := parseMap(request, taskId, user, &idx)
		if newBody, err := json.Marshal(newRequest); err != nil {
			return body, err
		} else {
			return newBody, nil
		}
	} else {
		var request []interface{}
		if err := json.Unmarshal(body, &request); err == nil {
			newRequest := parseArray(request, taskId, user, &idx)
			if newBody, err := json.Marshal(newRequest); err != nil {
				return body, err
			} else {
				return newBody, nil
			}
		}
	}
	return body, nil
}

func parseMap(aMap map[string]interface{}, taskId, user string, idx *int) map[string]interface{} {
	for key, val := range aMap {
		switch concreteVal := val.(type) {
		case map[string]interface{}:
			aMap[key] = parseMap(val.(map[string]interface{}), taskId, user, idx)
		case []interface{}:
			aMap[key] = parseArray(val.([]interface{}), taskId, user, idx)
		case string:
			if isImgPath(concreteVal) {
				base64, err := module.OssGlobal.DownloadFileToBase64(concreteVal)
				if err == nil {
					aMap[key] = *base64
				}
			} else if idx != nil && len(concreteVal) > base64MinLen {
				if taskId == "" {
					taskId = utils.RandStr(taskIdLength)
				}
				ossPath := fmt.Sprintf("images/%s/%s_%d.png", user, taskId, *idx)
				// check base64
				if err := uploadImages(&ossPath, &concreteVal); err == nil {
					*idx += 1
					aMap[key] = ossPath
				}
			}
		}
	}
	return aMap
}

func parseArray(anArray []interface{}, taskId, user string, idx *int) []interface{} {
	for i, val := range anArray {
		switch concreteVal := val.(type) {
		case map[string]interface{}:
			anArray[i] = parseMap(val.(map[string]interface{}), taskId, user, idx)
		case []interface{}:
			anArray[i] = parseArray(val.([]interface{}), taskId, user, idx)
		case string:
			if isImgPath(concreteVal) {
				base64, err := module.OssGlobal.DownloadFileToBase64(concreteVal)
				if err == nil {
					anArray[i] = *base64
				}
			} else if idx != nil && len(concreteVal) > base64MinLen {
				if taskId == "" {
					taskId = utils.RandStr(taskIdLength)
				}
				ossPath := fmt.Sprintf("images/%s/%s_%d.png", user, taskId, *idx)
				// check base64
				if err := uploadImages(&ossPath, &concreteVal); err == nil {
					*idx += 1
					anArray[i] = ossPath
				}

			}
		}
	}
	return anArray
}

func getFunctionDatas(store datastore.Datastore,
	request *models.BatchUpdateSdResourceRequest) (map[string]*module.FuncResource, error) {
	// get function resource
	Datas := make(map[string]*module.FuncResource)
	if request.Models == nil || len(*request.Models) == 0 {
		funcDatas, err := store.ListAll([]string{datastore.KModelServiceKey, datastore.KModelServiceFunctionName})
		if err != nil {
			return nil, err
		}
		for _, funcData := range funcDatas {
			functionName := funcData[datastore.KModelServiceFunctionName].(string)
			if resource := module.FuncManagerGlobal.GetFuncResource(functionName); resource != nil {
				funcDataNew, err := updateFuncResource(request, resource)
				if err != nil {
					return nil, err
				}
				if funcDataNew != nil {
					Datas[funcData[datastore.KModelServiceKey].(string)] = funcDataNew
				}
			}
		}
	} else {
		for _, model := range *request.Models {
			functionName := module.GetFunctionName(model)
			if resource := module.FuncManagerGlobal.GetFuncResource(functionName); resource != nil {
				funcDataNew, err := updateFuncResource(request, resource)
				if err != nil {
					return nil, err
				}
				if funcDataNew != nil {
					Datas[model] = funcDataNew
				}
			}
		}
	}
	return Datas, nil
}

func updateFuncResource(request *models.BatchUpdateSdResourceRequest,
	res *module.FuncResource) (*module.FuncResource, error) {
	isDiff := false
	// update resource
	// image
	if request.Image != nil && *request.Image != "" && *request.Image != res.Image {
		res.Image = *request.Image
		isDiff = true
	}
	// cpu
	if request.Cpu != nil && *request.Cpu > 0 && *request.Cpu != res.CPU {
		res.CPU = *request.Cpu
		isDiff = true
	}
	// env
	if request.Env != nil {
		if res.Env == nil {
			res.Env = make(map[string]*string)
		}
		for key, val := range *request.Env {
			res.Env[key] = utils.String(val.(string))
		}
		isDiff = true
	}
	// extraArgs
	if request.ExtraArgs != nil && *request.ExtraArgs != "" && *request.ExtraArgs != *res.Env["EXTRA_ARGS"] {
		res.Env["EXTRA_ARGS"] = request.ExtraArgs
		isDiff = true
	}
	// gpuMemorySize
	if request.GpuMemorySize != nil && *request.GpuMemorySize > 0 && *request.GpuMemorySize != res.GpuMemorySize {
		res.GpuMemorySize = *request.GpuMemorySize
		isDiff = true
	}
	// MemorySize
	if request.MemorySize != nil && *request.MemorySize > 0 && *request.MemorySize != res.MemorySize {
		res.MemorySize = *request.MemorySize
		isDiff = true
	}
	// instanceType
	if request.InstanceType != nil && *request.InstanceType != "" && *request.InstanceType != res.InstanceType {
		res.InstanceType = *request.InstanceType
		isDiff = true
	}
	// timeout
	if request.Timeout != nil && *request.Timeout > 0 && *request.Timeout != res.Timeout {
		res.Timeout = *request.Timeout
		isDiff = true
	}
	if request.VpcConfig != nil {
		res.VpcConfig = request.VpcConfig
		isDiff = true
	}
	if request.NasConfig != nil {
		res.NasConfig = request.NasConfig
		isDiff = true
	}
	if request.OssMountConfig != nil {
		res.OssMountConfig = request.OssMountConfig
		isDiff = true
	}
	if isDiff {
		return res, nil
	} else {
		return nil, nil
	}
}

// check stable_diffusion_model val not "" or nil
func checkSdModelValid(sdModel string) bool {
	return sdModel != ""
}

// extra ossUrl
func extraOssUrl(resp *http.Response) *[]string {
	in, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		logrus.Warn("get oss url error")
		return nil
	}
	var m models.SubmitTaskResponse
	if err := json.Unmarshal(in, &m); err != nil {
		return nil
	} else {
		return m.OssUrl
	}
}

// extra err message
func extraErrorMsg(resp *http.Response) *string {
	in, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		logrus.Warn("extra resp error")
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(in, &m); err != nil {
		return nil
	} else {
		if val, ok := m["message"]; ok {
			msg := val.(string)
			return &msg
		}
	}
	return utils.String(string(in))
}

func handleRespError(c *gin.Context, err error, resp *http.Response, taskId string) {
	msg := ""
	if err != nil {
		msg = err.Error()
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf("%v", err)
	} else {
		if v := extraErrorMsg(resp); v != nil {
			msg = *v
		} else {
			msg = config.INTERNALERROR
		}
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf("%v", resp)
	}
	c.JSON(http.StatusInternalServerError, models.SubmitTaskResponse{
		TaskId:  taskId,
		Status:  config.TASK_FAILED,
		Message: utils.String(msg),
	})
}
