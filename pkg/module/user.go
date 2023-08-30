package module

import (
	"fmt"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/datastore"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"strconv"
	"sync"
)

const SESSIONLENGTH = 64

var UserManagerGlobal *userManager

// user session info
type userSession struct {
	userName string
	session  string
	expired  int
}

type userManager struct {
	userStore datastore.Datastore
	cache     *sync.Map
}

func InitUserManager(userStore datastore.Datastore) error {
	UserManagerGlobal = &userManager{
		userStore: userStore,
		cache:     new(sync.Map),
	}
	return UserManagerGlobal.loadUserFromDb()
}

// load user info from db
func (u *userManager) loadUserFromDb() error {
	datas, err := u.userStore.ListAll([]string{datastore.KUserSession, datastore.KUserName,
		datastore.KUserSessionValidTime})
	if err != nil {
		return err
	}
	for _, data := range datas {
		userName := data[datastore.KUserName].(string)
		session := data[datastore.KUserSession].(string)
		expired, _ := strconv.Atoi(data[datastore.KUserSessionValidTime].(string))
		u.cache.Store(session, &userSession{
			userName: userName,
			session:  session,
			expired:  expired,
		})
	}
	return nil
}

// VerifyUserValid verify user valid
func (u *userManager) VerifyUserValid(userName, password string) (string, int, bool) {
	data, err := u.userStore.Get(userName, []string{datastore.KUserName, datastore.KUserPassword})
	if err != nil || data == nil || len(data) == 0 {
		return "", 0, false
	}
	passwordDb := data[datastore.KUserPassword].(string)
	if password == passwordDb {
		session := utils.RandStr(SESSIONLENGTH)
		expired := int(utils.TimestampS() + config.ConfigGlobal.SessionExpire)
		u.cache.Store(session, &userSession{
			userName: userName,
			session:  session,
			expired:  expired,
		})
		return session, expired, true
	}
	return "", 0, false
}

func (u *userManager) VerifySessionValid(session string) (string, bool) {
	if session == "" || len(session) != SESSIONLENGTH {
		return "", false
	}
	curTime := utils.TimestampS()
	alreadyLoadDb := false
	for {
		// check cache
		if val, ok := u.cache.Load(session); ok {
			info := val.(*userSession)
			if curTime <= int64(info.expired) {
				// Renewal expired
				info.expired = int(curTime + config.ConfigGlobal.SessionExpire)
				u.userStore.Update(info.userName, map[string]interface{}{
					datastore.KUserSessionValidTime: fmt.Sprintf("%d", info.expired),
					datastore.KUserModifyTime:       fmt.Sprintf("%d", utils.TimestampS()),
				})
				return info.userName, true
			}
		}
		if alreadyLoadDb {
			return "", false
		}
		// expired or not in cache
		// update all user info
		u.loadUserFromDb()
		alreadyLoadDb = true
	}
}
