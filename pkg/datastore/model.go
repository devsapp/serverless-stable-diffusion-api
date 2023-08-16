package datastore

import "log"

const (
	KModelTableName      = "models"
	KModelType           = "MODEL_TYPE"
	KModelName           = "MODEL_NAME"
	KModelOssPath        = "MODEL_OSS_PATH"
	KModelEtag           = "MODEL_ETAG"
	KModelStatus         = "MODEL_STATUS"
	KModelRegisteredTime = "MODEL_REGISTERED"
	KModelModifyTime     = "MODEL_MODIFY"
)

type ModelInterface interface {
	Put(modelName string, data map[string]interface{}) error
	Update(modelName string, data map[string]interface{}) error
	List(fields []string) ([]map[string]interface{}, error)
	Delete(modelName string) error
	Get(modelName string, fields []string) (map[string]interface{}, error)
}

func NewModelDataStore(dbType DatastoreType) (ModelInterface, error) {
	switch dbType {
	case SQLite:
		return NewModel(dbType)
	default:
		log.Fatalf("dbType %s not support", dbType)
	}
	return nil, nil
}
