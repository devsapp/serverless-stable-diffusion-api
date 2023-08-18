package handler

import (
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/module"
	"github.com/gin-gonic/gin"
	"net/http"
)

type AgentHandler struct {
	taskStore  datastore.Datastore
	modelStore datastore.Datastore
	httpClient *http.Client // the http client
	listenTask *module.ListenDbTask
}

func NewAgentHandler(taskStore datastore.Datastore,
	modelStore datastore.Datastore, listenTask *module.ListenDbTask) *AgentHandler {
	return &AgentHandler{
		taskStore:  taskStore,
		modelStore: modelStore,
		httpClient: &http.Client{},
		listenTask: listenTask,
	}
}

// Img2Img img to img predict
// (POST /img2img)
func (a *AgentHandler) Img2Img(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "predict task success",
	})
}

// Txt2Img txt to img predict
// (POST /txt2img)
func (a *AgentHandler) Txt2Img(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "predict task success",
	})
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
