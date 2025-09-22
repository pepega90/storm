package storm

import (
	"fmt"
	"math"
	"reflect"
	"strings"
)

// Query represents a SQL query builder for SELECT operations.
// It stores the target table, conditions, and pagination options.
type Query struct {
	storm         *Storm        // pointer of the orm struct
	table         string        // table name of the that we want to query, we get it from reflect typeof
	where         string        // where condition, so what field we want to use to find
	whereArgument []interface{} // where argument, so we passes the value to the where above
	limit         int           // limit, use for limit the number of return data from the database
}

// From initializes a query from the given model struct.
// It infers the table name based on struct type (structName + "s").
func (s *Storm) From(model interface{}) *Query {
	tipe := reflect.TypeOf(model).Elem().Name()
	return &Query{
		storm: s,
		table: strings.ToLower(tipe + "s"),
	}
}

// Where adds a WHERE condition with optional arguments to the query.
// Example: .Where("id = $1", 10)
func (q *Query) Where(condition string, args ...interface{}) *Query {
	q.where = condition
	q.whereArgument = args
	return q
}

// Limit adds a LIMIT clause to the query.
func (q *Query) Limit(n int) *Query {
	q.limit = n
	return q
}

// First executes the query and maps the first matching row into dest struct.
// You can optionally pass column names to select specific fields.
func (q *Query) First(dest interface{}, queryCol ...string) error {
	table := q.table

	isQueryColExist := len(queryCol) > 0
	selectedCols := "*"
	if isQueryColExist {
		selectedCols = strings.Join(queryCol, ",")
	}

	query := fmt.Sprintf("SELECT %s FROM %s", selectedCols, table)

	var args []interface{}
	// check if we have WHERE clause
	if q.where != "" {
		// if so, then we append the WHERE clause, and query WHERE like for example ID = ?
		query += " WHERE " + q.where
		// below we append the WHERE argument value, above the "?" it will become ID we find
		args = append(args, q.whereArgument...)
	}
	query += fmt.Sprintf(" LIMIT %d", 1)

	rows, err := q.storm.db.Query(query, args...)
	if err != nil {
		return err
	}

	columnNames, _ := rows.Columns()

	vals := make([]interface{}, len(columnNames))
	for rows.Next() {
		ptrs := make([]interface{}, len(columnNames))

		for i := range vals {
			ptrs[i] = &vals[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			return err
		}

	}

	newStructDestination := reflect.ValueOf(dest).Elem()
	typeInfo := newStructDestination.Type()
	ht := map[string]string{}
	for i := 0; i < newStructDestination.NumField(); i++ {
		field := typeInfo.Field(i)

		structFieldName := strings.ToLower(field.Name)

		if val, ok := field.Tag.Lookup("storm"); ok {
			stormTagSplit := strings.Split(val, ":")
			if len(stormTagSplit) == 2 {
				structFieldName = stormTagSplit[1]
			}
		}

		ht[structFieldName] = field.Name
	}

	for i, col := range columnNames {
		structFieldName, ok := ht[col]
		if !ok {
			continue
		}

		field := newStructDestination.FieldByName(structFieldName)

		if !field.IsValid() {
			continue
		}

		// in here we set the value, from database
		err := setFieldValue(field, vals[i])
		if err != nil {
			return err
		}
	}

	return nil
}

// Select executes the query and maps all rows into a slice of structs.
// Example usage: var users []User; db.From(&User{}).Select(&users)
func (q *Query) Select(dest interface{}, queryCol ...string) error {
	// below we got tipe of sturct, we do Elem() twice to get that, cause if we only do Elem() one, we got slice value, so for example User struct, we got []User
	tipe := reflect.TypeOf(dest).Elem().Elem()
	table := q.table

	isQueryColExist := len(queryCol) > 0
	selectedCols := "*"
	if isQueryColExist {
		selectedCols = strings.Join(queryCol, ",")
	}

	query := fmt.Sprintf("SELECT %s FROM %s", selectedCols, table)

	var args []interface{}
	// check if we have WHERE clause
	if q.where != "" {
		// if so, then we append the WHERE clause, and query WHERE like for example ID = ?
		query += " WHERE " + q.where
		// below we append the WHERE argument value, above the "?" it will become ID we find
		args = append(args, q.whereArgument...)
	}

	// check if limit apply
	if q.limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", q.limit)
	}

	rows, err := q.storm.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	// below we got list of the column name
	cols, _ := rows.Columns()
	// sliceVal, we reflect value of dest params, it will be empty slice since we will fill it with value of the struct we do reflectTypeOf(dest).Elem().Elem() above
	// for example if dest is *[]User then it will be []User
	sliceVal := reflect.ValueOf(dest).Elem()

	for rows.Next() {
		/*
			vals, is for actual value in the database
			ptrs, is for pointing to each value in vals[i] at i index
			for example if vals have 3 column (id name email), then it will be:
			vals = {nil nil nil}
			ptrs = {nil nil nil}
		*/
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))

		// then we use ptrs at index i we give pointer of value
		// so ptrs will be ptrs = {&vals[0], &vals[1], &vals[2]}
		for i := range vals {
			ptrs[i] = &vals[i]
		}

		// after that we scan it, the vals with get the data since its pointer to ptrs at index i
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}

		// we create struct of type reflect.TypeOf above
		newStruct := reflect.New(tipe).Elem()
		newStructType := newStruct.Type()

		// so below we create key value pair, of column name and field in the struct. cause if we change the column name in the db, its will not following the struct field name anymore.
		/*
			for example

			type User struct {
				Name string
				Email string
			}

			in database is
			| id | name_user | email_user |

			so is not match right, so hash_map will look like this

			{
				name_user: Name,
				email_user: Email
			}

			like so, so if we alter or rename the name of the field in the DB, we still got that
		*/

		ht := map[string]string{}
		for i := 0; i < newStructType.NumField(); i++ {
			field := newStructType.Field(i)

			col := strings.ToLower(field.Name)

			// if "storm" tag exists, extract "column:xxx"
			if tag, ok := field.Tag.Lookup("storm"); ok {
				parts := strings.Split(tag, ":")
				if len(parts) == 2 && parts[0] == "column" {
					col = parts[1]
				}
			}
			ht[col] = field.Name
		}

		for i, col := range cols {
			structFieldName, ok := ht[col]
			if !ok {
				continue
			}

			// FieldByName, its find name that match with col name from cols, its case-insensitive
			field := newStruct.FieldByName(structFieldName)

			if !field.IsValid() {
				continue
			}

			err := setFieldValue(field, vals[i])
			if err != nil {
				return fmt.Errorf("error setting field %s: %v", ht[col], err)
			}
		}
		sliceVal.Set(reflect.Append(sliceVal, newStruct))
	}
	return nil
}

