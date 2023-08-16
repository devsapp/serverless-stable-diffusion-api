package datastore

import (
	"log"
)

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

type TaskInterface interface {
	Put(taskId string, data map[string]interface{}) error
	Get(taskId string, fields []string) (map[string]interface{}, error)
	Update(taskId string, data map[string]interface{}) error
	PutProgress(taskId string, serializedProgress string) error
	GetProgress(taskId string) (string, error)
	PutCancel(taskId string, cancel int) error
	GetCancel(taskId string) (int64, error)
	Close() error
}

func NewTaskDataStore(dbType DatastoreType) (TaskInterface, error) {
	switch dbType {
	case SQLite:
		return NewTask(dbType)
	default:
		log.Fatalf("dbtyp: %s not support", dbType)
	}
	return nil, nil
}
