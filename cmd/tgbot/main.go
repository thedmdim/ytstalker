package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"strings"

	"zombiezen.com/go/sqlite/sqlitex"
	"github.com/google/uuid"
)

func main() {

	// prepare db
	db, err := sqlitex.NewPool(os.Getenv("DSN"), sqlitex.PoolOptions{PoolSize: 100})
	if err != nil {
		log.Fatal("cannot open db", err)
	}

	conn := db.Get(context.Background())
	if err := sqlitex.ExecuteScript(conn, CreateTablesIfNotExists, nil); err != nil {
		log.Fatal("cannot create db: ", err)
	}

	// populate db
	file, err := os.Open("cams.csv")
    if err != nil {
        log.Fatal("no cams.csv file")
    }
    defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		log.Println(line)
		line = strings.TrimSpace(line)
		columns := strings.Split(line, ",")

		id, err := uuid.NewRandom()
		if err != nil {
			log.Fatal("uuid.NewRandom():", err)
		}
		addr := columns[0]
		adminka := columns[2]
		stream := columns[1]
		model := columns[3]
		country := strings.ToLower(columns[4])

		stmt := conn.Prep(`
			INSERT INTO cams (id, addr, adminka, stream, model, country)
			VALUES(?, ?, ?, ?, ?, ?)
		`)

		stmt.BindBytes(1, id[:])
		stmt.BindText(2, addr)
		stmt.BindText(3, adminka)
		stmt.BindText(4, stream)
		stmt.BindText(5, model)
		stmt.BindText(6, country)

		_, err = stmt.Step()
		if err != nil {
			log.Fatal("cannot step:", err)
		}
		err = stmt.ClearBindings()
		if err != nil {
			log.Fatal("cannot clear bindings:", err)
		}
		err = stmt.Reset()
		if err != nil {
			log.Fatal("cannot reset stmt:", err)
		}
	}

	log.Println("end loop")

	db.Put(conn)
	err = db.Close()
	if err != nil {
		log.Fatal("cannot close db", err)
	}
}