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
	case TableStore:
		cfg := NewOtsConfig(tableName)
		otsStore, err := NewOtsDatastore(cfg)
		if err != nil {
			panic(fmt.Sprintf("init ots fail, err=%s", err.Error()))
			return nil
		}
		return otsStore
	default:
		panic(fmt.Sprintf("not support db type=%s", dbType))
	}
	return nil
}

func NewSQLiteConfig(tableName string) *Config {
	config := &Config{
		Type:      SQLite,
		DBName:    config2.ConfigGlobal.DbSqlite,
		TableName: tableName,
	}
	switch tableName {
	case KTaskTableName:
		config.ColumnConfig = map[string]string{
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
		config.ColumnConfig = map[string]string{
			KModelName:       "TEXT PRIMARY KEY NOT NULL",
			KModelType:       "TEXT",
			KModelOssPath:    "TEXT",
			KModelEtag:       "TEXT",
			KModelStatus:     "TEXT",
			KModelCreateTime: "TEXT",
			KModelModifyTime: "TEXT",
		}
		config.PrimaryKeyColumnName = KModelName
	case KModelServiceTableName:
		config.ColumnConfig = map[string]string{
			KModelServiceKey:            "TEXT PRIMARY KEY NOT NULL",
			KModelServiceFunctionName:   "TEXT",
			KModelServiceSdModel:        "TEXT",
			KModelServiceEndPoint:       "TEXT",
			KModelServerImage:           "TEXT",
			KModelServiceCreateTime:     "TEXT",
			KModelServiceLastModifyTime: "TEXT",
		}
		config.PrimaryKeyColumnName = KModelServiceKey
	case KUserTableName:
		config.ColumnConfig = map[string]string{
			KUserName:             "TEXT PRIMARY KEY NOT NULL",
			KUserSession:          "TEXT",
			KUserSessionValidTime: "TEXT",
			KUserConfig:           "TEXT",
			KUserConfigVer:        "TEXT",
			KUserCreateTime:       "TEXT",
			KUserModifyTime:       "TEXT",
			KUserPassword:         "TEXT",
		}
		config.PrimaryKeyColumnName = KUserName
	case KConfigTableName:
		config.ColumnConfig = map[string]string{
			KConfigKey:        "TEXT PRIMARY KEY NOT NULL",
			KConfigVal:        "TEXT",
			KConfigVer:        "TEXT",
			KConfigMd5:        "TEXT",
			KConfigCreateTime: "TEXT",
			KConfigModifyTime: "TEXT",
		}
		config.PrimaryKeyColumnName = KConfigKey
	}
	return config
}

func NewOtsConfig(tableName string) *Config {
	config := &Config{
		Type:        TableStore,
		TableName:   tableName,
		TimeToAlive: -1,
		MaxVersion:  1,
	}
	switch tableName {
	case KTaskTableName:
		config.ColumnConfig = map[string]string{
			KTaskIdColumnName:       "TEXT",
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
		config.ColumnConfig = map[string]string{
			KModelName:       "TEXT",
			KModelType:       "TEXT",
			KModelOssPath:    "TEXT",
			KModelEtag:       "TEXT",
			KModelStatus:     "TEXT",
			KModelCreateTime: "TEXT",
			KModelModifyTime: "TEXT",
		}
		config.PrimaryKeyColumnName = KModelName
	case KModelServiceTableName:
		config.ColumnConfig = map[string]string{
			KModelServiceKey:            "TEXT",
			KModelServiceFunctionName:   "TEXT",
			KModelServiceSdModel:        "TEXT",
			KModelServiceEndPoint:       "TEXT",
			KModelServerImage:           "TEXT",
			KModelServiceCreateTime:     "TEXT",
			KModelServiceLastModifyTime: "TEXT",
		}
		config.PrimaryKeyColumnName = KModelServiceKey
	case KUserTableName:
		config.ColumnConfig = map[string]string{
			KUserName:             "TEXT",
			KUserSession:          "TEXT",
			KUserSessionValidTime: "TEXT",
			KUserConfig:           "TEXT",
			KUserConfigVer:        "TEXT",
			KUserCreateTime:       "TEXT",
			KUserModifyTime:       "TEXT",
			KUserPassword:         "TEXT",
		}
		config.PrimaryKeyColumnName = KUserName
	case KConfigTableName:
		config.ColumnConfig = map[string]string{
			KConfigKey:        "TEXT",
			KConfigVal:        "TEXT",
			KConfigVer:        "TEXT",
			KConfigMd5:        "TEXT",
			KConfigCreateTime: "TEXT",
			KConfigModifyTime: "TEXT",
		}
		config.PrimaryKeyColumnName = KConfigKey
	}
	return config
}
