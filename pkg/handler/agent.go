package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/log"
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
	"time"
)

type AgentHandler struct {
	taskStore   datastore.Datastore
	modelStore  datastore.Datastore
	configStore datastore.Datastore
	httpClient  *http.Client // the http client
	listenTask  *module.ListenDbTask
}

func NewAgentHandler(taskStore datastore.Datastore,
	modelStore datastore.Datastore, configStore datastore.Datastore,
	listenTask *module.ListenDbTask) *AgentHandler {
	return &AgentHandler{
		taskStore:   taskStore,
		modelStore:  modelStore,
		httpClient:  &http.Client{},
		listenTask:  listenTask,
		configStore: configStore,
	}
}

// ExtraImages image upcaling
// (POST /extra_images)
func (a *AgentHandler) ExtraImages(c *gin.Context) {
	username := c.GetHeader(userKey)
	taskId := c.GetHeader(taskKey)
	c.Writer.Header().Set("taskId", taskId)
	log.SDLogInstance.SetTaskId(taskId)
	defer log.SDLogInstance.SetTaskId("")
	request := new(models.ExtraImagesJSONRequestBody)
	if err := getBindResult(c, request); err != nil {
		// update task status
		a.taskStore.Update(taskId, map[string]interface{}{
			datastore.KTaskStatus:     config.TASK_FAILED,
			datastore.KTaskCode:       int64(requestFail),
			datastore.KTaskModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
		})
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf(err.Error())
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}
	// preprocess request ossPath image to base64
	if err := preprocessRequest(request); err != nil {
		// update task status
		a.taskStore.Update(taskId, map[string]interface{}{
			datastore.KTaskStatus:     config.TASK_FAILED,
			datastore.KTaskCode:       int64(requestFail),
			datastore.KTaskModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
		})
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf(err.Error())
		handleError(c, http.StatusBadRequest, err.Error())
		return
	}
	// update task status
	a.taskStore.Update(taskId, map[string]interface{}{
		datastore.KTaskStatus: config.TASK_INPROGRESS,
	})
	body, err := json.Marshal(request)
	if err != nil {
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf("request to json err=%s", err.Error())
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}
	if err := a.extraImages(username, taskId, config.EXTRAIMAGES, body); err != nil {
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Error(err.Error())
		handleError(c, http.StatusInternalServerError, err.Error())
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message": "predict task success",
		})
	}
}

// Img2Img img to img predict
// (POST /img2img)
func (a *AgentHandler) Img2Img(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	username := c.GetHeader(userKey)
	taskId := c.GetHeader(taskKey)
	c.Writer.Header().Set("taskId", taskId)
	log.SDLogInstance.SetTaskId(taskId)
	defer log.SDLogInstance.SetTaskId("")
	configVer := c.GetHeader(versionKey)
	if username == "" || taskId == "" || configVer == "" {
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}

	request := new(models.Img2ImgJSONRequestBody)
	if err := getBindResult(c, request); err != nil {
		// update task status
		a.taskStore.Update(taskId, map[string]interface{}{
			datastore.KTaskStatus:     config.TASK_FAILED,
			datastore.KTaskCode:       int64(requestFail),
			datastore.KTaskModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
		})
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}
	// preprocess request ossPath image to base64
	if err := preprocessRequest(request); err != nil {
		// update task status
		a.taskStore.Update(taskId, map[string]interface{}{
			datastore.KTaskStatus:     config.TASK_FAILED,
			datastore.KTaskCode:       int64(requestFail),
			datastore.KTaskModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
		})
		handleError(c, http.StatusBadRequest, err.Error())
		return
	}
	// update request OverrideSettings
	if request.OverrideSettings == nil {
		overrideSettings := make(map[string]interface{})
		request.OverrideSettings = &overrideSettings
	}
	if err := a.updateOverrideSettingsRequest(request.OverrideSettings, username, configVer,
		request.StableDiffusionModel, request.SdVae); err != nil {
		handleError(c, http.StatusInternalServerError, "please check config")
		return
	}
	// default OverrideSettingsRestoreAfterwards = true
	request.OverrideSettingsRestoreAfterwards = utils.Bool(false)
	// update task status
	a.taskStore.Update(taskId, map[string]interface{}{
		datastore.KTaskStatus: config.TASK_INPROGRESS,
	})
	// add cancel event task
	a.listenTask.AddTask(taskId, module.CancelListen, module.CancelEvent)
	// async progress
	go func() {
		if err := a.taskProgress(ctx, username, taskId); err != nil {
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf(
				"update task progress error %s", err.Error())
		}
	}()

	body, err := json.Marshal(request)
	if err != nil {
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorln("request to json err=", err.Error())
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}
	// predict task
	a.predictTask(username, taskId, config.IMG2IMG, body)
	c.JSON(http.StatusOK, gin.H{
		"message": "predict task success",
	})
}