// Paginate executes the query with pagination support.
// It fills dest with results, and also updates total and totalPages values.
func (q *Query) Paginate(dest interface{}, page, pageSize int, total *int, totalPages *int, queryCol ...string) error {
	tipe := reflect.TypeOf(dest).Elem().Elem()
	if page < 1 {
		page = 1
	}

	if pageSize <= 0 {
		pageSize = 1
	}

	// count total of data
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", q.table)
	if err := q.storm.db.QueryRow(countQuery).Scan(total); err != nil {
		return err
	}

	// calculate total pages
	*totalPages = int(math.Ceil(float64(*total) / float64(pageSize)))

	isQueryColExist := len(queryCol) > 0
	selectedCols := "*"
	if isQueryColExist {
		selectedCols = strings.Join(queryCol, ",")
	}

	offset := (page - 1) * pageSize
	query := fmt.Sprintf("SELECT %s FROM %s ORDER BY id LIMIT $1 OFFSET $2", selectedCols, q.table)

	rows, err := q.storm.db.Query(query, pageSize, offset)
	if err != nil {
		return err
	}
	defer rows.Close()

	// below we got list of the column name
	cols, _ := rows.Columns()
	// sliceVal, we reflect value of dest params, it will be empty slice since we will fill it with value of the struct we do reflectTypeOf(dest).Elem().Elem() above
	// for example if dest is *[]User then it will be []User
	sliceVal := reflect.ValueOf(dest).Elem()

	for rows.Next() {
		/*
			vals, is for actual value in the database
			ptrs, is for pointing to each value in vals[i] at i index
			for example if vals have 3 column (id name email), then it will be:
			vals = {nil nil nil}
			ptrs = {nil nil nil}
		*/
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))

		// then we use ptrs at index i we give pointer of value
		// so ptrs will be ptrs = {&vals[0], &vals[1], &vals[2]}
		for i := range vals {
			ptrs[i] = &vals[i]
		}

		// after that we scan it, the vals with get the data since its pointer to ptrs at index i
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}

		// we create struct of type reflect.TypeOf above
		newStruct := reflect.New(tipe).Elem()
		newStructType := newStruct.Type()

		// so below we create key value pair, of column name and field in the struct. cause if we change the column name in the db, its will not following the struct field name anymore.
		/*
			for example

			type User struct {
				Name string
				Email string
			}

			in database is
			| id | name_user | email_user |

			so is not match right, so hash_map will look like this

			{
				name_user: Name,
				email_user: Email
			}

			like so, so if we alter or rename the name of the field in the DB, we still got that
		*/

		ht := map[string]string{}
		for i := 0; i < newStructType.NumField(); i++ {
			field := newStructType.Field(i)

			col := strings.ToLower(field.Name)

			// if "storm" tag exists, extract "column:xxx"
			if tag, ok := field.Tag.Lookup("storm"); ok {
				parts := strings.Split(tag, ":")
				if len(parts) == 2 && parts[0] == "column" {
					col = parts[1]
				}
			}
			ht[col] = field.Name
		}

		for i, col := range cols {
			structFieldName, ok := ht[col]
			if !ok {
				continue
			}

			// FieldByName, its find name that match with col name from cols, its case-insensitive
			field := newStruct.FieldByName(structFieldName)

			if !field.IsValid() {
				continue
			}

			err := setFieldValue(field, vals[i])
			if err != nil {
				return fmt.Errorf("error setting field %s: %v", ht[col], err)
			}
		}
		sliceVal.Set(reflect.Append(sliceVal, newStruct))
	}
	return nil
}

