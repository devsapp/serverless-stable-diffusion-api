package config

import "time"

const (
	// model status
	MODEL_REGISTERING = "registering"
	MODEL_LOADING     = "loading"
	MODEL_LOADED      = "loaded"
	MODEL_UNLOADED    = "unloaded"

	// task status
	TASK_INPROGRESS = "running"
	TASK_FAILED     = "failed"
	TASK_QUEUE      = "waiting"
	TASK_FINISH     = "succeeded"

	HTTPTIMEOUT = 60 * time.Second
)

// ERROR message
const (
	INTERNALERROR = "an internal error"
	BADREQUEST    = "bad request body"
	NOTFOUND      = "not found"
)

// model type
const (
	SD_MODEL         = "stableDiffusion"
	SD_VAE           = "sdVae"
	LORA_MODEL       = "lora"
	CONTORLNET_MODEL = "controlNet"
)

// sd api path
const (
	REFRESH_LORAS      = "/sdapi/v1/loras"
	REFRESH_CONTROLNET = "/controlnet/model_list"
	CANCEL             = "/sdapi/v1/interrupt"
	TXT2IMG            = "/sdapi/v1/txt2img"
	IMG2IMG            = "/sdapi/v1/img2img"
	PROGRESS           = "/sdapi/v1/progress"
)

// ots
const (
	COLPK = "PK"
)
