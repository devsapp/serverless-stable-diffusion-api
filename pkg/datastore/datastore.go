package datastore

type DatastoreType string

const (
	SQLite     DatastoreType = "sqlite"
	MySQL      DatastoreType = "mysql"
	TableStore DatastoreType = "tableStore"
)

type Config struct {
	Type                 DatastoreType // the datastore type
	DBName               string        // the database name
	TableName            string
	ColumnConfig         map[string]string // map of column name to column type
	PrimaryKeyColumnName string
	TimeToAlive          int
	MaxVersion           int
}

type Datastore interface {
	// Put inserts or updates the column values in the datastore.
	// It takes a key and a map of column names to values, and returns an error if the operation failed.
	Put(key string, values map[string]interface{}) error

	// Update the partial column values.
	// It tasks a key and a map of column names to values, and returns an error if the operation failed.
	Update(key string, values map[string]interface{}) error

	// Get retrieves the column values from the datastore.
	// It takes a key and a slice of column names, and returns a map of column names to values,
	// along with an error if the operation failed.
	// If the key does not exist, the returned map and error are both nil.
	Get(key string, columns []string) (map[string]interface{}, error)

	//Put(key string, value string) error
	//Get(key string) (string, error)

	// Delete removes a value from the datastore.
	// It takes a key, and returns an error if the operation failed.
	// Note: delete a non-existent key will not return an error.
	Delete(key string) error

	// ListAll read all data from the datastore.
	// It takes a list of column name, and  return a nested map, which means map[primaryKey]map[columanName]columanValue.
	// Note: since it reads all data and store them in memory, so do not call this function on a large datastore.
	ListAll(columns []string) (map[string]map[string]interface{}, error)

	// Close close the datastore.
	Close() error
}