// Txt2Img txt to img predict
// (POST /txt2img)
func (a *AgentHandler) Txt2Img(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	username := c.GetHeader(userKey)
	taskId := c.GetHeader(taskKey)
	c.Writer.Header().Set("taskId", taskId)
	log.SDLogInstance.SetTaskId(taskId)
	defer log.SDLogInstance.SetTaskId("")
	configVer := c.GetHeader(versionKey)
	if username == "" || taskId == "" || configVer == "" {
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}

	request := new(models.Txt2ImgJSONRequestBody)
	if err := getBindResult(c, request); err != nil {
		// update task status
		a.taskStore.Update(taskId, map[string]interface{}{
			datastore.KTaskStatus:     config.TASK_FAILED,
			datastore.KTaskCode:       int64(requestFail),
			datastore.KTaskModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
		})
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}
	// preprocess request ossPath image to base64
	if err := preprocessRequest(request); err != nil {
		// update task status
		a.taskStore.Update(taskId, map[string]interface{}{
			datastore.KTaskStatus:     config.TASK_FAILED,
			datastore.KTaskCode:       int64(requestFail),
			datastore.KTaskModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
		})
		handleError(c, http.StatusBadRequest, err.Error())
		return
	}
	// update request OverrideSettings
	if request.OverrideSettings == nil {
		overrideSettings := make(map[string]interface{})
		request.OverrideSettings = &overrideSettings
	}
	if err := a.updateOverrideSettingsRequest(request.OverrideSettings, username, configVer,
		request.StableDiffusionModel, request.SdVae); err != nil {
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf("update OverrideSettings err=%s", err.Error())
		handleError(c, http.StatusInternalServerError, "please check config")
		return
	}
	// default OverrideSettingsRestoreAfterwards = true
	request.OverrideSettingsRestoreAfterwards = utils.Bool(false)
	// update task status
	if err := a.taskStore.Update(taskId, map[string]interface{}{
		datastore.KTaskStatus: config.TASK_INPROGRESS,
	}); err != nil {
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf("put db err=%s", err.Error())
		handleError(c, http.StatusInternalServerError, err.Error())
		return
	}
	// add cancel event task
	a.listenTask.AddTask(taskId, module.CancelListen, module.CancelEvent)
	// async progress
	go func() {
		if err := a.taskProgress(ctx, username, taskId); err != nil {
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf(
				"update task progress error %s", err.Error())
		}
	}()

	body, err := json.Marshal(request)
	if err != nil {
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorln("request to json err=", err.Error())
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}
	// predict task
	err = a.predictTask(username, taskId, config.TXT2IMG, body)
	if err != nil {
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorln(err.Error())
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "predict task success",
	})
}

func (a *AgentHandler) predictTask(user, taskId, path string, body []byte) error {
	url := fmt.Sprintf("%s%s", config.ConfigGlobal.SdUrlPrefix, path)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}

	body, err = io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	var result *models.Txt2ImgResult

	if err := json.Unmarshal(body, &result); err != nil {
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorln(err.Error())
		return err
	}
	if result == nil {
		if err := a.taskStore.Update(taskId, map[string]interface{}{
			datastore.KTaskCode:       int64(resp.StatusCode),
			datastore.KTaskStatus:     config.TASK_FAILED,
			datastore.KTaskInfo:       string(body),
			datastore.KTaskModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
		}); err != nil {
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Println(err.Error())
			return err
		}
		return errors.New("predict fail")
	}
	if result.Parameters != nil {
		result.Parameters["alwayson_scripts"] = ""
	}
	params, err := json.Marshal(result.Parameters)
	if err != nil {
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Println("json:", err.Error())
	}
	var images []string
	if resp.StatusCode == requestOk {
		count := len(result.Images)
		for i := 1; i <= count; i++ {
			// upload image to oss
			ossPath := fmt.Sprintf("images/%s/%s_%d.png", user, taskId, i)
			if err := uploadImages(&ossPath, &result.Images[i-1]); err != nil {
				return fmt.Errorf("output image err=%s", err.Error())
			}

			images = append(images, ossPath)
		}
	}
	if err := a.taskStore.Update(taskId, map[string]interface{}{
		datastore.KTaskCode:       int64(resp.StatusCode),
		datastore.KTaskStatus:     config.TASK_FINISH,
		datastore.KTaskImage:      strings.Join(images, ","),
		datastore.KTaskParams:     string(params),
		datastore.KTaskInfo:       result.Info,
		datastore.KTaskModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
	}); err != nil {
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorln(err.Error())
		return err
	}
	return nil
}

