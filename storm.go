package storm

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

// Storm is the main ORM struct that wraps a *sql.DB connection.
// It provides methods to perform basic CRUD operations (Insert, Update, Delete)
// and query building (via Query).
type Storm struct {
	db *sql.DB
}

// New creates a new Storm instance by opening a database connection using
// the provided driverName (e.g., "postgres", "mysql") and dsn (data source name).
// It verifies the connection with Ping and returns a Storm instance or an error.
func New(driverName, dsn string) (*Storm, error) {
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("Failed to open database connection: %v", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	return &Storm{db}, nil
}

// DB returns the underlying *sql.DB instance so you can execute raw queries if needed.
func (s *Storm) DB() *sql.DB {
	return s.db
}

// Insert inserts a struct record into the database.
// It uses reflection to read struct tags (`storm:"column:..."`) and build
// the appropriate SQL INSERT statement.
func (s *Storm) Insert(model interface{}) error {
	// val, its reflect the value of the struct that we passes
	val := reflect.ValueOf(model).Elem()
	// tipe, its reflect the datatype of this struct above
	tipe := val.Type()

	// columns, its all column that we need to insert represent the struct
	var columns []string
	// placeholders, is for value placeholder to insert the column
	var placeholders []string
	// values, is the values of column we want to insert
	var values []interface{}

	col := ""

	// below we loop the number of field in the struct
	for i := 0; i < val.NumField(); i++ {
		// field, we get the field of the struct, like name of struct, tag etc
		field := tipe.Field(i)
		// tag, we get the tag of struct like when we describe for example `json:""` in this below, we get the `storm:name` tag
		tag := field.Tag.Get("storm")

		// if the field is primary_key, then we skip that
		is_primary := strings.Contains(tag, "pk")
		is_column := strings.Contains(tag, "column")
		if is_primary {
			continue
		}

		// if in the tag we using column tag, for specify column name, then we use that to insert
		if is_column {
			col = strings.Split(tag, ":")[1]
		} else {
			// otheriwise we use, the field name
			col = strings.ToLower(field.Name)
		}

		placeHolderVal := fmt.Sprintf("$%d", i)

		columns = append(columns, col)
		placeholders = append(placeholders, placeHolderVal)
		values = append(values, val.Field(i).Interface())
	}

	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		strings.ToLower(tipe.Name()+"s"), // table name = struct name
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := s.db.Exec(q, values...)

	return err
}

// Update updates an existing struct record in the database based on its primary key.
// It reads `storm` struct tags and generates a dynamic SQL UPDATE statement.
// Only non-zero fields will be updated.
func (s *Storm) Update(model interface{}) error {
	val := reflect.ValueOf(model).Elem()
	tipe := val.Type()

	paramCount := 1

	var setClause []string  // this is for set clause column to update
	var vals []interface{}  // this for value that we want to update
	var pkField string      // this is field that primary_key
	var pkValue interface{} // this is for primary_key value to update
	var col string

	for i := 0; i < val.NumField(); i++ {
		field := tipe.Field(i)
		tag := field.Tag.Get("storm")

		is_primary := strings.Contains(tag, "pk")
		is_column := strings.Contains(tag, "column")

		if is_primary {
			pkField = field.Name
			pkValue = val.Field(i).Interface()
		} else {
			// if in the tag we using column tag, for specify column name, then we use that
			if is_column {
				col = strings.Split(tag, ":")[1]
			} else {
				// otheriwise we use, the field name
				col = strings.ToLower(field.Name)
			}
			if !val.Field(i).IsZero() {
				setClause = append(setClause, fmt.Sprintf("%s = $%d", col, i))
				vals = append(vals, val.Field(i).Interface())
				paramCount++
			}
		}
	}

	if pkField == "" {
		return fmt.Errorf("no primary key is found for update")
	}

	vals = append(vals, pkValue)
	q := fmt.Sprintf(`
		UPDATE %s SET %s WHERE %s = $%d
	`,
		strings.ToLower(tipe.Name()+"s"),
		strings.Join(setClause, ", "),
		pkField,
		paramCount,
	)
	_, err := s.db.Exec(q, vals...)
	return err
}

// Delete deletes a struct record from the database based on its primary key.
// It uses reflection to detect the primary key field (`storm:"pk"`) and
// generates a SQL DELETE statement.
func (s *Storm) Delete(model interface{}) error {
	val := reflect.ValueOf(model).Elem()
	tipe := val.Type()

	paramCount := 0

	var pkField string
	var pkValue interface{}
	var vals []interface{}

	for i := 0; i < val.NumField(); i++ {
		field := tipe.Field(i)
		tag := field.Tag.Get("storm")

		col := field.Name
		is_primary := strings.Contains(tag, "pk")
		if is_primary {
			pkField = col
			pkValue = val.Field(i).Interface()
			paramCount++
		}
	}

	vals = append(vals, pkValue)

	q := fmt.Sprintf(`
	DELETE FROM %s WHERE %s = $%d
	`,
		strings.ToLower(tipe.Name()+"s"),
		pkField,
		paramCount,
	)

	_, err := s.db.Exec(q, vals...)

	return err
}
