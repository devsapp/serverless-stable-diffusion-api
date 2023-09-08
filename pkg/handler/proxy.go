package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/client"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/models"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/module"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strings"
)

const DEFAULT_USER = "default"

type ProxyHandler struct {
	userStore   datastore.Datastore
	taskStore   datastore.Datastore
	modelStore  datastore.Datastore
	configStore datastore.Datastore
}

func NewProxyHandler(taskStore datastore.Datastore,
	modelStore datastore.Datastore, userStore datastore.Datastore,
	configStore datastore.Datastore) *ProxyHandler {
	return &ProxyHandler{
		taskStore:   taskStore,
		modelStore:  modelStore,
		userStore:   userStore,
		configStore: configStore,
	}
}

// Login user login
// (POST /login)
func (p *ProxyHandler) Login(c *gin.Context) {
	request := new(models.UserLoginRequest)
	if err := getBindResult(c, request); err != nil {
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}
	token, expired, ok := module.UserManagerGlobal.VerifyUserValid(request.UserName, request.Password)
	if !ok {
		c.JSON(http.StatusGone, models.UserLoginResponse{
			Message: utils.String("login fail"),
		})
	} else {
		// update db
		p.userStore.Update(request.UserName, map[string]interface{}{
			datastore.KUserSession:          token,
			datastore.KUserSessionValidTime: fmt.Sprintf("%d", expired),
			datastore.KUserModifyTime:       fmt.Sprintf("%d", utils.TimestampS()),
		})
		c.JSON(http.StatusOK, models.UserLoginResponse{
			UserName: request.UserName,
			Token:    token,
			Message:  utils.String("login success"),
		})
	}

}

