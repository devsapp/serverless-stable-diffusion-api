package datastore

// function table
const (
	KFuncTableName      = "function"
	KFuncKey            = "PRIMARY_KEY"
	KFuncSdModel        = "SD_MODEL"
	KFuncSdVae          = "SD_VAE"
	KFuncEndPoint       = "END_POINT"
	KFuncCreateTime     = "FUNC_CREATE_TIME"
	KFuncLastModifyTime = "FUNC_LAST_MODIFY_TIME"
)

// models table
const (
	KModelTableName  = "models"
	KModelType       = "MODEL_TYPE"
	KModelName       = "MODEL_NAME"
	KModelOssPath    = "MODEL_OSS_PATH"
	KModelEtag       = "MODEL_ETAG"
	KModelStatus     = "MODEL_STATUS"
	KModelCreateTime = "MODEL_REGISTERED"
	KModelModifyTime = "MODEL_MODIFY"
)

// tasks table
const (
	KTaskTableName          = "tasks"
	KTaskIdColumnName       = "TASK_ID"
	KTaskUser               = "TASK_USER"
	KTaskProgressColumnName = "TASK_PROGRESS"
	KTaskInfo               = "TASK_INFO"
	KTaskImage              = "TASK_IMAGE"
	KTaskCode               = "TASK_CODE"
	KTaskCancel             = "TASK_CANCEL"
	KTaskParams             = "TASK_PARAMS"
	KTaskStatus             = "TASK_STATUS"
	KTaskCreateTime         = "TASK_CREATE_TIME"
	KTaskModifyTime         = "TASK_MODIFY_TIME"
)

// user table
const (
	KUserTableName        = "users"
	KUserName             = "USER_NAME"
	KUserSession          = "USER_SESSION"
	KUserSessionValidTime = "USER_SESSION_VALID"
	kUserConfig           = "USER_CONFIG"
	KUserConfigVer        = "USER_CONFIG_VERSION"
	KUserCreateTime       = "USER_CREATE_TIME"
	kUserModifyTime       = "USER_MODIFY_TIME"
)
