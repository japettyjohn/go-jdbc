package gojdbc

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"
)

const testConnString = "tcp://localhost:7777/"

type Test struct {
	Id      int64
	Title   string
	Age     int64
	Created time.Time
}

func TestJDBCBasic(t *testing.T) {

	db, err := sql.Open("jdbc", testConnString)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec("drop table if exists test;")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec("create table test(Id int auto_increment primary key, Title varchar(255), Age int, Created datetime)")
	if err != nil {
		t.Fatal(err)
	}

	// Parallel inserts
	testTime := time.Now().Round(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	stmt, err := db.Prepare("insert into test(Title,Age,Created) values(?,?,?)")
	defer stmt.Close()

	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func(i int) {
			defer wg.Done()
			r, err := stmt.Exec(fmt.Sprintf("The %d", i), i, testTime)
			if err != nil {
				t.Fatal(err)
			}
			_, err = r.RowsAffected()
			if err != nil {
				t.Fatal(err)
			}
		}(i)
	}
	wg.Wait()

	// Select rows
	rows, err := db.Query("select * from test")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		i = i + 1
		r := Test{}
		if e := rows.Scan(&r.Id, &r.Title, &r.Age, &r.Created); e != nil {
			t.Fatal(e)
		}
		expectedTitle := fmt.Sprintf("The %d", r.Age)
		switch {
		case r.Id == 0:
			t.Fatalf("Invalid Id: %+v", r)
		case r.Title != expectedTitle:
			t.Fatalf("Expected Title %s but got %s", expectedTitle, r.Title)
		case !r.Created.Equal(testTime):
			t.Fatalf("Expected time %v but got %v", testTime, r.Created)

		}
	}
	if i < 100 {
		t.Fatalf("Expected 100 but got %d.", i)
	}

}

func TestJDBCWithTransactions(t *testing.T) {
	fatalErr := func(e error) {
		if e != nil {
			t.Fatal(e)
		}
	}
	db, err := sql.Open("jdbc", testConnString)
	fatalErr(err)
	defer db.Close()

	_, err = db.Exec("drop table if exists test;")
	fatalErr(err)

	_, err = db.Exec("create table test(Id int auto_increment primary key, Title varchar(255), Age int, Created datetime)")
	fatalErr(err)

	// Parallel inserts
	testTime := time.Now().Round(time.Second)
	tx, err := db.Begin()
	fatalErr(err)
	stmt, err := tx.Prepare("insert into test(Title,Age,Created) values(?,?,?)")
	defer stmt.Close()

	fatalErr(err)
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func(i int) {
			defer wg.Done()
			r, err := stmt.Exec(fmt.Sprintf("The %d", i), i, testTime)
			fatalErr(err)
			_, err = r.RowsAffected()
			fatalErr(err)
		}(i)
	}
	wg.Wait()
	fatalErr(tx.Commit())

	// Select rows
	rows, err := db.Query("select * from test")
	fatalErr(err)
	defer rows.Close()

	i := 0
	for rows.Next() {
		i = i + 1
		r := Test{}
		if e := rows.Scan(&r.Id, &r.Title, &r.Age, &r.Created); e != nil {
			t.Fatal(e)
		}
		expectedTitle := fmt.Sprintf("The %d", r.Age)
		switch {
		case r.Id == 0:
			t.Fatalf("Invalid Id: %+v", r)
		case r.Title != expectedTitle:
			t.Fatalf("Expected Title %s but got %s", expectedTitle, r.Title)
		case !r.Created.Equal(testTime):
			t.Fatalf("Expected time %v but got %v", testTime, r.Created)

		}
	}
	if i < 100 {
		t.Fatalf("Expected 100 but got %d.", i)
	}

}

