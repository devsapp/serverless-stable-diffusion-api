package datastore

import (
	"log"
)

const (
	KTaskTableName          = "task"
	KTaskIdColumnName       = "TASK_ID"
	KTaskProgressColumnName = "TASK_PROGRESS"
	KTaskResult             = "TASK_RESULT"
	KTaskImage              = "TASK_IMAGE"
	KTaskCancel             = "TASK_CANCEL"
	KTaskParams             = "TASK_PARAMS"
	//KTaskInfo               = "TASK_INFO"
	//KTaskUser               = "TASK_USER"
)

type TaskInterface interface {
	Put(taskId string, data map[string]interface{}) error
	Get(taskId string, fields []string) (map[string]interface{}, error)
	PutProgress(taskId string, serializedProgress string) error
	GetProgress(taskId string) (string, error)
	PutCancel(taskId string, cancel int) error
	GetCancel(taskId string) (int, error)
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
