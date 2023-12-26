package datastore

// function table
const (
	KModelServiceTableName      = "function"
	KModelServiceKey            = "PRIMARY_KEY"
	KModelServiceFunctionName   = "FUNCTION"
	KModelServiceSdModel        = "SD_MODEL"
	KModelServiceEndPoint       = "END_POINT"
	KModelServerImage           = "IMAGE"
	KModelServiceResource       = "RESOURCE"
	KModelServiceMessage        = "MESSAGE"
	KModelServiceCreateTime     = "FUNC_CREATE_TIME"
	KModelServiceLastModifyTime = "FUNC_LAST_MODIFY_TIME"
)

// models table
const (
	KModelTableName  = "models"
	KModelType       = "MODEL_TYPE"
	KModelName       = "MODEL_NAME"
	KModelOssPath    = "MODEL_OSS_PATH"
	KModelEtag       = "MODEL_ETAG"
	KModelStatus     = "MODEL_STATUS"
	KModelLocalPath  = "MODEL_LOCAL_PATH"
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
	KUserPassword         = "USER_PASSWORD"
	KUserSession          = "USER_SESSION"
	KUserSessionValidTime = "USER_SESSION_VALID"
	KUserConfig           = "USER_CONFIG"
	KUserConfigVer        = "USER_CONFIG_VERSION"
	KUserCreateTime       = "USER_CREATE_TIME"
	KUserModifyTime       = "USER_MODIFY_TIME"
)

// config
const (
	KConfigTableName  = "config"
	KConfigKey        = "CONFIG_KEY"
	KConfigVal        = "CONFIG_VAL"
	KConfigMd5        = "CONFIG_MD5"
	KConfigVer        = "CONFIG_VERSION"
	KConfigCreateTime = "CONFIG_CREATE_TIME"
	KConfigModifyTime = "CONFIG_MODIFY_TIME"
)