func TestJDBCWithQueryTimeout(t *testing.T) {
	db, err := sql.Open("jdbc", fmt.Sprintf("%s%s", testConnString, "?queryTimeout=1"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("drop table if exists test;")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec("create table test(Id int auto_increment primary key, Title varchar(255), Age int, Created datetime)")
	if err != nil {
		t.Fatal(err)
	}

	// Parallel inserts
	testTime := time.Now().Round(time.Second)
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	stmt, err := tx.Prepare("insert into test(Title,Age,Created) values(?,?,?)")
	defer stmt.Close()

	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	groupSize := 1000
	wg.Add(groupSize)
	for i := 0; i < groupSize; i++ {
		go func(i int) {
			defer wg.Done()
			r, err := stmt.Exec(fmt.Sprintf("The %d", i), i, testTime)
			if err != nil {
				t.Fatal(err)
			}
			_, err = r.RowsAffected()
			if err != nil {
				t.Fatal(err)
			}
		}(i)
	}
	wg.Wait()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	// Select rows
	rows, err := db.Query("select t.* from test t join test t2")
	if err == nil {
		t.Fatalf("Expected query time out")
	}

	// This should work
	if rows, err = db.Query("select t.* from test t"); err != nil {
		t.Fatal(err)
	}

	defer rows.Close()

	i := 0
	for rows.Next() {
		i = i + 1
		r := Test{}
		if e := rows.Scan(&r.Id, &r.Title, &r.Age, &r.Created); e != nil {
			t.Fatal(e)
		}
		expectedTitle := fmt.Sprintf("The %d", r.Age)
		switch {
		case r.Id == 0:
			t.Fatalf("Invalid Id: %+v", r)
		case r.Title != expectedTitle:
			t.Fatalf("Expected Title %s but got %s", expectedTitle, r.Title)
		case !r.Created.Equal(testTime):
			t.Fatalf("Expected time %v but got %v", testTime, r.Created)

		}
	}
	if i < groupSize {
		t.Fatalf("Expected %d but got %d.", groupSize, i)
	}

}

func TestJDBCSystemStatus(t *testing.T) {
	fatalErr := func(e error) {
		if e != nil {
			t.Fatal(e)
		}
	}
	db, err := sql.Open("jdbc", testConnString)
	fatalErr(err)
	defer db.Close()

	if _, err = db.Exec("drop table if exists test;"); err != nil {
		t.Fatal(err)
	}

	if _, err = db.Exec("create table test(Id int auto_increment primary key, Title varchar(255), Age int, Created datetime)"); err != nil {
		t.Fatal(err)
	}

	// Parallel inserts
	testTime := time.Now().Round(time.Second)

	stmt, err := db.Prepare("insert into test(Title,Age,Created) values(?,?,?)")
	fatalErr(err)
	defer stmt.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for i := 0; i < 10; i++ {
			if r, err := stmt.Exec(fmt.Sprintf("The %d", i), i, testTime); err != nil {
				t.Fatal(err)
			} else {
				if _, err = r.RowsAffected(); err != nil {
					t.Fatal(err)
				}
			}
		}
		wg.Done()
	}()

	if status, e := ServerStatus(testConnString); e != nil {
		t.Fatal(e)
	} else {
		t.Log(status)
	}
	wg.Wait()

}

func TestJDBCWithReadDeadline(t *testing.T) {
	db, err := sql.Open("jdbc", fmt.Sprintf("%s?%s=%d", testConnString, paramReadDeadline, 10))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("drop table if exists test;")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec("create table test(Id int auto_increment primary key, Title varchar(255), Age int, Created datetime)")
	if err != nil {
		t.Fatal(err)
	}

	// Parallel inserts
	testTime := time.Now().Round(time.Second)
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	stmt, err := tx.Prepare("insert into test(Title,Age,Created) values(?,?,?)")
	defer stmt.Close()

	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	groupSize := 1000
	wg.Add(groupSize)
	for i := 0; i < groupSize; i++ {
		go func(i int) {
			defer wg.Done()
			r, err := stmt.Exec(fmt.Sprintf("The %d", i), i, testTime)
			if err != nil {
				t.Fatal(err)
			}
			_, err = r.RowsAffected()
			if err != nil {
				t.Fatal(err)
			}
		}(i)
	}
	wg.Wait()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	// This should work

	rows, err := db.Query("select t.* from test t")
	if err != nil {
		t.Fatal(err)
	}

	defer rows.Close()

	i := 0
	for rows.Next() {
		i = i + 1
		r := Test{}
		if e := rows.Scan(&r.Id, &r.Title, &r.Age, &r.Created); e != nil {
			t.Fatal(e)
		}
		expectedTitle := fmt.Sprintf("The %d", r.Age)
		switch {
		case r.Id == 0:
			t.Fatalf("Invalid Id: %+v", r)
		case r.Title != expectedTitle:
			t.Fatalf("Expected Title %s but got %s", expectedTitle, r.Title)
		case !r.Created.Equal(testTime):
			t.Fatalf("Expected time %v but got %v", testTime, r.Created)

		}
	}
	if i < groupSize {
		t.Fatalf("Expected %d but got %d.", groupSize, i)
	}

}

func TestJDBCWithFetchSize(t *testing.T) {
	db, err := sql.Open("jdbc", fmt.Sprintf("%s?%s=%d", testConnString, paramFetchSize, 500))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("drop table if exists test;")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec("create table test(Id int auto_increment primary key, Title varchar(255), Age int, Created datetime)")
	if err != nil {
		t.Fatal(err)
	}

	// Parallel inserts
	testTime := time.Now().Round(time.Second)
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	stmt, err := tx.Prepare("insert into test(Title,Age,Created) values(?,?,?)")
	defer stmt.Close()

	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	groupSize := 1000
	wg.Add(groupSize)
	for i := 0; i < groupSize; i++ {
		go func(i int) {
			defer wg.Done()
			r, err := stmt.Exec(fmt.Sprintf("The %d", i), i, testTime)
			if err != nil {
				t.Fatal(err)
			}
			_, err = r.RowsAffected()
			if err != nil {
				t.Fatal(err)
			}
		}(i)
	}
	wg.Wait()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	// This should work

	rows, err := db.Query("select t.* from test t")
	if err != nil {
		t.Fatal(err)
	}

	defer rows.Close()

	i := 0
	for rows.Next() {
		i = i + 1
		r := Test{}
		if e := rows.Scan(&r.Id, &r.Title, &r.Age, &r.Created); e != nil {
			t.Fatal(e)
		}
		expectedTitle := fmt.Sprintf("The %d", r.Age)
		switch {
		case r.Id == 0:
			t.Fatalf("Invalid Id: %+v", r)
		case r.Title != expectedTitle:
			t.Fatalf("Expected Title %s but got %s", expectedTitle, r.Title)
		case !r.Created.Equal(testTime):
			t.Fatalf("Expected time %v but got %v", testTime, r.Created)

		}
	}
	if i < groupSize {
		t.Fatalf("Expected %d but got %d.", groupSize, i)
	}

}
