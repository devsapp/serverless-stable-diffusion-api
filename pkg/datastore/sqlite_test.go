package datastore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSQLiteDatastore(t *testing.T) {
	primaryKeyColumnName := "primaryKey"
	config := &Config{
		DBName:    ":memory:", // the memory database for testing purposes
		TableName: "TestSQLiteDatastore",
		ColumnConfig: map[string]string{
			primaryKeyColumnName: "text primary key not null",
			"value":              "text",
			"intCol":             "int",
			"floatCol":           "float",
		},
		PrimaryKeyColumnName: primaryKeyColumnName,
	}
	ds := NewSQLiteDatastore(config)
	defer ds.Close()

	key := "testKey"
	value := "testValue"
	intValue := 123
	floatValue := 123.45

	// Test Put.
	err := ds.Put(key, map[string]interface{}{"value": value, "intCol": intValue, "floatCol": floatValue})
	assert.NoError(t, err)

	// Test Get.
	result, err := ds.Get(key, []string{"value", "intCol", "floatCol"})
	assert.NoError(t, err)
	assert.Equal(t, value, result["value"].(string))
	assert.Equal(t, int64(intValue), result["intCol"].(int64))
	assert.Equal(t, floatValue, result["floatCol"].(float64))

	// Test Delete.
	err = ds.Delete(key)
	assert.NoError(t, err)

	// Test that the key is indeed deleted.
	result, err = ds.Get(key, []string{"value", "intCol", "floatCol"})
	assert.NoError(t, err)
	assert.Nil(t, result)

	// Test deleting a non-existent key.
	err = ds.Delete("non-existent key")
	assert.NoError(t, err)

	// Test Put with non-existent column.
	err = ds.Put(key, map[string]interface{}{"non_existent_column": value})
	assert.Error(t, err)

	// Test Put with wrong value type.
	// Note: we do not expect wrong value type will result in error, since Go database/sql will try to convert it automatically.
	err = ds.Put(key, map[string]interface{}{"value": 123, "intCol": "123", "floatCol": "123.45"})
	assert.NoError(t, err)

	// Test Get with non-existent column.
	_, err = ds.Get(key, []string{"non_existent_column"})
	assert.Error(t, err)

	// Test Get with non-existent key.
	_, err = ds.Get("non-existent key", []string{"value", "intCol", "floatCol"})
	assert.NoError(t, err)
}

func TestListAll(t *testing.T) {
	primaryKeyColumnName := "primaryKey"
	config := &Config{
		DBName:    ":memory:", // the memory database for testing purposes
		TableName: "TestListAll",
		ColumnConfig: map[string]string{
			primaryKeyColumnName: "text primary key not null",
			"value":              "text",
			"intCol":             "int",
			"floatCol":           "float",
		},
		PrimaryKeyColumnName: primaryKeyColumnName,
	}
	ds := NewSQLiteDatastore(config)
	defer ds.Close()

	// Insert some test data.
	testData := map[string]map[string]interface{}{
		"key1": {"value": "value1", "intCol": 1, "floatCol": 1.1},
		"key2": {"value": "value2", "intCol": 2, "floatCol": 2.2},
		"key3": {"value": "value3", "intCol": 3, "floatCol": 3.3},
	}
	for k, v := range testData {
		err := ds.Put(k, v)
		assert.NoError(t, err)
	}

	// Call ListAll and check the result.
	result, err := ds.ListAll()
	assert.NoError(t, err)
	for k, v := range testData {
		r, ok := result[k]
		assert.True(t, ok)
		assert.Equal(t, v["value"], r["value"].(string))
		assert.Equal(t, int64(v["intCol"].(int)), r["intCol"].(int64))
		assert.Equal(t, v["floatCol"].(float64), r["floatCol"].(float64))
	}

	// Delete all data.
	for k := range testData {
		err = ds.Delete(k)
		assert.NoError(t, err)
	}

	// Call ListAll again and check the result.
	result, err = ds.ListAll()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(result))

}
