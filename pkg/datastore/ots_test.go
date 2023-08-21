package datastore

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOts(t *testing.T) {
	// create table
	tableName := "sd_test1"
	config := &Config{
		Type:      TableStore,
		TableName: tableName,
		ColumnConfig: map[string]string{
			"user": "TEXT",
			"info": "TEXT",
		},
		TimeToAlive: -1,
		MaxVersion:  1,
	}
	otsStore, err := NewOtsDatastore(config)
	assert.Nil(t, err)
	assert.NotNil(t, otsStore)

	// put
	err = otsStore.Put("lll", map[string]interface{}{
		"info": "test",
	})
	assert.Nil(t, err)
	data, err := otsStore.Get("lll", []string{"info"})
	assert.Nil(t, err)
	assert.Equal(t, data["info"].(string), "test")

	// update
	err = otsStore.Update("lll", map[string]interface{}{
		"info": "test1",
		"user": "admin",
	})
	assert.Nil(t, err)
	data, err = otsStore.Get("lll", []string{"info", "user"})
	assert.Nil(t, err)
	assert.Equal(t, data["info"].(string), "test1")
	assert.Equal(t, data["user"].(string), "admin")

	// ListAll

	datas, err := otsStore.ListAll([]string{"info", "user"})
	assert.Nil(t, err)
	assert.Equal(t, len(datas), 1)
}