func (a *AgentHandler) taskProgress(ctx context.Context, user, taskId string) error {
	var isStart bool
	notifyDone := false
	for {
		select {
		case <-ctx.Done():
			notifyDone = true
		default:
			// Do nothing, go to the next
		}
		progressUrl := fmt.Sprintf("%s%s", config.ConfigGlobal.SdUrlPrefix, config.PROGRESS)
		req, _ := http.NewRequest("GET", progressUrl, nil)
		resp, err := a.httpClient.Do(req)
		if err != nil {
			return err
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		resp.Body.Close()

		var result models.ProgressResult
		if err := json.Unmarshal(body, &result); err != nil {
			return err
		}

		// Get progress judge task done
		if isStart && result.Progress <= 0 {
			return nil
		}
		if result.Progress > 0 {
			// output to oss
			if result.CurrentImage != "" && config.ConfigGlobal.EnableProgressImg() {
				ossPath := fmt.Sprintf("images/%s/%s_progress.png", user, taskId)
				if err := uploadImages(&ossPath, &result.CurrentImage); err != nil {
					return fmt.Errorf("output image err=%s", err.Error())
				}
				result.CurrentImage = ossPath
			} else {
				result.CurrentImage = ""
			}
			// write to db, struct to json str
			if resultStr, err := json.Marshal(result); err == nil {
				// modify key: current_image=>currentImage, eta_relative=>etaRelative
				resultByte := strings.Replace(string(resultStr), "current_image", "currentImage", 1)
				resultByte = strings.Replace(string(resultByte), "eta_relative", "etaRelative", 1)
				// Update the task progress to DB.
				if err = a.taskStore.Update(taskId, map[string]interface{}{
					datastore.KTaskProgressColumnName: string(resultByte),
					datastore.KTaskModifyTime:         fmt.Sprintf("%d", utils.TimestampS()),
				}); err != nil {
					logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorln("err:", err.Error())
				}
			}
			isStart = true
		}

		// notifyDone means the update-progress is notified by other goroutine to finish,
		// either because the task has been aborted or succeed.
		if notifyDone {
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Infof(
				"the task %s is done, either success or failed", taskId)
			return nil
		}

		time.Sleep(config.PROGRESS_INTERVAL * time.Millisecond)
	}
}

func (a *AgentHandler) extraImages(user, taskId, path string, body []byte) error {
	url := fmt.Sprintf("%s%s", config.ConfigGlobal.SdUrlPrefix, path)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}

	body, err = io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	var result *models.ExtraImageResult

	if err := json.Unmarshal(body, &result); err != nil {
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorln(err.Error())
		return err
	}
	if result == nil {
		if err := a.taskStore.Update(taskId, map[string]interface{}{
			datastore.KTaskCode:       int64(resp.StatusCode),
			datastore.KTaskStatus:     config.TASK_FAILED,
			datastore.KTaskInfo:       string(body),
			datastore.KTaskModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
		}); err != nil {
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Info(err.Error())
			return err
		}
		return errors.New("predict fail")
	}
	var images []string
	if resp.StatusCode == requestOk {
		// upload image to oss
		ossPath := fmt.Sprintf("images/%s/%s_%d.png", user, taskId, 1)
		if err := uploadImages(&ossPath, &result.Image); err != nil {
			return fmt.Errorf("output image err=%s", err.Error())
		}

		images = append(images, ossPath)
	}
	if err := a.taskStore.Update(taskId, map[string]interface{}{
		datastore.KTaskCode:       int64(resp.StatusCode),
		datastore.KTaskStatus:     config.TASK_FINISH,
		datastore.KTaskImage:      strings.Join(images, ","),
		datastore.KTaskParams:     "{}",
		datastore.KTaskInfo:       fmt.Sprintf("{\"html_info\":\"%s\"}", result.HTMLInfo),
		datastore.KTaskModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
	}); err != nil {
		logrus.WithFields(logrus.Fields{"taskId": taskId}).Error(err.Error())
		return err
	}
	return nil
}

