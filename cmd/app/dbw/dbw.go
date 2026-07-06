package db

import (
	"database/sql"
	"log"
	"sync"
)

type DBWrapper struct {
	*sql.DB
	mu    sync.RWMutex
	stmts map[string]*sql.Stmt
}

func NewDBWrapper(dsn string) *DBWrapper {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal("cannot open db", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	return &DBWrapper{
		DB:    db,
		stmts: make(map[string]*sql.Stmt),
	}
}

func (dbw *DBWrapper) Prep(query string) (*sql.Stmt, error) {
	dbw.mu.RLock()
	stmt, ok := dbw.stmts[query]
	dbw.mu.RUnlock()
	if ok {
		return stmt, nil
	}

	dbw.mu.Lock()
	defer dbw.mu.Unlock()

	if stmt, ok := dbw.stmts[query]; ok {
		return stmt, nil
	}

	stmt, err := dbw.Prepare(query)
	if err != nil {
		return nil, err
	}
	dbw.stmts[query] = stmt
	return stmt, nil
}
