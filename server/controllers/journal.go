package controllers

import (
	"context"

	pb "github.com/mikerjacobi/notify-app/server/rpc"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

func (s *NotifyAppServer) insertJournal(ctx context.Context, db Database, j *pb.Journal) error {
	stmt, err := db.Prepare(`
		INSERT INTO journals (journal_id, comms_id, phone_number, title, entry, created, updated)
		VALUES (?, ?, ?, ?, ?, NOW(6), NOW(6))
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare")
	}
	j.JournalId = uuid.NewV4().String()
	if _, err = stmt.Exec(j.JournalId, j.CommsId, j.PhoneNumber, j.Title, j.Entry); err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (s *NotifyAppServer) updateJournal(ctx context.Context, db Database, j *pb.Journal) error {
	stmt, err := db.Prepare(`
		UPDATE journals SET updated=NOW(6), entry=?
		WHERE journal_id=? AND phone_number=?
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare")
	}
	if _, err = stmt.Exec(j.Entry, j.JournalId, j.PhoneNumber); err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (s *NotifyAppServer) deleteJournal(ctx context.Context, db Database, j *pb.Journal) error {
	stmt, err := db.Prepare(`
		DELETE FROM journals
		WHERE journal_id=? AND phone_number=?
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare")
	}
	if _, err = stmt.Exec(j.JournalId, j.PhoneNumber); err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (s *NotifyAppServer) getJournalEntries(ctx context.Context, db Database, phoneNumber string) ([]*pb.Journal, error) {
	stmt, err := db.Prepare(`SELECT journal_id,comms_id,phone_number,title,entry,created,updated 
		FROM journals 
		WHERE phone_number=?
		ORDER BY created DESC`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare")
	}
	rows, err := stmt.Query(phoneNumber)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()
	entries := []*pb.Journal{}
	for rows.Next() {
		j := &pb.Journal{}
		if err := rows.Scan(&j.JournalId, &j.CommsId, &j.PhoneNumber, &j.Title, &j.Entry, &j.Created, &j.Updated); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		entries = append(entries, j)
	}
	return entries, nil
}