func (a *AgentHandler) updateOverrideSettingsRequest(overrideSettings *map[string]interface{},
	username, configVersion, sdModel, sdVae string) error {
	//if config.ConfigGlobal.GetFlexMode() == config.MultiFunc {
	//	// remove sd_model_checkpoint and sd_vae
	//	delete(*overrideSettings, "sd_model_checkpoint")
	//	(*overrideSettings)["sd_vae"] = sdVae
	//} else {
	(*overrideSettings)["sd_model_checkpoint"] = sdModel
	(*overrideSettings)["sd_vae"] = sdVae
	//}
	// version == -1 use default
	if configVersion == "-1" {
		return nil
	}
	// read config from db
	key := fmt.Sprintf("%s_%s", username, configVersion)
	data, err := a.configStore.Get(key, []string{datastore.KConfigVal})
	if err != nil {
		return err
	}
	// no user config, user default
	if data == nil || len(data) == 0 {
		return nil
	}
	val := data[datastore.KConfigVal].(string)
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(val), &m); err != nil {
		return nil
	}
	// priority request > db
	for k, v := range m {
		if _, ok := (*overrideSettings)[k]; !ok {
			(*overrideSettings)[k] = v
		}
	}
	return nil
}

// deal ossImg to base64
func preprocessRequest(req any) error {
	switch req.(type) {
	case *models.ExtraImagesJSONRequestBody:
		request := req.(*models.ExtraImagesJSONRequestBody)
		if request.Image != "" {
			if isImgPath(request.Image) {

				base64, err := module.OssGlobal.DownloadFileToBase64(request.Image)
				if err != nil {
					return err
				}
				request.Image = *base64
			}
		}
	case *models.Txt2ImgJSONRequestBody:
		request := req.(*models.Txt2ImgJSONRequestBody)
		if request.AlwaysonScripts != nil {
			return updateControlNet(request.AlwaysonScripts)
		}
	case *models.Img2ImgJSONRequestBody:
		request := req.(*models.Img2ImgJSONRequestBody)
		// init images: ossPath to base64Str
		for i, str := range *request.InitImages {
			if !isImgPath(str) {
				continue
			}
			base64, err := module.OssGlobal.DownloadFileToBase64(str)
			if err != nil {
				return err
			}
			(*request.InitImages)[i] = *base64
		}

		// mask images: ossPath to base64St
		if request.Mask != nil && isImgPath(*request.Mask) {
			base64, err := module.OssGlobal.DownloadFileToBase64(*request.Mask)
			if err != nil {
				return err
			}
			*request.Mask = *base64
		}

		// controlNet images: ossPath to base64Str
		if request.AlwaysonScripts != nil {
			return updateControlNet(request.AlwaysonScripts)
		}
	}
	return nil
}
func updateControlNet(alwaysonScripts *map[string]interface{}) error {
	*alwaysonScripts = parseMap(*alwaysonScripts, "", "", nil)
	//controlNet, ok := (*alwaysonScripts)["controlnet"]
	//if !ok {
	//	return nil
	//}
	//args, ok := controlNet.(map[string]interface{})["args"]
	//if !ok {
	//	return nil
	//}
	//for i, item := range args.([]interface{}) {
	//	if val, ok := item.(map[string]interface{})["image"]; ok && val != nil {
	//		inputImage := val.(string)
	//		if !isImgPath(inputImage) {
	//			continue
	//		}
	//		// oss image to base64
	//		if base64Str, err := module.OssGlobal.DownloadFileToBase64(inputImage); err == nil {
	//			args.([]interface{})[i].(map[string]interface{})["image"] = *base64Str
	//		} else {
	//			return err
	//		}
	//	}
	//	if val, ok := item.(map[string]interface{})["mask"]; ok && val != nil {
	//		inputImage := val.(string)
	//		if !isImgPath(inputImage) {
	//			continue
	//		}
	//		// oss image to base64
	//		if base64Str, err := module.OssGlobal.DownloadFileToBase64(inputImage); err == nil {
	//			args.([]interface{})[i].(map[string]interface{})["mask"] = *base64Str
	//		} else {
	//			return err
	//		}
	//	}
	//}
	//(*alwaysonScripts)["controlnet"] = map[string]interface{}{"args": args}
	return nil
}

// ListModels list model, not support
// (GET /models)
func (a *AgentHandler) ListModels(c *gin.Context) {
	c.String(http.StatusNotFound, "api not support")
}

// RegisterModel register model, not support
// (POST /models)
func (a *AgentHandler) RegisterModel(c *gin.Context) {
	c.String(http.StatusNotFound, "api not support")
}

// DeleteModel delete model, not support
// (DELETE /models/{model_name})
func (a *AgentHandler) DeleteModel(c *gin.Context, modelName string) {
	c.String(http.StatusNotFound, "api not support")
}

// GetModel get model info, not support
// (GET /models/{model_name})
func (a *AgentHandler) GetModel(c *gin.Context, modelName string) {
	c.String(http.StatusNotFound, "api not support")
}

