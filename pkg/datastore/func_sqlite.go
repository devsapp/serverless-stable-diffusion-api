package datastore

import config2 "github.com/devsapp/serverless-stable-diffusion-api/pkg/config"

type FuncSqlite struct {
	ds Datastore
}

func NewFunc(dbType DatastoreType) (*FuncSqlite, error) {
	config := &Config{
		Type:      dbType,
		DBName:    config2.ConfigGlobal.DbSqlite,
		TableName: KFuncTableName,
		ColumnConfig: map[string]string{
			KFuncKey:        "TEXT PRIMARY KEY NOT NULL",
			KFuncSdModel:    "TEXT",
			KFuncSdVae:      "TEXT",
			KFuncEndPoint:   "TEXT",
			KCreateTime:     "TEXT",
			kLastModifyTime: "TEXT",
		},
		PrimaryKeyColumnName: KFuncKey,
	}
	df := DatastoreFactory{}
	ds, err := df.New(config)
	if err != nil {
		return nil, err
	}
	t := &FuncSqlite{
		ds: ds,
	}
	return t, nil
}

func (f *FuncSqlite) Put(key string, data map[string]string) error {
	return nil
}
func (f *FuncSqlite) Get(key string, fields []string) (map[string]string, error) {
	return nil, nil
}
func (f *FuncSqlite) ListAll(fields []string) ([]map[string]string, error) {
	return nil, nil
}
