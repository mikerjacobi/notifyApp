package controllers

import (
	"database/sql"
	"time"

	"github.com/sirupsen/logrus"
)

type contextKey string

var (
	userKey contextKey = "user"
)

type Database interface {
	Prepare(query string) (*sql.Stmt, error)
}

func Contains(slice []string, ele string) bool {
	for _, val := range slice {
		if val == ele {
			return true
		}
	}
	return false
}

func now(db Database) time.Time {
	stmt, err := db.Prepare(`SELECT NOW() as now`)
	if err != nil {
		logrus.Errorf("failed to prepare NOW: %s", err)
		return time.Now()
	}
	rows, err := stmt.Query()
	if err != nil {
		logrus.Errorf("failed to select NOW: %s", err)
		return time.Now()
	}
	defer rows.Close()
	if !rows.Next() {
		logrus.Errorf("no NOW rows: %s", err)
		return time.Now()
	}

	var nowStr string
	if err := rows.Scan(&nowStr); err != nil {
		logrus.Errorf("failed to scan NOW: %s", err)
		return time.Now()
	}
	now, err := time.Parse(timeFormat, nowStr)
	if err != nil {
		logrus.Errorf("failed to parse NOW: %s", err)
		return time.Now()
	}
	return now
}
