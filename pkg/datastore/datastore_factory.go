package datastore

import (
	"fmt"
	config2 "github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
)

type DatastoreFactory struct{}

func (f *DatastoreFactory) NewTable(dbType DatastoreType, tableName string) Datastore {
	switch dbType {
	case SQLite:
		cfg := NewSQLiteConfig(tableName)
		return NewSQLiteDatastore(cfg)
	default:
		return nil
	}
}

func NewSQLiteConfig(tableName string) *Config {
	config := &Config{
		Type:      SQLite,
		DBName:    config2.ConfigGlobal.DbSqlite,
		TableName: tableName,
	}
	switch tableName {
	case KTaskTableName:
		config.ColumnConfig = map[string]interface{}{
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
		}
		config.PrimaryKeyColumnName = KTaskIdColumnName
	case KModelTableName:
		config.ColumnConfig = map[string]interface{}{
			KModelName:       "TEXT PRIMARY KEY NOT NULL",
			KModelType:       "TEXT",
			KModelOssPath:    "TEXT",
			KModelEtag:       "TEXT",
			KModelStatus:     "TEXT",
			KModelCreateTime: "TEXT",
			KModelModifyTime: "TEXT",
		}
		config.PrimaryKeyColumnName = KModelName
	case KFuncTableName:
		config.ColumnConfig = map[string]interface{}{
			KFuncKey:        "TEXT PRIMARY KEY NOT NULL",
			KFuncSdModel:    "TEXT",
			KFuncSdVae:      "TEXT",
			KFuncEndPoint:   "TEXT",
			KCreateTime:     "TEXT",
			KLastModifyTime: "TEXT",
		}
		config.PrimaryKeyColumnName = KFuncKey
	case KUserTableName:
		config.ColumnConfig = map[string]interface{}{
			KUserName:             "TEXT PRIMARY KEY NOT NULL",
			KUserSession:          "TEXT",
			KUserSessionValidTime: "TEXT",
			kUserConfig:           "TEXT",
			KUserConfigVer:        "INT",
			KUserCreateTime:       "INT",
			kUserModifyTime:       "TEXT",
		}
		config.PrimaryKeyColumnName = KUserName
	}
	return config
}

func (f *DatastoreFactory) New(cfg *Config) (Datastore, error) {
	switch cfg.Type {
	case SQLite:
		return NewSQLiteDatastore(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported datastore type: %s", cfg.Type)
	}
}