// setFieldValue, private function for set value for each struct field have 2 parameter field is the field we want to set the  value, and value itself
func setFieldValue(field reflect.Value, value interface{}) error {
	if value == nil {
		return nil
	}

	fieldType := field.Type()
	val := reflect.ValueOf(value)

	if val.Type().AssignableTo(fieldType) {
		field.Set(val)
		return nil
	}

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case int64:
			field.SetInt(v)
		case int32:
			field.SetInt(int64(v))
		case int:
			field.SetInt(int64(v))
		case float64:
			field.SetInt(int64(v))
		default:
			return fmt.Errorf("cannot convert %T to %v", value, fieldType)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := value.(type) {
		case int64:
			field.SetUint(uint64(v))
		case int32:
			field.SetUint(uint64(v))
		case int:
			field.SetUint(uint64(v))
		case float64:
			field.SetUint(uint64(v))
		default:
			return fmt.Errorf("cannot convert %T to %v", value, fieldType)
		}

	case reflect.Float32, reflect.Float64:
		switch v := value.(type) {
		case float64:
			field.SetFloat(v)
		case int64:
			field.SetFloat(float64(v))
		case int:
			field.SetFloat(float64(v))
		default:
			return fmt.Errorf("cannot convert %T to %v", value, fieldType)
		}

	case reflect.String:
		switch v := value.(type) {
		case string:
			field.SetString(v)
		case []byte:
			field.SetString(string(v))
		default:
			return fmt.Errorf("cannot convert %T to string", value)
		}

	case reflect.Bool:
		switch v := value.(type) {
		case bool:
			field.SetBool(v)
		case int64:
			field.SetBool(v != 0)
		default:
			return fmt.Errorf("cannot convert %T to bool", value)
		}

	default:
		return fmt.Errorf("unsupported field type: %v", fieldType)
	}

	return nil
}
