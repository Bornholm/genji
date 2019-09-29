package genji_test

//go:generate genji -s User -f example_test.go

import (
	"fmt"
	"log"

	"github.com/asdine/genji"
	"github.com/asdine/genji/engine/memory"
	"github.com/asdine/genji/query"
)

type User struct {
	ID   int64 `genji:"pk"`
	Name string
	Age  uint32
}

func Example() {
	ng := memory.NewEngine()
	db, err := genji.New(ng)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	dbx := query.NewSQLDB(db)

	_, err = dbx.Exec("CREATE TABLE user IF NOT EXISTS")
	if err != nil {
		log.Fatal(err)
	}

	_, err = dbx.Exec("INSERT INTO user (ID, Name, Age) VALUES (?, ?, ?)", 10, "foo", 15)
	if err != nil {
		log.Fatal(err)
	}

	_, err = dbx.Exec("INSERT INTO user RECORDS ?, ?", &User{ID: 1, Name: "bar", Age: 100}, &User{ID: 2, Name: "baz"})
	if err != nil {
		log.Fatal(err)
	}

	rows, err := dbx.Query("SELECT * FROM user WHERE Name = ?", "bar")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var u User
		err = rows.Scan(&u)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(u)
	}

	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	// Output: {1 bar 100}
}