// UpdateModel update model, not support
// (PUT /models/{model_name})
func (a *AgentHandler) UpdateModel(c *gin.Context, modelName string) {
	c.String(http.StatusNotFound, "api not support")
}

// CancelTask cancel predict task, not support
// (GET /tasks/{taskId}/cancellation)
func (a *AgentHandler) CancelTask(c *gin.Context, taskId string) {
	c.String(http.StatusNotFound, "api not support")
}

// GetTaskProgress get predict progress, not support
// (GET /tasks/{taskId}/progress)
func (a *AgentHandler) GetTaskProgress(c *gin.Context, taskId string) {
	c.String(http.StatusNotFound, "api not support")
}

// GetTaskResult get predict result, not support
// (GET /tasks/{taskId}/result)
func (a *AgentHandler) GetTaskResult(c *gin.Context, taskId string) {
	c.String(http.StatusNotFound, "api not support")
}

// UpdateOptions update config options
// (POST /options)
func (a *AgentHandler) UpdateOptions(c *gin.Context) {
	c.String(http.StatusNotFound, "api not support")
}

// Login user login
// (POST /login)
func (a *AgentHandler) Login(c *gin.Context) {
	c.String(http.StatusNotFound, "api not support")
}

// Restart user login
// (POST /restart)
func (a *AgentHandler) Restart(c *gin.Context) {
	c.String(http.StatusNotFound, "api not support")
}

// BatchUpdateResource update sd function resource by batch, Supports a specified list of functions, or all
// (POST /batch_update_sd_resource)
func (a *AgentHandler) BatchUpdateResource(c *gin.Context) {
	c.String(http.StatusNotFound, "api not support")
}

// ListSdFunc get sdapi function
// (GET /list/sdapi/functions)
func (a *AgentHandler) ListSdFunc(c *gin.Context) {
	c.String(http.StatusNotFound, "api not support")
}

func ReverseProxy(c *gin.Context) {
	target := config.ConfigGlobal.SdUrlPrefix
	remote, err := url.Parse(target)
	if err != nil {
		panic(err)
	}
	requestId := c.GetHeader(config.FcRequestID)
	//requestId := utils.RandStr(10)
	log.SDLogInstance.AddRequestId(requestId)
	defer log.SDLogInstance.DelRequestId(requestId)
	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Director = func(req *http.Request) {
		req.Header = c.Request.Header
		req.Host = remote.Host
		req.URL.Scheme = remote.Scheme
		req.URL.Host = remote.Host
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Expose-Headers")
		resp.Header.Del("Access-Control-Max-Age")
		resp.Header.Del("Access-Control-Allow-Credentials")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Allow-Headers")
		return nil
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}

func (a *AgentHandler) NoRouterAgentHandler(c *gin.Context) {
	username := c.GetHeader(userKey)
	// taskId
	taskId := c.GetHeader(taskKey)
	c.Writer.Header().Set("taskId", taskId)
	if taskId != "" {
		// update task status
		if err := a.taskStore.Update(taskId, map[string]interface{}{
			datastore.KTaskStatus: config.TASK_INPROGRESS,
		}); err != nil {
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Errorf("put db err=%s", err.Error())
			handleError(c, http.StatusInternalServerError, err.Error())
			return
		}
	}
	body, err := io.ReadAll(c.Request.Body)
	defer c.Request.Body.Close()
	if err != nil {
		handleError(c, http.StatusBadRequest, err.Error())
		return
	}
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodDelete {
		body, err = convertImgToBase64(body)
		if err != nil {
			handleError(c, http.StatusBadRequest, err.Error())
			return
		}
	}
	req, err := http.NewRequest(c.Request.Method,
		fmt.Sprintf("%s%s", config.ConfigGlobal.SdUrlPrefix,
			c.Request.URL.String()), bytes.NewBuffer(body))
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	req.Header = c.Request.Header
	req.Header.Set("Accept-Encoding", "identity")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	respBody, err := convertBase64ToImg(body, taskId, username)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	if taskId != "" {
		if err := a.taskStore.Update(taskId, map[string]interface{}{
			datastore.KTaskCode:       int64(resp.StatusCode),
			datastore.KTaskStatus:     config.TASK_FINISH,
			datastore.KTaskImage:      "",
			datastore.KTaskParams:     "{}",
			datastore.KTaskInfo:       string(respBody),
			datastore.KTaskModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
		}); err != nil {
			logrus.WithFields(logrus.Fields{"taskId": taskId}).Error(err.Error())
			c.String(http.StatusInternalServerError, err.Error())
		}
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}
