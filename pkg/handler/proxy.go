package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/client"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/concurrency"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/models"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/module"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
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

// Restart restart webui api server
// (POST /restart)
func (p *ProxyHandler) Restart(c *gin.Context) {
	if config.ConfigGlobal.IsServerTypeMatch(config.PROXY) {
		//retransmission to control
		target := config.ConfigGlobal.Downstream
		remote, err := url.Parse(target)
		if err != nil {
			panic(err)
		}
		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.Director = func(req *http.Request) {
			req.Header = c.Request.Header
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	} else if config.ConfigGlobal.IsServerTypeMatch(config.CONTROL) {
		// update agent env
		err := module.FuncManagerGlobal.UpdateAllFunctionEnv()
		if err != nil {
			handleError(c, http.StatusInternalServerError, "update function env error")
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	} else {
		c.JSON(http.StatusNotFound, gin.H{"message": "not support"})
	}
}

// CancelTask predict task
// (POST /tasks/{taskId}/cancellation)
func (p *ProxyHandler) CancelTask(c *gin.Context, taskId string) {
	if err := p.taskStore.Update(taskId, map[string]interface{}{
		datastore.KTaskCancel: int64(config.CANCEL_VALID),
	}); err != nil {
		handleError(c, http.StatusInternalServerError, "update task cancel error")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// GetTaskResult  get predict progress
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
	if config.ConfigGlobal.UseLocalModel() {
		// get from local disk
		ret := make([]*models.ModelAttributes, 0)
		// sdModel
		path := fmt.Sprintf("%s/models/%s", config.ConfigGlobal.SdPath, "Stable-diffusion")
		ret = append(ret, listModelFile(path, config.SD_MODEL)...)
		// sdVae
		path = fmt.Sprintf("%s/models/%s", config.ConfigGlobal.SdPath, "VAE")
		ret = append(ret, listModelFile(path, config.SD_VAE)...)
		// lora
		path = fmt.Sprintf("%s/models/%s", config.ConfigGlobal.SdPath, "Lora")
		ret = append(ret, listModelFile(path, config.LORA_MODEL)...)
		// controlNet
		path = fmt.Sprintf("%s/models/%s", config.ConfigGlobal.SdPath, "ControlNet")
		ret = append(ret, listModelFile(path, config.CONTORLNET_MODEL)...)
		c.JSON(http.StatusOK, ret)
	} else {
		// get from db
		val, err := p.modelStore.ListAll([]string{datastore.KModelType, datastore.KModelName,
			datastore.KModelOssPath, datastore.KModelEtag, datastore.KModelStatus, datastore.KModelCreateTime,
			datastore.KModelModifyTime})
		if err != nil {
			handleError(c, http.StatusInternalServerError, "read model from db error")
			return
		}
		c.JSON(http.StatusOK, convertToModelResponse(val))
	}

}

// RegisterModel upload model
// (POST /models)
func (p *ProxyHandler) RegisterModel(c *gin.Context) {
	if config.ConfigGlobal.UseLocalModel() {
		c.String(http.StatusNotFound, "useLocalModel=yes not support")
		return
	}
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
	if config.ConfigGlobal.UseLocalModel() {
		c.String(http.StatusNotFound, "useLocalModel=yes not support")
		return
	}
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
	if ok, err := utils.DeleteLocalFile(localFile); !ok {
		handleError(c, http.StatusInternalServerError, err.Error())
		return
	}
	// model status set deleted
	if err := p.modelStore.Update(modelName, map[string]interface{}{
		datastore.KModelStatus:     config.MODEL_DELETE,
		datastore.KModelModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
	}); err != nil {
		handleError(c, http.StatusInternalServerError, "update model status error")
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "delete success"})
	}
}

// GetModel get model info
// (GET /models/{model_name})
func (p *ProxyHandler) GetModel(c *gin.Context, modelName string) {
	if config.ConfigGlobal.UseLocalModel() {
		c.String(http.StatusNotFound, "useLocalModel=yes not support")
		return
	}
	data, err := p.modelStore.Get(modelName, []string{datastore.KModelType, datastore.KModelName,
		datastore.KModelOssPath, datastore.KModelEtag, datastore.KModelStatus, datastore.KModelCreateTime,
		datastore.KModelModifyTime})
	if err != nil {
		handleError(c, http.StatusInternalServerError, "get model info from db error")
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
	if config.ConfigGlobal.UseLocalModel() {
		c.String(http.StatusNotFound, "useLocalModel=yes not support")
		return
	}
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
		if err := module.FuncManagerGlobal.UpdateFunctionEnv(request.Name, request.Name); err != nil {
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
	data, err := p.taskStore.Get(taskId, []string{datastore.KTaskIdColumnName, datastore.KTaskStatus,
		datastore.KTaskProgressColumnName})
	if err != nil || data == nil || len(data) == 0 {
		handleError(c, http.StatusNotFound, config.NOTFOUND)
		return
	}
	resp := new(models.TaskProgressResponse)
	if progress, ok := data[datastore.KTaskProgressColumnName]; ok {
		if err := json.Unmarshal([]byte(progress.(string)), resp); err != nil {
			handleError(c, http.StatusInternalServerError, config.NOTFOUND)
			return
		}
	}
	if status, ok := data[datastore.KTaskStatus]; ok && (status == config.TASK_FINISH || status == config.TASK_FAILED) {
		resp.Progress = 1
	}
	resp.TaskId = taskId
	c.JSON(http.StatusOK, resp)
}

// ExtraImages image upcaling
// (POST /extra_images)
func (p *ProxyHandler) ExtraImages(c *gin.Context) {
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
	request := new(models.ExtraImagesJSONRequestBody)
	if err := getBindResult(c, request); err != nil {
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}
	// taskId
	taskId := c.GetHeader(taskKey)
	if taskId == "" {
		// init taskId
		taskId = utils.RandStr(taskIdLength)
	}
	c.Writer.Header().Set("taskId", taskId)

	endPoint := config.ConfigGlobal.Downstream
	var err error
	if config.ConfigGlobal.IsServerTypeMatch(config.CONTROL) {
		if endPoint = module.FuncManagerGlobal.GetLastInvokeEndpoint(request.StableDiffusionModel); endPoint == "" {
			handleError(c, http.StatusInternalServerError, "not found valid endpoint")
			return
		}
	}

	// write db
	if err := p.taskStore.Put(taskId, map[string]interface{}{
		datastore.KTaskIdColumnName: taskId,
		datastore.KTaskUser:         username,
		datastore.KTaskStatus:       config.TASK_QUEUE,
		datastore.KTaskCreateTime:   fmt.Sprintf("%d", utils.TimestampS()),
	}); err != nil {
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf("put db err=%s", err.Error())
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
	resp, err := client.ExtraImages(ctx, *request, func(ctx context.Context, req *http.Request) error {
		req.Header.Add(userKey, username)
		req.Header.Add(taskKey, taskId)
		if isAsync(invokeType) {
			req.Header.Add(FcAsyncKey, "Async")
		}
		return nil
	})
	if err != nil || (resp.StatusCode != syncSuccessCode && resp.StatusCode != asyncSuccessCode) {
		if err != nil {
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Error(err.Error())
		} else {
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf("%v", resp)
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
	// taskId
	taskId := c.GetHeader(taskKey)
	if taskId == "" {
		// init taskId
		taskId = utils.RandStr(taskIdLength)
	}
	c.Writer.Header().Set("taskId", taskId)

	endPoint := config.ConfigGlobal.Downstream
	var err error
	version := c.GetHeader(versionKey)
	if config.ConfigGlobal.IsServerTypeMatch(config.CONTROL) {
		// get endPoint
		sdModel := request.StableDiffusionModel
		c.Writer.Header().Set("model", sdModel)
		// wait to valid
		if concurrency.ConCurrencyGlobal.WaitToValid(sdModel) {
			// cold start
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Infof("sd %s cold start ....", sdModel)
			defer concurrency.ConCurrencyGlobal.DecColdNum(sdModel, taskId)
		}
		defer concurrency.ConCurrencyGlobal.DoneTask(sdModel, taskId)
		endPoint, err = module.FuncManagerGlobal.GetEndpoint(sdModel)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.SubmitTaskResponse{
				TaskId:  taskId,
				Status:  config.TASK_FAILED,
				Message: utils.String(config.NOFOUNDENDPOINT),
			})
			return
		}
	}
	if config.ConfigGlobal.IsServerTypeMatch(config.PROXY) {
		// check request valid: sdModel and sdVae exist
		if existed := p.checkModelExist(request.StableDiffusionModel, request.SdVae); !existed {
			handleError(c, http.StatusNotFound, "model not found, please check request")
			return
		}
		// write db
		if err := p.taskStore.Put(taskId, map[string]interface{}{
			datastore.KTaskIdColumnName: taskId,
			datastore.KTaskUser:         username,
			datastore.KTaskStatus:       config.TASK_QUEUE,
			datastore.KTaskCancel:       int64(config.CANCEL_INIT),
			datastore.KTaskCreateTime:   fmt.Sprintf("%d", utils.TimestampS()),
		}); err != nil {
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf("put db err=%s", err.Error())
			c.JSON(http.StatusInternalServerError, models.SubmitTaskResponse{
				TaskId:  taskId,
				Status:  config.TASK_FAILED,
				Message: utils.String(config.INTERNALERROR),
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
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf("get config version err=%s", err.Error())
			return
		}
		version = func() string {
			if version, ok := userItem[datastore.KUserConfigVer]; !ok {
				return "-1"
			} else {
				return version.(string)
			}
		}()
	}
	ctx, cancel := context.WithTimeout(context.Background(), config.HTTPTIMEOUT)
	defer cancel()
	// get client by endPoint
	client := client.ManagerClientGlobal.GetClient(endPoint)
	// async request
	resp, err := client.Txt2Img(ctx, *request, func(ctx context.Context, req *http.Request) error {
		req.Header.Add(userKey, username)
		req.Header.Add(taskKey, taskId)
		req.Header.Add(versionKey, version)
		if isAsync(invokeType) {
			req.Header.Add(FcAsyncKey, "Async")
		}
		return nil
	})
	if err != nil || (resp.StatusCode != syncSuccessCode && resp.StatusCode != asyncSuccessCode) {
		if err != nil {
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Error(err.Error())
		} else {
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf("%v", resp)
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
	// taskId
	taskId := c.GetHeader(taskKey)
	if taskId == "" {
		// init taskId
		taskId = utils.RandStr(taskIdLength)
	}
	c.Writer.Header().Set("taskId", taskId)

	endPoint := config.ConfigGlobal.Downstream
	var err error
	version := c.GetHeader(versionKey)
	if config.ConfigGlobal.IsServerTypeMatch(config.CONTROL) {
		// get endPoint
		sdModel := request.StableDiffusionModel
		c.Writer.Header().Set("model", sdModel)
		// wait to valid
		if concurrency.ConCurrencyGlobal.WaitToValid(sdModel) {
			// cold start
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Infof("sd %s cold start ....", sdModel)
			defer concurrency.ConCurrencyGlobal.DecColdNum(sdModel, taskId)
		}
		defer concurrency.ConCurrencyGlobal.DoneTask(sdModel, taskId)
		endPoint, err = module.FuncManagerGlobal.GetEndpoint(sdModel)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.SubmitTaskResponse{
				TaskId:  taskId,
				Status:  config.TASK_FAILED,
				Message: utils.String(config.NOFOUNDENDPOINT),
			})
			return
		}
	}
	if config.ConfigGlobal.IsServerTypeMatch(config.PROXY) {
		// check request valid: sdModel and sdVae exist
		if existed := p.checkModelExist(request.StableDiffusionModel, request.SdVae); !existed {
			handleError(c, http.StatusNotFound, "model not found, please check request")
			return
		}
		// write db
		if err := p.taskStore.Put(taskId, map[string]interface{}{
			datastore.KTaskIdColumnName: taskId,
			datastore.KTaskUser:         username,
			datastore.KTaskStatus:       config.TASK_QUEUE,
			datastore.KTaskCancel:       int64(config.CANCEL_INIT),
			datastore.KTaskCreateTime:   fmt.Sprintf("%d", utils.TimestampS()),
		}); err != nil {
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Error("[Error] put db err=", err.Error())
			c.JSON(http.StatusInternalServerError, models.SubmitTaskResponse{
				TaskId:  taskId,
				Status:  config.TASK_FAILED,
				Message: utils.String(config.INTERNALERROR),
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
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Error("get config version err=", err.Error())
			return
		}
		version = func() string {
			if version, ok := userItem[datastore.KUserConfigVer]; !ok {
				return "-1"
			} else {
				return version.(string)
			}
		}()
	}
	ctx, cancel := context.WithTimeout(context.Background(), config.HTTPTIMEOUT)
	defer cancel()
	// get client by endPoint
	client := client.ManagerClientGlobal.GetClient(endPoint)
	// async request
	resp, err := client.Img2Img(ctx, *request, func(ctx context.Context, req *http.Request) error {
		req.Header.Add(userKey, username)
		req.Header.Add(taskKey, taskId)
		req.Header.Add(versionKey, version)
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
		Status:     config.TASK_QUEUE,
		Parameters: new(map[string]interface{}),
		Info:       new(map[string]interface{}),
		Images:     new([]string),
	}
	data, err := p.taskStore.Get(taskId, []string{datastore.KTaskStatus, datastore.KTaskImage, datastore.KTaskInfo,
		datastore.KTaskParams, datastore.KTaskCode})
	if err != nil || data == nil || len(data) == 0 {
		return nil, errors.New("not found")
	}

	// not success
	if status, ok := data[datastore.KTaskStatus]; ok && (status != config.TASK_FINISH) {
		result.Status = status.(string)
		return result, nil
	} else if ok {
		result.Status = config.TASK_FINISH
	}

	if code, ok := data[datastore.KTaskCode]; ok && code.(int64) != requestOk {
		result.Status = config.TASK_FAILED
		return result, nil
	} else if !ok {
		return nil, fmt.Errorf("task:%s predict fail", taskId)
	}

	// images
	*result.Images = strings.Split(data[datastore.KTaskImage].(string), ",")
	// params
	paramsStr := data[datastore.KTaskParams].(string)
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(paramsStr), &m); err != nil {
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Println("Unmarshal params error=", err.Error())
	}
	*result.Parameters = m
	// info
	var mm map[string]interface{}
	infoStr := data[datastore.KTaskInfo].(string)
	if err := json.Unmarshal([]byte(infoStr), &mm); err != nil {
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Println("Unmarshal Info error=", err.Error())
	}
	*result.Info = mm
	return result, nil
}

func (p *ProxyHandler) checkModelExist(sdModel, sdVae string) bool {
	models := [][]string{{config.SD_MODEL, sdModel}}
	// remove sdVae = None || Automatic
	if sdVae != "None" && sdVae != "Automatic" {
		models = append(models, []string{config.MODEL_SD_VAE, sdVae})
	}
	for _, model := range models {
		// check local existed
		switch model[0] {
		case config.SD_MODEL:
			sdModelPath := fmt.Sprintf("%s/models/%s/%s", config.ConfigGlobal.SdPath, "Stable-diffusion", sdModel)
			if !utils.FileExists(sdModelPath) {
				// list check image models
				path := fmt.Sprintf("%s/models/%s", config.ConfigGlobal.SdPath, "Stable-diffusion")
				tmp := utils.ListFile(path)
				for _, one := range tmp {
					if one == sdModel {
						return true
					}
				}
				return false
			}
		case config.MODEL_SD_VAE:
			sdVaePath := fmt.Sprintf("%s/models/%s/%s", config.ConfigGlobal.SdPath, "VAE", sdVae)
			if !utils.FileExists(sdVaePath) {
				// list check image models
				path := fmt.Sprintf("%s/models/%s", config.ConfigGlobal.SdPath, "VAE")
				tmp := utils.ListFile(path)
				for _, one := range tmp {
					if one == sdVae {
						return true
					}
				}
				return false
			}
		}
	}
	return true
}

func convertToModelResponse(datas map[string]map[string]interface{}) []*models.ModelAttributes {
	ret := make([]*models.ModelAttributes, 0, len(datas))
	for _, data := range datas {
		registeredTime := data[datastore.KModelCreateTime].(string)
		modifyTime := data[datastore.KModelModifyTime].(string)
		ret = append(ret, &models.ModelAttributes{
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
	// control server default sync
	if config.ConfigGlobal.GetFlexMode() == config.MultiFunc && config.ConfigGlobal.IsServerTypeMatch(config.CONTROL) {
		return false
	}
	if invokeType == "sync" {
		return false
	}
	return true
}

func (p *ProxyHandler) NoRouterHandler(c *gin.Context) {
	username := c.GetHeader(userKey)
	if username == "" {
		if config.ConfigGlobal.EnableLogin() {
			handleError(c, http.StatusBadRequest, config.BADREQUEST)
			return
		} else {
			username = DEFAULT_USER
		}
	}
	taskId := ""
	if isTask := c.GetHeader("Task-Flag"); isTask == "true" {
		// taskId
		taskId = c.GetHeader(taskKey)
		if taskId == "" {
			// init taskId
			taskId = utils.RandStr(taskIdLength)
		}
		c.Writer.Header().Set("taskId", taskId)
	}
	// control
	endPoint := config.ConfigGlobal.Downstream
	// get endPoint
	sdModel := ""
	body, _ := io.ReadAll(c.Request.Body)
	defer c.Request.Body.Close()
	if config.ConfigGlobal.IsServerTypeMatch(config.CONTROL) {
		// extra body
		request := make(map[string]interface{})
		err := json.Unmarshal(body, &request)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
		}
		if sd, ok := request["StableDiffusionModel"]; ok {
			sdModel = sd.(string)
		}
		c.Writer.Header().Set("model", sdModel)
		// wait to valid
		if concurrency.ConCurrencyGlobal.WaitToValid(sdModel) {
			// cold start
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Infof("sd %s cold start ....", sdModel)
			defer concurrency.ConCurrencyGlobal.DecColdNum(sdModel, taskId)
		}
		defer concurrency.ConCurrencyGlobal.DoneTask(sdModel, taskId)
		endPoint, err = module.FuncManagerGlobal.GetEndpoint(sdModel)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.SubmitTaskResponse{
				TaskId:  taskId,
				Status:  config.TASK_FAILED,
				Message: utils.String(config.NOFOUNDENDPOINT),
			})
			return
		}
	}
	// proxy
	if config.ConfigGlobal.IsServerTypeMatch(config.PROXY) {
		// check request valid: sdModel and sdVae exist
		if sdModel != "" {
			if existed := p.checkModelExist(sdModel, "None"); !existed {
				handleError(c, http.StatusNotFound, "model not found, please check request")
				return
			}
		}
		if taskId != "" {
			// write db
			if err := p.taskStore.Put(taskId, map[string]interface{}{
				datastore.KTaskIdColumnName: taskId,
				datastore.KTaskUser:         username,
				datastore.KTaskStatus:       config.TASK_QUEUE,
				datastore.KTaskCancel:       int64(config.CANCEL_INIT),
				datastore.KTaskCreateTime:   fmt.Sprintf("%d", utils.TimestampS()),
			}); err != nil {
				logrus.WithFields(logrus.Fields{"taskId": taskId}).Error("[Error] put db err=", err.Error())
				c.JSON(http.StatusInternalServerError, models.SubmitTaskResponse{
					TaskId:  taskId,
					Status:  config.TASK_FAILED,
					Message: utils.String(err.Error()),
				})
				return
			}
			c.Header("taskId", taskId)
		}
	}
	req, err := http.NewRequest(c.Request.Method, fmt.Sprintf("%s%s", endPoint, c.Request.URL.String()),
		bytes.NewReader(body))
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	req.Header = c.Request.Header
	if taskId != "" {
		req.Header.Set("taskId", taskId)
	}
	if isAsync(c.GetHeader(requestType)) {
		req.Header.Set(FcAsyncKey, "Async")
	}
	req.Header.Set(userKey, username)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	if isAsync(c.GetHeader(requestType)) {
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
	} else {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		c.Data(http.StatusOK, resp.Header.Get("Content-Type"), body)
	}
}
