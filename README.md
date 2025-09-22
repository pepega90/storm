
---
# Storm - Simple Tiny ORM for Go

`storm` (**S**imple **T**iny **ORM**) is a lightweight ORM library for Go built on top of `database/sql`.
It provides simple CRUD operations, query building and built-in pagination

If you like [GORM](https://gorm.io/) but want something much smaller and easier to understand, Storm is for you.

---

## Why Storm Instead of GORM?

| Feature | Storm | GORM |
|---------|-------|------|
| **Learning Curve** | Simple API, easy to master | Complex with many features |
| **Code Size** | ~500 lines, minimal dependencies | Large codebase |
| **Performance** | Lightweight, close to raw `database/sql` | More overhead |
| **Transparency** | Easy to debug and understand | "Magic" behind scenes |
| **Use Case** | Small to medium projects, learning | Enterprise with complex needs |

**Choose Storm if you want:**
- A minimal, educational ORM you can read and understand in an hour
- Something lightweight for small/medium projects
- To avoid complex migrations and advanced features you don't need
- Full control and transparency over SQL generation
- **Built-in pagination** - no need to manually write pagination logic
- Simple and predictable behavior

---

## Installation

```bash
go get github.com/pepega90/storm
```

---

## Quick Start

### 1. Define your model

```go
package models

type User struct {
	ID    int    `storm:"pk" json:"id"`                // primary key
	Name  string `storm:"column:name_user" json:"name"`
	Email string `storm:"column:email_user" json:"email"`
}
```

* Use `storm:"pk"` for the primary key.
* Use `storm:"column:xxx"` to map struct fields to DB columns.
* Table name is automatically pluralized (`User` → `users`).

---

### 2. Connect to the database

```go
package main

import (
	"log"

	_ "github.com/lib/pq"  // Currently only PostgreSQL is supported
	"github.com/pepega90/storm/storm"
)

func main() {
	dsn := "host=localhost user=postgres password=postgres dbname=storm_db port=5432 sslmode=disable"
	db, err := storm.New("postgres", dsn)
	if err != nil {
		log.Fatal("Storm is not initiated:", err.Error())
	}

	// use db.Insert, db.Update, db.Delete, db.From, etc.
}
```

**Note:** Currently only PostgreSQL is supported via `github.com/lib/pq`.

---

## Examples

### Insert

```go
user := &models.User{
	Name:  "aji",
	Email: "aji@handsome.com",
}
err := db.Insert(user)
if err != nil {
	log.Fatal("Error inserting data:", err.Error())
}
```

---

### Update

```go
user := &models.User{
	ID:    5,
	Name:  "aji",
	Email: "aji@handsome.com",
}
err := db.Update(user)
if err != nil {
	log.Fatal("Error updating data:", err.Error())
}
```

---

### Delete

```go
user := &models.User{
	ID: 5,
}
err := db.Delete(user)
if err != nil {
	log.Fatal("Error deleting data:", err.Error())
}
```

---

### Select (multiple rows)

```go
var users []models.User
err := db.
	From(&models.User{}).
	Limit(2).
	Select(&users, "id", "name_user", "email_user")
if err != nil {
	log.Fatal("Error selecting data:", err.Error())
}

fmt.Println("Users:", users)
```

---

### First (single row)

```go
var user models.User
err := db.
	From(&models.User{}).
	Where("id = $1", 14).
	First(&user)
if err != nil {
	log.Fatal("Error finding user:", err.Error())
}

fmt.Println("User:", user)
```

---

### Pagination (Built-in Feature)

**No need to write manual pagination logic!** Storm handles it for you:

```go
var users []models.User
var total, totalPages int
page := 2
pageSize := 3

err := db.
	From(&models.User{}).
	Paginate(&users, page, pageSize, &total, &totalPages, "id", "name_user")
if err != nil {
	log.Fatal("Error paginating users:", err.Error())
}
```

Storm automatically calculates:
- Total records count
- Total pages
- Applies proper LIMIT and OFFSET
- Returns the paginated data

---

## Current Limitations

- ✅ **Supported**: PostgreSQL via `github.com/lib/pq`
- ❌ **Not yet supported**: MySQL, SQLite, other databases
- ❌ **Not yet supported**: Joins, transactions, migrations

---

## Roadmap / TODO

* Support other databases (MySQL, SQLite)
* Support joins (`INNER JOIN`, `LEFT JOIN`)
* Support transactions
* Auto-migrations (like GORM)
* Better error handling

---

## License

MIT License © 2025 [pepega90](https://github.com/pepega90)

---