// CancelTask predict task
// (POST /tasks/{taskId}/cancellation)
func (p *ProxyHandler) CancelTask(c *gin.Context, taskId string) {
	if err := p.taskStore.Update(taskId, map[string]interface{}{
		datastore.KTaskCancel: int64(config.CANCEL_VALID),
	}); err != nil {
		handleError(c, http.StatusInternalServerError, config.INTERNALERROR)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// GetTaskResult GetTaskResult get predict progress
// (GET /tasks/{taskId}/result)
func (p *ProxyHandler) GetTaskResult(c *gin.Context, taskId string) {
	result, err := p.getTaskResult(taskId)
	if err != nil {
		handleError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

// ListModels list model
// (GET /models)
func (p *ProxyHandler) ListModels(c *gin.Context) {
	val, err := p.modelStore.ListAll([]string{datastore.KModelType, datastore.KModelName,
		datastore.KModelOssPath, datastore.KModelEtag, datastore.KModelStatus, datastore.KModelCreateTime,
		datastore.KModelModifyTime})
	if err != nil {
		handleError(c, http.StatusInternalServerError, config.INTERNALERROR)
		return
	}
	c.JSON(http.StatusOK, convertToModelResponse(val))

}

// RegisterModel upload model
// (POST /models)
func (p *ProxyHandler) RegisterModel(c *gin.Context) {
	request := new(models.RegisterModelJSONRequestBody)
	if err := getBindResult(c, request); err != nil {
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}
	// check models exist or not
	data, err := p.modelStore.Get(request.Name, []string{datastore.KModelName,
		datastore.KModelEtag, datastore.KModelOssPath, datastore.KModelStatus})
	if err != nil {
		handleError(c, http.StatusInternalServerError, "read models db error")
		return
	}

	// models existed
	if data != nil && len(data) != 0 && data[datastore.KModelStatus].(string) != config.MODEL_DELETE && data[datastore.KModelEtag].(string) == request.Etag &&
		data[datastore.KModelOssPath].(string) == request.OssPath {
		c.JSON(http.StatusOK, gin.H{"message": "models existed"})
		return
	}
	// from oss download model to local
	localFile, err := downloadModelsFromOss(request.Type, request.OssPath, request.Name)
	if err != nil {
		handleError(c, http.StatusInternalServerError, fmt.Sprintf("please check oss model valid, "+
			"err=%s", err.Error()))
		return
	}

	// update db
	data = map[string]interface{}{
		datastore.KModelType:       request.Type,
		datastore.KModelName:       request.Name,
		datastore.KModelOssPath:    request.OssPath,
		datastore.KModelEtag:       request.Etag,
		datastore.KModelLocalPath:  localFile,
		datastore.KModelStatus:     getModelsStatus(request.Type),
		datastore.KModelCreateTime: fmt.Sprintf("%d", utils.TimestampS()),
		datastore.KModelModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
	}
	p.modelStore.Put(request.Name, data)
	c.JSON(http.StatusOK, gin.H{"message": "register success"})
}

// DeleteModel delete model
// (DELETE /models/{model_name})
func (p *ProxyHandler) DeleteModel(c *gin.Context, modelName string) {
	// get local file path
	data, err := p.modelStore.Get(modelName, []string{datastore.KModelLocalPath, datastore.KModelStatus})
	if err != nil {
		handleError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if data == nil || len(data) == 0 || data[datastore.KModelStatus] == config.MODEL_DELETE {
		handleError(c, http.StatusInternalServerError, "model not exist")
		return
	}
	localFile := data[datastore.KModelLocalPath].(string)
	// delete nas models
	if ok, err := utils.DeleteLocalModelFile(localFile); !ok {
		handleError(c, http.StatusInternalServerError, err.Error())
		return
	}
	// model status set deleted
	if err := p.modelStore.Update(modelName, map[string]interface{}{
		datastore.KModelStatus:     config.MODEL_DELETE,
		datastore.KModelModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
	}); err != nil {
		handleError(c, http.StatusInternalServerError, config.INTERNALERROR)
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "delete success"})
	}
}

// GetModel get model info
// (GET /models/{model_name})
func (p *ProxyHandler) GetModel(c *gin.Context, modelName string) {
	data, err := p.modelStore.Get(modelName, []string{datastore.KModelType, datastore.KModelName,
		datastore.KModelOssPath, datastore.KModelEtag, datastore.KModelStatus, datastore.KModelCreateTime,
		datastore.KModelModifyTime})
	if err != nil {
		handleError(c, http.StatusInternalServerError, config.INTERNALERROR)
		return
	}
	if data == nil || len(data) == 0 {
		handleError(c, http.StatusNotFound, config.NOTFOUND)
		return
	}
	c.JSON(http.StatusOK, convertToModelResponse(map[string]map[string]interface{}{
		modelName: data,
	}))

}

// UpdateModel update model
// (PUT /models/{model_name})
func (p *ProxyHandler) UpdateModel(c *gin.Context, modelName string) {
	request := new(models.UpdateModelJSONRequestBody)
	if err := getBindResult(c, request); err != nil {
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}
	// check models exist or not
	data, err := p.modelStore.Get(modelName, []string{datastore.KModelName,
		datastore.KModelEtag, datastore.KModelOssPath, datastore.KModelStatus})
	if err != nil {
		handleError(c, http.StatusInternalServerError, "read models db error")
		return
	}
	// models existed and not change
	if data != nil {
		if data[datastore.KModelStatus].(string) == config.MODEL_DELETE {
			handleError(c, http.StatusNotFound, "model not register, please register first")
			return
		} else if data[datastore.KModelEtag].(string) == request.Etag &&
			data[datastore.KModelOssPath].(string) == request.OssPath {
			c.JSON(http.StatusOK, gin.H{"message": "models existed and not change"})
			return
		}
	} else {
		handleError(c, http.StatusNotFound, "model not register, please register first")
		return
	}
	// from oss download nas
	if _, err := downloadModelsFromOss(request.Type, request.OssPath, request.Name); err != nil {
		handleError(c, http.StatusInternalServerError, fmt.Sprintf("please check oss model valid, "+
			"err=%s", err.Error()))
		return
	}
	// sdModel and sdVae enable env update
	if request.Type == config.SD_MODEL || request.Type == config.SD_VAE {
		if err := module.FuncManagerGlobal.UpdateFunctionEnv(request.Name); err != nil {
			handleError(c, http.StatusInternalServerError, config.MODELUPDATEFCERROR)
			return
		}
	}

	// update db
	data = map[string]interface{}{
		datastore.KModelType:       request.Type,
		datastore.KModelOssPath:    request.OssPath,
		datastore.KModelEtag:       request.Etag,
		datastore.KModelStatus:     getModelsStatus(request.Type),
		datastore.KModelModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
	}
	if err := p.modelStore.Update(modelName, data); err != nil {
		handleError(c, http.StatusInternalServerError, config.NOTFOUND)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success"})

}

// GetTaskProgress get predict progress
// (GET /tasks/{taskId}/progress)
func (p *ProxyHandler) GetTaskProgress(c *gin.Context, taskId string) {
	data, err := p.taskStore.Get(taskId, []string{datastore.KTaskProgressColumnName})
	if err != nil || data == nil || len(data) == 0 {
		handleError(c, http.StatusNotFound, config.NOTFOUND)
		return
	}
	resp := new(models.TaskProgressResponse)
	if err := json.Unmarshal([]byte(data[datastore.KTaskProgressColumnName].(string)), resp); err != nil {
		handleError(c, http.StatusInternalServerError, config.NOTFOUND)
		return
	}
	resp.TaskId = taskId
	c.JSON(http.StatusOK, resp)

}

// Txt2Img txt to img predict
// (POST /txt2img)
func (p *ProxyHandler) Txt2Img(c *gin.Context) {
	username := c.GetHeader(userKey)
	invokeType := c.GetHeader(requestType)
	if username == "" {
		if config.ConfigGlobal.EnableLogin() {
			handleError(c, http.StatusBadRequest, config.BADREQUEST)
			return
		} else {
			username = DEFAULT_USER
		}
	}
	request := new(models.Txt2ImgJSONRequestBody)
	if err := getBindResult(c, request); err != nil {
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}
	// init taskId
	taskId := utils.RandStr(taskIdLength)
	log.Println("taskid:", taskId)
	// check request valid: sdModel and sdVae exist
	if existed := p.checkModelExist(request.StableDiffusionModel, request.SdVae); !existed {
		handleError(c, http.StatusNotFound, "model not found, please check request")
		return
	}
	// write db
	if err := p.taskStore.Put(taskId, map[string]interface{}{
		datastore.KTaskIdColumnName: taskId,
		datastore.KTaskUser:         username,
		datastore.KTaskCancel:       int64(config.CANCEL_INIT),
		datastore.KTaskCreateTime:   fmt.Sprintf("%d", utils.TimestampS()),
	}); err != nil {
		log.Println("[Error] put db err=", err.Error())
		c.JSON(http.StatusInternalServerError, models.SubmitTaskResponse{
			TaskId:  taskId,
			Status:  config.TASK_FAILED,
			Message: utils.String(config.INTERNALERROR),
		})
		return
	}

	// get endPoint
	sdModel := request.StableDiffusionModel
	sdVae := request.SdVae
	endPoint, err := module.FuncManagerGlobal.GetEndpoint(sdModel, sdVae)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.SubmitTaskResponse{
			TaskId:  taskId,
			Status:  config.TASK_FAILED,
			Message: utils.String(config.NOFOUNDENDPOINT),
		})
		return
	}

	// get user current config version
	userItem, err := p.userStore.Get(username, []string{datastore.KUserConfigVer})
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.SubmitTaskResponse{
			TaskId:  taskId,
			Status:  config.TASK_FAILED,
			Message: utils.String(config.INTERNALERROR),
		})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), config.HTTPTIMEOUT)
	defer cancel()
	// get client by endPoint
	client := client.ManagerClientGlobal.GetClient(endPoint)
	// async request
	resp, err := client.Txt2Img(ctx, *request, func(ctx context.Context, req *http.Request) error {
		req.Header.Add(userKey, username)
		req.Header.Add(taskKey, taskId)
		req.Header.Add(versionKey, func() string {
			if version, ok := userItem[datastore.KUserConfigVer]; !ok {
				return "-1"
			} else {
				return version.(string)
			}
		}())
		if isAsync(invokeType) {
			req.Header.Add(FcAsyncKey, "Async")
		}
		return nil
	})
	if err != nil || (resp.StatusCode != syncSuccessCode && resp.StatusCode != asyncSuccessCode) {
		if err != nil {
			log.Println(err.Error())
		} else {
			log.Println(resp)
		}
		c.JSON(http.StatusInternalServerError, models.SubmitTaskResponse{
			TaskId:  taskId,
			Status:  config.TASK_FAILED,
			Message: utils.String(config.INTERNALERROR),
		})
	} else {
		c.JSON(http.StatusOK, models.SubmitTaskResponse{
			TaskId: taskId,
			Status: func() string {
				if resp.StatusCode == syncSuccessCode {
					return config.TASK_FINISH
				}
				if resp.StatusCode == asyncSuccessCode {
					return config.TASK_QUEUE
				}
				return config.TASK_FAILED
			}(),
		})
	}
}

