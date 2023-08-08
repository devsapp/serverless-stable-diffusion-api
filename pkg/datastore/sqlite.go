package datastore

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteDatastore struct {
	db     *sql.DB
	config *Config
}

func NewSQLiteDatastore(config *Config) *SQLiteDatastore {
	db, err := sql.Open("sqlite3", config.DBName)
	if err != nil {
		panic(fmt.Errorf("failed to open database: %v", err))
	}

	// Create table if it doesn't exist.
	columnDefs := make([]string, 0, len(config.ColumnConfig))
	for name, typ := range config.ColumnConfig {
		columnDefs = append(columnDefs, fmt.Sprintf("%s %s", name, typ))
	}
	query := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (%s)",
		config.TableName,
		strings.Join(columnDefs, ", "),
	)
	_, err = db.Exec(query)
	if err != nil {
		panic(fmt.Errorf("failed to create table %s: %v", config.TableName, err))
	}
	return &SQLiteDatastore{
		db:     db,
		config: config,
	}
}

func (ds *SQLiteDatastore) Close() error {
	return ds.db.Close()
}

func (ds *SQLiteDatastore) Get(key string, columns []string) (map[string]interface{}, error) {
	row := ds.db.QueryRow(
		fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?",
			strings.Join(columns, ", "), ds.config.TableName, ds.config.PrimaryKeyColumnName),
		key,
	)

	// Prepare a slice to hold the values.
	values := make([]interface{}, len(columns))
	for i, column := range columns {
		// We use the type information stored in the Config to create a variable of the correct type.
		var value interface{}
		switch ds.config.ColumnConfig[column] {
		case "text":
			value = new(string)
		case "int":
			// For simplicity, we use int64 for all integers.
			value = new(int64)
		case "float":
			value = new(float64)
		default:
			// If the column type is not supported, we return an error.
			return nil, fmt.Errorf("unsupported column type: %s", ds.config.ColumnConfig[column])
		}
		values[i] = value
	}

	// Scan the result into the values slice.
	err := row.Scan(values...)
	if err != nil {
		if err == sql.ErrNoRows {
			// There is no row with the given key.
			return nil, nil
		}
		return nil, err
	}

	// Prepare the result map and fill it with values.
	result := make(map[string]interface{})
	for i, column := range columns {
		// We use the reflect package to dereference the pointer.
		value := reflect.ValueOf(values[i]).Elem().Interface()
		result[column] = value
	}

	return result, nil
}

func (ds *SQLiteDatastore) Put(key string, values map[string]interface{}) error {
	columns := []string{ds.config.PrimaryKeyColumnName}
	placeholders := []string{"?"}
	args := []interface{}{key}
	for column, value := range values {
		columns = append(columns, column)
		placeholders = append(placeholders, "?")
		args = append(args, value)
	}
	query := fmt.Sprintf(
		"INSERT OR REPLACE INTO %s (%s) VALUES (%s)",
		ds.config.TableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
	_, err := ds.db.Exec(query, args...)
	return err
}

func (ds *SQLiteDatastore) Delete(key string) error {
	_, err := ds.db.Exec(
		fmt.Sprintf(
			"DELETE FROM %s WHERE %s = ?", ds.config.TableName, ds.config.PrimaryKeyColumnName),
		key)
	return err
}

func (ds *SQLiteDatastore) ListAll() (map[string]map[string]interface{}, error) {
	rows, err := ds.db.Query(fmt.Sprintf("SELECT * FROM %s", ds.config.TableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	results := make(map[string]map[string]interface{})
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		err := rows.Scan(columnPointers...)
		if err != nil {
			return nil, err
		}

		m := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			m[colName] = *val
		}

		key := m[ds.config.PrimaryKeyColumnName].(string)
		results[key] = m
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
