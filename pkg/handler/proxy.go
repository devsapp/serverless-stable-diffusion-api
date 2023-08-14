package handler

import (
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/gin-gonic/gin"
	"net/http"
)

const (
	taskIdLength = 10
)

type ProxyHandler struct {
	taskStore  datastore.TaskInterface
	modelStore datastore.ModelInterface
}

func NewProxyHandler(taskStore datastore.TaskInterface,
	modelStore datastore.ModelInterface) *ProxyHandler {
	return &ProxyHandler{
		taskStore:  taskStore,
		modelStore: modelStore,
	}
}

// CancelTask predict task
// (POST /tasks/{taskId}/cancellation)
func (p *ProxyHandler) CancelTask(c *gin.Context, taskId string) {
	c.String(http.StatusOK, "success")
}

// GetTaskResult GetGetResult get predict progress
// (GET /tasks/{taskId}/result)
func (p *ProxyHandler) GetTaskResult(c *gin.Context, taskId string) {
	c.JSON(http.StatusOK, gin.H{})
}

// Img2Img img to img predict
// (POST /img2img)
func (p *ProxyHandler) Img2Img(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// ListModels list model
// (GET /models)
func (p *ProxyHandler) ListModels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})

}

// RegisterModel upload model
// (POST /models)
func (p *ProxyHandler) RegisterModel(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}

// DeleteModel delete model
// (DELETE /models/{model_name})
func (p *ProxyHandler) DeleteModel(c *gin.Context, modelName string) {
	c.JSON(http.StatusOK, gin.H{})
}

// GetModel get model info
// (GET /models/{model_name})
func (p *ProxyHandler) GetModel(c *gin.Context, modelName string) {
	c.JSON(http.StatusOK, gin.H{})

}

// UpdateModel update model
// (PUT /models/{model_name})
func (p *ProxyHandler) UpdateModel(c *gin.Context, modelName string) {
	c.JSON(http.StatusOK, gin.H{})

}

// GetTaskProgress get predict progress
// (GET /tasks/{taskId}/progress)
func (p *ProxyHandler) GetTaskProgress(c *gin.Context, taskId string) {
	c.JSON(http.StatusOK, gin.H{})

}

// Txt2Img txt to img predict
// (POST /txt2img)
func (p *ProxyHandler) Txt2Img(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{})
}
