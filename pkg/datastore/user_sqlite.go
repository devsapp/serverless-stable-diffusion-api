package datastore

import (
	"fmt"
	config2 "github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
)

type UserSqlite struct {
	ds             Datastore
	sessionExpired int
}

func NewUser(dbType DatastoreType) (*UserSqlite, error) {
	config := &Config{
		Type:      dbType,
		DBName:    config2.ConfigGlobal.DbSqlite,
		TableName: KUserTableName,
		ColumnConfig: map[string]string{
			KUserName:             "TEXT PRIMARY KEY NOT NULL",
			KUserSession:          "TEXT",
			KUserSessionValidTime: "TEXT",
			kUserConfig:           "TEXT",
			KUserCreateTime:       "INT",
			kUserModifyTime:       "TEXT",
		},
		PrimaryKeyColumnName: KUserName,
	}
	df := DatastoreFactory{}
	ds, err := df.New(config)
	if err != nil {
		return nil, err
	}
	t := &UserSqlite{
		ds:             ds,
		sessionExpired: config2.ConfigGlobal.SessionExpire,
	}
	return t, nil
}

// Close the underlying datastore.
func (t *UserSqlite) Close() error {
	return t.ds.Close()
}

func (t *UserSqlite) Put(user string, data map[string]interface{}) error {
	if user == "" {
		return fmt.Errorf("user cannot be empty")
	}
	data[kUserModifyTime] = utils.TimestampS()
	return t.ds.Put(user, data)
}

func (t *UserSqlite) Update(user string, data map[string]interface{}) error {
	if user == "" {
		return fmt.Errorf("user cannot be empty")
	}
	data[kUserModifyTime] = utils.TimestampS()
	return t.ds.Update(user, data)
}

func (t *UserSqlite) Get(user string, fields []string) (map[string]interface{}, error) {
	result, err := t.ds.Get(user, fields)
	return result, err
}
func (t *UserSqlite) PutSession(user string, sessionId string) error {
	return nil
}
func (t *UserSqlite) GetSession(user string) (string, error) {
	return "", nil
}
func (t *UserSqlite) PutConfig(user string, data map[string]string) error {
	return nil
}
func (t *UserSqlite) GetConfig(user string) (map[string]string, error) {
	return nil, nil
}
func (t *UserSqlite) GetConfigVersion(user string) (string, error) {
	return "0", nil
}
