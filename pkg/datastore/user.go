package datastore

import "log"

const (
	KUserTableName        = "users"
	KUserName             = "USER_NAME"
	KUserSession          = "USER_SESSION"
	KUserSessionValidTime = "USER_SESSION_VALID"
	kUserConfig           = "USER_CONFIG"
	KUserCreateTime       = "USER_CREATE_TIME"
	kUserModifyTime       = "USER_MODIFY_TIME"
)

type UserInterface interface {
	Put(user string, data map[string]interface{}) error
	Update(user string, data map[string]interface{}) error
	Get(user string, fields []string) (map[string]interface{}, error)
	PutSession(user string, sessionId string) error
	GetSession(user string) (string, error)
	PutConfig(user string, data map[string]string) error
	GetConfig(user string) (map[string]string, error)
	GetConfigVersion(user string) (string, error)
	Close() error
}

func NewUserDataStore(dbType DatastoreType) (UserInterface, error) {
	switch dbType {
	case SQLite:
		return NewUser(dbType)
	default:
		log.Fatalf("dbType %s not support", dbType)
	}
	return nil, nil
}
