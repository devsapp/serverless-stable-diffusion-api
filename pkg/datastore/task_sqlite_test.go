package datastore

import (
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/config"
	"github.com/devsapp/serverless-stable-diffusion-api/pkg/utils"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

var taskStore TaskInterface

func TestTaskSqlite(t *testing.T) {
	config.InitConfig("")
	config.ConfigGlobal.DbSqlite = "./sqlite"
	os.Remove(config.ConfigGlobal.DbSqlite)
	taskStore, _ = NewTaskDataStore(SQLite)
	t.Run("task", task)
	t.Run("cancel", cancel)
	t.Run("progress", progress)

}

func task(t *testing.T) {
	taskId := utils.RandStr(10)
	err := taskStore.Put(taskId, map[string]interface{}{
		KTaskIdColumnName: taskId,
		KTaskStatus:       "inprogress",
	})
	assert.Nil(t, err)
	err = taskStore.Update(taskId, map[string]interface{}{
		KTaskStatus: "queue",
	})
	assert.Nil(t, err)
	data, err := taskStore.Get(taskId, []string{KTaskStatus})
	assert.Nil(t, err)
	assert.Equal(t, "queue", data[KTaskStatus].(string))
}

func cancel(t *testing.T) {
	// create
	taskId := utils.RandStr(10)
	cancel, err := taskStore.GetCancel(taskId)
	assert.Nil(t, err)
	assert.Equal(t, cancel, int64(-1))
	err = taskStore.Put(taskId, map[string]interface{}{KTaskCancel: 0})
	assert.Nil(t, err)
	err = taskStore.PutCancel(taskId, 1)
	assert.Nil(t, err)
	cancel, err = taskStore.GetCancel(taskId)
	assert.Nil(t, err)
	assert.Equal(t, cancel, int64(1))

}

func progress(t *testing.T) {
	// create
	taskId := utils.RandStr(10)
	err := taskStore.Put(taskId, map[string]interface{}{KTaskIdColumnName: taskId})
	assert.Nil(t, err)
	err = taskStore.PutProgress(taskId, "{\"current_image\":xxxx}")
	assert.Nil(t, err)
	progress, err := taskStore.GetProgress(taskId)
	assert.Nil(t, err)
	assert.Equal(t, progress, "{\"currentImage\":xxxx}")

}
