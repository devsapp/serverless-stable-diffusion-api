package config

import "time"

const (
	// model status
	MODEL_REGISTERING = "registering"
	MODEL_LOADING     = "loading"
	MODEL_LOADED      = "loaded"
	MODEL_UNLOADED    = "unloaded"
	MODEL_DELETE      = "deleted"

	// task status
	TASK_INPROGRESS = "running"
	TASK_FAILED     = "failed"
	TASK_QUEUE      = "waiting"
	TASK_FINISH     = "succeeded"

	HTTPTIMEOUT = 10 * 60 * time.Second

	// cancel val
	CANCEL_INIT  = 0
	CANCEL_VALID = 1

	PROGRESS_INTERVAL = 500
)

// error message
const (
	INTERNALERROR      = "an internal error"
	BADREQUEST         = "bad request body"
	NOTFOUND           = "not found"
	NOFOUNDENDPOINT    = "not found sd endpoint, please retry"
	MODELUPDATEFCERROR = "model update fc error"
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
	//REFRESH_LORAS      = "/sdapi/v1/refresh-loras"
	//GET_LORAS          = "/sdapi/v1/loras"
	GET_SD_MODEL       = "/sdapi/v1/sd-models"
	REFRESH_SD_MODEL   = "/sdapi/v1/refresh-checkpoints"
	GET_SD_VAE         = "/sdapi/v1/sd-vae"
	REFRESH_VAE        = "/sdapi/v1/refresh-vae"
	REFRESH_CONTROLNET = "/controlnet/model_list"
	CANCEL             = "/sdapi/v1/interrupt"
	TXT2IMG            = "/sdapi/v1/txt2img"
	IMG2IMG            = "/sdapi/v1/img2img"
	PROGRESS           = "/sdapi/v1/progress"
	EXTRAIMAGES        = "/sdapi/v1/extra-single-image"
)

// ots
const (
	COLPK = "PK"
)

// env
const (
	ACCOUNT_ID              = "FC_ACCOUNT_ID"
	ACCESS_KEY_ID           = "ALIBABA_CLOUD_ACCESS_KEY_ID"
	ACCESS_KEY_SECRET       = "ALIBABA_CLOUD_ACCESS_KEY_SECRET"
	ACCESS_KET_TOKEN        = "ALIBABA_CLOUD_SECURITY_TOKEN"
	REGION                  = "FC_REGION"
	SERVICE_NAME            = "FC_SERVICE_NAME"
	OTS_ENDPOINT            = "OTS_ENDPOINT"
	OTS_INSTANCE            = "OTS_INSTANCE"
	OSS_ENDPOINT            = "OSS_ENDPOINT"
	OSS_BUCKET              = "OSS_BUCKET"
	OSS_PATH                = "OSS_PATH"
	OSS_MODE                = "OSS_MODE"
	LOGINSWITCH             = "LOGIN_SWITCH"
	USER_LOCAL_MODEL        = "USE_LOCAL_MODEL"
	SD_IMAGE                = "SD_IMAGE"
	FLEX_MODE               = "FLEX_MODE"
	EXPOSE_TO_USER          = "EXPOSE_TO_USER"
	SERVER_NAME             = "SERVER_NAME"
	DOWNSTREAM              = "DOWNSTREAM"
	GPU_MEMORY_SIZE         = "GPU_MEMORY_SIZE"
	COLD_START_CONCURRENCY  = "COLD_START_CONCURRENCY"
	MODEL_COLD_START_SERIAL = "MODEL_COLD_START_SERIAL"
)

// function http trigger
const (
	TRIGGER_TYPE         = "http"
	TRIGGER_NAME         = "defaultTrigger"
	HTTP_GET             = "GET"
	HTTP_POST            = "POST"
	HTTP_PUT             = "PUT"
	AUTH_TYPE            = "anonymous"
	MODEL_REFRESH_SIGNAL = "MODEL_REFRESH_SIGNAL"
	MODEL_SD             = "SD_MODEL"
	MODEL_SD_VAE         = "SD_VAE"
	SD_START_PARAMS      = "EXTRA_ARGS"
)

// oss mode
const (
	LOCAL  = "local"
	REMOTE = "remote"
)

type FlexMode int32

const (
	SingleFunc FlexMode = iota
	MultiFunc
)

const (
	PROXY   = "proxy"
	AGENT   = "agent"
	CONTROL = "control"
)

const (
	ColdStartConcurrency = 10
	ModelColdStartSerial = false
)
