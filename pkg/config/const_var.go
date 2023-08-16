package config

import "time"

const (
	// model status
	REGISTERING = "registering"
	LOADING     = "loading"
	LOADED      = "loaded"
	UNLOADED    = "unloaded"

	// task status
	TASK_INPROGRESS = "inprogress"
	TASK_FAILED     = "failed"
	TASK_QUEUE      = "queue"
	TASK_FINISH     = "finish"

	HTTPTIMEOUT = 60 * time.Second
)

// ERROR message
const (
	// message
	INTERNALERROR = "an internal error"
	BADREQUEST    = "bad request body"
	NOTFOUND      = "not found"
)

// model type
const (
	SD_MODEL         = "stable-diffusion"
	SD_VAE           = "sd_vae"
	LORA_MODEL       = "lora"
	CONTORLNET_MODEL = "controlNet"
)