// Img2Img img to img predict
// (POST /img2img)
func (p *ProxyHandler) Img2Img(c *gin.Context) {
	username := c.GetHeader(userKey)
	invokeType := c.GetHeader(requestType)
	if username == "" {
		if config.ConfigGlobal.EnableLogin() {
			handleError(c, http.StatusBadRequest, config.BADREQUEST)
			return
		} else {
			username = DEFAULT_USER
		}
	}
	request := new(models.Img2ImgJSONRequestBody)
	if err := getBindResult(c, request); err != nil {
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}
	// init taskId
	taskId := utils.RandStr(taskIdLength)

	// check request valid: sdModel and sdVae exist
	if existed := p.checkModelExist(request.StableDiffusionModel, request.SdVae); !existed {
		handleError(c, http.StatusNotFound, "model not found, please check request")
		return
	}

	// write db
	if err := p.taskStore.Put(taskId, map[string]interface{}{
		datastore.KTaskIdColumnName: taskId,
		datastore.KTaskUser:         username,
		datastore.KTaskCancel:       int64(config.CANCEL_INIT),
		datastore.KTaskCreateTime:   fmt.Sprintf("%d", utils.TimestampS()),
	}); err != nil {
		log.Println("[Error] put db err=", err.Error())
		c.JSON(http.StatusInternalServerError, models.SubmitTaskResponse{
			TaskId:  taskId,
			Status:  config.TASK_FAILED,
			Message: utils.String(config.INTERNALERROR),
		})
		return
	}

	// get endPoint
	sdModel := request.StableDiffusionModel
	sdVae := request.SdVae
	endPoint, err := module.FuncManagerGlobal.GetEndpoint(sdModel, sdVae)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.SubmitTaskResponse{
			TaskId:  taskId,
			Status:  config.TASK_FAILED,
			Message: utils.String(config.NOFOUNDENDPOINT),
		})
		return
	}

	// get user current config version
	userItem, err := p.userStore.Get(username, []string{datastore.KUserConfigVer})
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.SubmitTaskResponse{
			TaskId:  taskId,
			Status:  config.TASK_FAILED,
			Message: utils.String(config.INTERNALERROR),
		})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), config.HTTPTIMEOUT)
	defer cancel()
	// get client by endPoint
	client := client.ManagerClientGlobal.GetClient(endPoint)
	// async request
	resp, err := client.Img2Img(ctx, *request, func(ctx context.Context, req *http.Request) error {
		req.Header.Add(userKey, username)
		req.Header.Add(taskKey, taskId)
		req.Header.Add(versionKey, func() string {
			if version, ok := userItem[datastore.KUserConfigVer]; !ok {
				return "-1"
			} else {
				return version.(string)
			}
		}())
		if isAsync(invokeType) {
			req.Header.Add(FcAsyncKey, "Async")
		}
		return nil
	})
	if err != nil || (resp.StatusCode != syncSuccessCode && resp.StatusCode != asyncSuccessCode) {
		c.JSON(http.StatusInternalServerError, models.SubmitTaskResponse{
			TaskId:  taskId,
			Status:  config.TASK_FAILED,
			Message: utils.String(config.INTERNALERROR),
		})
	} else {
		c.JSON(http.StatusOK, models.SubmitTaskResponse{
			TaskId: taskId,
			Status: func() string {
				if resp.StatusCode == syncSuccessCode {
					return config.TASK_FINISH
				}
				if resp.StatusCode == asyncSuccessCode {
					return config.TASK_QUEUE
				}
				return config.TASK_FAILED
			}(),
		})
	}
}

