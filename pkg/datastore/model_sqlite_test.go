package datastore

import (
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var modelStore ModelInterface

func TestModelSqlite(t *testing.T) {
	config.InitConfig("")
	config.ConfigGlobal.DbSqlite = "./sqlite"
	os.Remove(config.ConfigGlobal.DbSqlite)
	modelStore, _ = NewModelDataStore(SQLite)
	t.Run("model", model)
}

func model(t *testing.T) {
	modelName := utils.RandStr(10)
	err := modelStore.Put(modelName, map[string]interface{}{
		KModelName:   modelName,
		KModelStatus: config.LOADED,
	})
	assert.Nil(t, err)
	err = modelStore.Update(modelName, map[string]interface{}{
		KModelStatus: config.UNLOADED,
	})
	assert.Nil(t, err)
	data, err := modelStore.Get(modelName, []string{KModelStatus})
	assert.Nil(t, err)
	assert.Equal(t, config.UNLOADED, data[KModelStatus].(string))

	err = modelStore.Delete(modelName)
	assert.Nil(t, err)

	dataList, err := modelStore.List([]string{KModelStatus})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(dataList))
}
