package controllers

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
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

func parseDuration(durstr string) (time.Duration, error) {
	if strings.Contains(durstr, "d") {
		dint, err := strconv.Atoi(durstr[:len(durstr)-1])
		if err != nil {
			return time.Second, errors.Wrap(err, "failed to atoi")
		}
		durstr = fmt.Sprintf("%dh", dint*24)
	}
	duration, err := time.ParseDuration(durstr)
	if err != nil {
		return time.Second, errors.Wrapf(err, "duration '%s' is invalid", durstr)
	}
	return duration, nil
}