// UpdateOptions update config options
// (POST /options)
func (p *ProxyHandler) UpdateOptions(c *gin.Context) {
	username := c.GetHeader(userKey)
	if username == "" {
		if config.ConfigGlobal.EnableLogin() {
			handleError(c, http.StatusBadRequest, config.BADREQUEST)
			return
		} else {
			username = DEFAULT_USER
		}
	}
	request := new(models.OptionRequest)
	if err := getBindResult(c, request); err != nil {
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}
	configStr, err := json.Marshal(request.Data)
	if err != nil {
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}
	version := fmt.Sprintf("%d", utils.TimestampS())
	key := fmt.Sprintf("%s_%s", username, version)
	if err := p.configStore.Put(key, map[string]interface{}{
		datastore.KConfigVal:      string(configStr),
		datastore.KConfigVer:      version,
		datastore.KUserModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
	}); err != nil {
		handleError(c, http.StatusInternalServerError, "update db error")
		return
	}
	if err := p.userStore.Update(username, map[string]interface{}{
		datastore.KUserConfigVer:  version,
		datastore.KUserModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
	}); err != nil {
		if !config.ConfigGlobal.EnableLogin() {
			// if username not existed add user
			if err = p.userStore.Put(username, map[string]interface{}{
				datastore.KUserConfigVer:  version,
				datastore.KUserCreateTime: fmt.Sprintf("%d", utils.TimestampS()),
				datastore.KUserModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
			}); err == nil {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
				return
			}

		}
		handleError(c, http.StatusInternalServerError, "update db error")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

func (p *ProxyHandler) getTaskResult(taskId string) (*models.TaskResultResponse, error) {
	result := &models.TaskResultResponse{
		TaskId:     taskId,
		Parameters: new(map[string]interface{}),
		Info:       new(map[string]interface{}),
	}
	data, err := p.taskStore.Get(taskId, []string{datastore.KTaskImage, datastore.KTaskInfo,
		datastore.KTaskParams, datastore.KTaskCode})
	if err != nil || data == nil || len(data) == 0 {
		return nil, errors.New("not found")
	}
	if code, ok := data[datastore.KTaskCode]; ok && code.(int64) != requestOk {
		return nil, fmt.Errorf("task:%s predict fail", taskId)
	} else if !ok {
		return nil, fmt.Errorf("task:%s predict fail", taskId)
	}

	// images
	result.Images = strings.Split(data[datastore.KTaskImage].(string), ",")
	// params
	paramsStr := data[datastore.KTaskParams].(string)
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(paramsStr), &m); err != nil {
		log.Println("[getTaskResult] Unmarshal params error=", err.Error())
	}
	*result.Parameters = m
	// info
	infoStr := data[datastore.KTaskInfo].(string)
	if err := json.Unmarshal([]byte(infoStr), &m); err != nil {
		log.Println("[getTaskResult] Unmarshal Info error=", err.Error())
	}
	*result.Info = m
	return result, nil
}

func (p *ProxyHandler) checkModelExist(sdModel, sdVae string) bool {
	models := []string{sdModel}
	// check local existed
	sdModelPath := fmt.Sprintf("%s/models/%s/%s", config.ConfigGlobal.SdPath, "Stable-diffusion", sdModel)
	sdVaePath := fmt.Sprintf("%s/models/%s/%s", config.ConfigGlobal.SdPath, "VAE", sdVae)
	if utils.FileExists(sdModelPath) && (sdVae == "None" || sdVae == "Automatic" || utils.FileExists(sdVaePath)) {
		return true
	}
	// check remote db existed
	// remove sdVae = None || Automatic
	if sdVae != "None" && sdVae != "Automatic" {
		models = append(models, sdVae)
	}
	for _, model := range models {
		// check sdModel
		if data, err := p.modelStore.Get(model, []string{datastore.KModelStatus}); err != nil || data == nil || len(data) == 0 {
			return false
		} else if data[datastore.KModelStatus] == config.MODEL_DELETE {
			return false
		}
	}
	return true
}

func convertToModelResponse(datas map[string]map[string]interface{}) []models.ModelAttributes {
	ret := make([]models.ModelAttributes, 0, len(datas))
	for _, data := range datas {
		registeredTime := data[datastore.KModelCreateTime].(string)
		modifyTime := data[datastore.KModelModifyTime].(string)
		ret = append(ret, models.ModelAttributes{
			Type:                 data[datastore.KModelType].(string),
			Name:                 data[datastore.KModelName].(string),
			OssPath:              data[datastore.KModelOssPath].(string),
			Etag:                 data[datastore.KModelEtag].(string),
			Status:               data[datastore.KModelStatus].(string),
			RegisteredTime:       &registeredTime,
			LastModificationTime: &modifyTime,
		})
	}
	return ret
}

func getModelsStatus(modelType string) string {
	switch modelType {
	case config.SD_MODEL, config.SD_VAE:
		return config.MODEL_LOADED
	default:
		return config.MODEL_LOADED
	}
}

func ApiAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path != "/login" {
			tokenString := c.Request.Header.Get("Token")
			userName, ok := module.UserManagerGlobal.VerifySessionValid(tokenString)
			if !ok {
				c.JSON(http.StatusGone, gin.H{"message": "please login first or login expired"})
				c.Abort()
			}
			c.Request.Header.Set("userName", userName)
		}
	}
}

func isAsync(invokeType string) bool {
	if invokeType == "sync" {
		return false
	}
	return true
}
