package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/models"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/module"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
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

// Img2Img img to img predict
// (POST /img2img)
func (a *AgentHandler) Img2Img(c *gin.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	username := c.GetHeader(userKey)
	taskId := c.GetHeader(taskKey)
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
	if err := a.updateRequest(request.OverrideSettings, username, configVer); err != nil {
		handleError(c, http.StatusInternalServerError, "please check config")
		return
	}
	// default OverrideSettingsRestoreAfterwards = true
	request.OverrideSettingsRestoreAfterwards = utils.Bool(true)
	// update task status
	a.taskStore.Update(taskId, map[string]interface{}{
		datastore.KTaskStatus: config.TASK_INPROGRESS,
	})
	// add cancel event task
	a.listenTask.AddTask(taskId, module.CancelListen, module.CancelEvent)
	// async progress
	go func() {
		if err := a.taskProgress(ctx, username, taskId); err != nil {
			log.Printf("update task progress error %s", err.Error())
		}
	}()

	body, err := json.Marshal(request)
	if err != nil {
		log.Println("request to json err=", err.Error())
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}
	// predict task
	a.predictTask(username, taskId, config.IMG2IMG, body)
	//log.Printf("task=%s predict success, request body=%s", taskId, string(body))
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
	if err := a.updateRequest(request.OverrideSettings, username, configVer); err != nil {
		handleError(c, http.StatusInternalServerError, "please check config")
		return
	}
	// default OverrideSettingsRestoreAfterwards = true
	request.OverrideSettingsRestoreAfterwards = utils.Bool(true)
	// update task status
	a.taskStore.Update(taskId, map[string]interface{}{
		datastore.KTaskStatus: config.TASK_INPROGRESS,
	})
	// add cancel event task
	a.listenTask.AddTask(taskId, module.CancelListen, module.CancelEvent)
	// async progress
	go func() {
		if err := a.taskProgress(ctx, username, taskId); err != nil {
			log.Printf("update task progress error %s", err.Error())
		}
	}()

	body, err := json.Marshal(request)
	if err != nil {
		log.Println("request to json err=", err.Error())
		handleError(c, http.StatusBadRequest, config.BADREQUEST)
		return
	}
	// predict task
	err = a.predictTask(username, taskId, config.TXT2IMG, body)
	if err != nil {
		log.Println(err.Error())
	}
	//log.Printf("task=%s predict success, request body=%s", taskId, string(body))
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
		log.Println(err.Error())
		return err
	}
	if result == nil {
		if err := a.taskStore.Update(taskId, map[string]interface{}{
			datastore.KTaskCode:       int64(resp.StatusCode),
			datastore.KTaskStatus:     config.TASK_FAILED,
			datastore.KTaskInfo:       string(body),
			datastore.KTaskModifyTime: fmt.Sprintf("%d", utils.TimestampS()),
		}); err != nil {
			log.Println(err.Error())
			return err
		}
		return errors.New("predict fail")
	}
	if result.Parameters != nil {
		result.Parameters["alwayson_scripts"] = ""
	}
	params, err := json.Marshal(result.Parameters)
	if err != nil {
		log.Println("json:", err.Error())
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
		log.Println(err.Error())
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
			log.Printf("taskid=%s is done", taskId)
			return nil
		}
		if result.Progress > 0 {
			log.Println("progress:", result.Progress)
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
					log.Println("err:", err.Error())
				}
			}
			isStart = true
		}

		// notifyDone means the update-progress is notified by other goroutine to finish,
		// either because the task has been aborted or succeed.
		if notifyDone {
			log.Printf("the task %s is done, either success or failed", taskId)
			return nil
		}

		time.Sleep(config.PROGRESS_INTERVAL * time.Millisecond)
	}
}

func (a *AgentHandler) updateRequest(overrideSettings *map[string]interface{}, username, configVersion string) error {
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
	// remove sd_model_checkpoint and sd_vae
	delete(*overrideSettings, "sd_model_checkpoint")
	delete(*overrideSettings, "sd_vae")
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

		// controlNet images: ossPath to base64Str
		if request.AlwaysonScripts != nil {
			return updateControlNet(request.AlwaysonScripts)
		}
	}
	return nil
}
func updateControlNet(alwaysonScripts *map[string]interface{}) error {
	controlNet, ok := (*alwaysonScripts)["controlnet"]
	if !ok {
		return nil
	}
	args, ok := controlNet.(map[string]interface{})["args"]
	if !ok {
		return nil
	}
	for i, item := range args.([]interface{}) {
		if val, ok := item.(map[string]interface{})["image"]; ok {
			inputImage := val.(string)
			if !isImgPath(inputImage) {
				continue
			}
			// oss image to base64
			if base64Str, err := module.OssGlobal.DownloadFileToBase64(inputImage); err == nil {
				args.([]interface{})[i].(map[string]interface{})["image"] = *base64Str
			} else {
				return err
			}
		}
	}
	(*alwaysonScripts)["controlnet"] = map[string]interface{}{"args": args}
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
func (p *AgentHandler) Login(c *gin.Context) {
	c.String(http.StatusNotFound, "api not support")
}
