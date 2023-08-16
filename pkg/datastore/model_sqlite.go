package datastore

import (
	"fmt"
	config2 "github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
)

// ModelSqlite read/write the task progress to the underlying datastore.
type ModelSqlite struct {
	ds Datastore
}

func NewModel(dbType DatastoreType) (*ModelSqlite, error) {
	config := &Config{
		Type:      dbType,
		DBName:    config2.ConfigGlobal.DbSqlite,
		TableName: KModelTableName,
		ColumnConfig: map[string]string{
			KModelName:           "TEXT PRIMARY KEY NOT NULL",
			KModelType:           "TEXT",
			KModelOssPath:        "TEXT",
			KModelEtag:           "TEXT",
			KModelStatus:         "TEXT",
			KModelRegisteredTime: "TEXT",
			KModelModifyTime:     "TEXT",
		},
		PrimaryKeyColumnName: KModelName,
	}
	df := DatastoreFactory{}
	ds, err := df.New(config)
	if err != nil {
		return nil, err
	}
	t := &ModelSqlite{
		ds: ds,
	}
	return t, nil
}

// Close the underlying datastore.
func (m *ModelSqlite) Close() error {
	return m.ds.Close()
}

func (m *ModelSqlite) Put(modelName string, data map[string]interface{}) error {
	if modelName == "" {
		return fmt.Errorf("modelName cannot be empty")
	}
	data[KModelRegisteredTime] = utils.TimestampS()
	err := m.ds.Put(modelName, data)
	return err
}

func (m *ModelSqlite) Update(modelName string, data map[string]interface{}) error {
	if modelName == "" {
		return fmt.Errorf("modelName cannot be empty")
	}
	data[KModelModifyTime] = utils.TimestampS()
	err := m.ds.Put(modelName, data)
	return err
}

func (m *ModelSqlite) Get(modelName string, fields []string) (map[string]interface{}, error) {
	result, err := m.ds.Get(modelName, fields)
	return result, err
}

func (m *ModelSqlite) List(fields []string) ([]map[string]interface{}, error) {
	result, err := m.ds.ListAll()
	if err != nil {
		return nil, err
	}
	ret := make([]map[string]interface{}, 0)
	for _, v := range result {
		ret = append(ret, func() map[string]interface{} {
			val := make(map[string]interface{})
			for _, field := range fields {
				if fieldVal, existed := v[field]; existed {
					val[field] = fieldVal
				}
			}
			return val
		}())
	}
	return ret, nil
}

func (m *ModelSqlite) Delete(modelName string) error {
	return m.ds.Delete(modelName)
}
