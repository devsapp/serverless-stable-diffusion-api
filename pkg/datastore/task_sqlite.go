package datastore

import (
	"fmt"
	config2 "github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"strings"
)

// TaskProgress read/write the task progress to the underlying datastore.
type TaskSqlite struct {
	ds Datastore
}

func NewTask(dbType DatastoreType) (*TaskSqlite, error) {
	config := &Config{
		Type:      dbType,
		DBName:    config2.ConfigGlobal.DbSqlite,
		TableName: KTaskTableName,
		ColumnConfig: map[string]string{
			KTaskIdColumnName:       "TEXT PRIMARY KEY NOT NULL",
			KTaskProgressColumnName: "TEXT",
			KTaskUser:               "TEXT",
			KTaskImage:              "TEXT",
			KTaskCode:               "INT",
			KTaskCancel:             "INT",
			KTaskParams:             "TEXT",
			KTaskInfo:               "TEXT",
			KTaskStatus:             "TEXT",
			KTaskCreateTime:         "TEXT",
			KTaskModifyTime:         "TEXT",
		},
		PrimaryKeyColumnName: KTaskIdColumnName,
	}
	df := DatastoreFactory{}
	ds, err := df.New(config)
	if err != nil {
		return nil, err
	}
	t := &TaskSqlite{
		ds: ds,
	}
	return t, nil
}

// Close the underlying datastore.
func (t *TaskSqlite) Close() error {
	return t.ds.Close()
}

func (t *TaskSqlite) Put(taskId string, data map[string]interface{}) error {
	if taskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	data[KTaskModifyTime] = utils.TimestampS()
	err := t.ds.Put(taskId, data)
	return err
}

func (t *TaskSqlite) Get(taskId string, fields []string) (map[string]interface{}, error) {
	result, err := t.ds.Get(taskId, fields)
	return result, err
}

func (t *TaskSqlite) Update(taskId string, data map[string]interface{}) error {
	if taskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	data[KTaskModifyTime] = utils.TimestampS()
	err := t.ds.Update(taskId, data)
	return err
}

// PutProgress persist the task progress to the underlying datastore.
func (t *TaskSqlite) PutProgress(taskId string, serializedProgress string) error {
	if taskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	body := strings.Replace(serializedProgress, "eta_relative", "etaRelative", 1)
	body = strings.Replace(body, "current_image", "currentImage", 1)
	err := t.ds.Update(taskId, map[string]interface{}{
		KTaskProgressColumnName: body,
		KTaskModifyTime:         utils.TimestampS(),
	})
	return err
}

// GetProgress get the specified task progress from the underlying datastore,
// and return the result as json serialized string.
func (t *TaskSqlite) GetProgress(taskId string) (string, error) {
	result, err := t.ds.Get(taskId, []string{KTaskProgressColumnName})
	if err != nil {
		return "", err
	}
	val := result[KTaskProgressColumnName]
	if val == nil {
		return "", nil // return empty string for non-existent task
	}
	return val.(string), nil
}

func (t *TaskSqlite) PutCancel(taskId string, cancel int) error {
	if taskId == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	err := t.ds.Update(taskId, map[string]interface{}{
		KTaskCancel:     cancel,
		KTaskModifyTime: utils.TimestampS(),
	})
	return err
}
func (t *TaskSqlite) GetCancel(taskId string) (int64, error) {
	result, err := t.ds.Get(taskId, []string{KTaskCancel})
	if err != nil {
		return -1, err
	}
	val := result[KTaskCancel]
	if val == nil {
		return -1, nil // return empty -1 for non-existent task
	}
	return val.(int64), nil
}
