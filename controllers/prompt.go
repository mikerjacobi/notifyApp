package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/asaskevich/govalidator"
	gpb "github.com/golang/protobuf/ptypes/empty"
	pb "github.com/mikerjacobi/notify-app/server/rpc"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/twitchtv/twirp"
)

func (s *NotifyAppServer) getPrompt(ctx context.Context, db Database, promptID string) (*pb.Prompt, error) {
	stmt, err := db.Prepare(`
		SELECT name,type,template
		FROM prompts 
		WHERE prompt_id=?`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare")
	}
	rows, err := stmt.Query(promptID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()
	prompt := &pb.Prompt{PromptId: promptID}
	if rows.Next() {
		if err := rows.Scan(&prompt.Name, &prompt.Type, &prompt.Template); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
	} else {
		return nil, fmt.Errorf("prompt id '%s' not found", promptID)
	}
	return prompt, nil
}

func (s *NotifyAppServer) getPrompts(ctx context.Context, db Database) ([]*pb.Prompt, error) {
	stmt, err := db.Prepare(`
		SELECT prompt_id,name,type,template
		FROM prompts`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare")
	}
	rows, err := stmt.Query()
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()
	prompts := []*pb.Prompt{}
	for rows.Next() {
		p := &pb.Prompt{}
		if err := rows.Scan(&p.PromptId, &p.Name, &p.Type, &p.Template); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		prompts = append(prompts, p)
	}
	return prompts, nil
}

func (s *NotifyAppServer) insertPrompt(ctx context.Context, db Database, p *pb.Prompt) error {
	p.PromptId = uuid.NewV4().String()
	stmt, err := db.Prepare(`
		INSERT INTO prompts (prompt_id, name, type, template, created, updated)
		VALUES (?, ?, ?, ?, NOW(6), NOW(6))
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare")
	}
	if _, err = stmt.Exec(p.PromptId, p.Name, p.Type, p.Template); err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (s *NotifyAppServer) AddUserPrompt(ctx context.Context, req *pb.UserPrompt) (*gpb.Empty, error) {
	if arg, err := s.validateAddUserPrompt(ctx, req); err != nil {
		logrus.Errorf("failed validation: %s", err)
		return nil, twirp.InvalidArgumentError(arg, "invalid")
	}

	if err := s.insertUserPrompt(ctx, s.DB, req); err != nil {
		logrus.Error("failed to insert user prompt: %s", err)
		return nil, twirp.InternalError("failed to add user prompt")
	}
	return &gpb.Empty{}, nil
}

func (s *NotifyAppServer) validateAddUserPrompt(ctx context.Context, req *pb.UserPrompt) (string, error) {
	if !govalidator.IsUUID(req.PromptId) {
		return "prompt_id", fmt.Errorf("prompt_id '%s' is invalid", req.PromptId)
	}
	if len(req.PhoneNumber) != 10 || !govalidator.IsNumeric(req.PhoneNumber) {
		return "phone_number", fmt.Errorf("phone_number '%s' is invalid", req.PhoneNumber)
	}
	if _, err := time.Parse(timeFormat, req.NextPromptTime); err != nil {
		return "next_prompt_time", errors.Wrapf(err, "next_prompt_time '%s' is invalid", req.NextPromptTime)
	}

	//parse frequency
	if _, err := time.ParseDuration(req.Frequency); err != nil {
		return "frequency", errors.Wrapf(err, "frequency '%s' is invalid", req.Frequency)
	}
	return "", nil
}

func (s *NotifyAppServer) insertUserPrompt(ctx context.Context, db Database, up *pb.UserPrompt) error {
	stmt, err := db.Prepare(`
		INSERT INTO user_prompts (prompt_id, phone_number, next_prompt_time, frequency, created, updated)
		VALUES (?, ?, ?, ?, NOW(6), NOW(6))
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare")
	}
	if _, err = stmt.Exec(up.PromptId, up.PhoneNumber, up.NextPromptTime, up.Frequency); err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (s *NotifyAppServer) deleteUserPrompt(ctx context.Context, db Database, phoneNumber, promptID string) error {
	stmt, err := db.Prepare(`
		DELETE FROM user_prompts
		WHERE phone_number=?
		AND prompt_id=?
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare")
	}
	if _, err = stmt.Exec(phoneNumber, promptID); err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (s *NotifyAppServer) getUserPrompts(ctx context.Context, db Database, phoneNumber string) ([]*pb.UserPrompt, error) {
	stmt, err := db.Prepare(`
		SELECT up.prompt_id,up.phone_number,up.next_prompt_time,up.frequency,p.template,p.type,p.name
		FROM user_prompts up, prompts p
		WHERE up.phone_number = ?
		AND up.prompt_id=p.prompt_id`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare")
	}
	rows, err := stmt.Query(phoneNumber)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()
	userPrompts := []*pb.UserPrompt{}
	for rows.Next() {
		up := &pb.UserPrompt{Prompt: &pb.Prompt{}}
		if err := rows.Scan(&up.PromptId, &up.PhoneNumber, &up.NextPromptTime, &up.Frequency, &up.Prompt.Template, &up.Prompt.Type, &up.Prompt.Name); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		userPrompts = append(userPrompts, up)
	}
	return userPrompts, nil
}

func (s *NotifyAppServer) getAllUserPrompts(ctx context.Context, db Database) ([]*pb.UserPrompt, error) {
	stmt, err := db.Prepare(`
		SELECT up.prompt_id,up.phone_number,up.next_prompt_time,up.frequency,p.template,p.type,p.name
		FROM user_prompts up, prompts p
		WHERE up.next_prompt_time <= DATE_SUB(NOW(6), INTERVAL 15 SECOND)
		AND up.prompt_id=p.prompt_id`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare")
	}
	rows, err := stmt.Query()
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()
	userPrompts := []*pb.UserPrompt{}
	for rows.Next() {
		up := &pb.UserPrompt{Prompt: &pb.Prompt{}}
		if err := rows.Scan(&up.PromptId, &up.PhoneNumber, &up.NextPromptTime, &up.Frequency, &up.Prompt.Template, &up.Prompt.Type, &up.Prompt.Name); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		userPrompts = append(userPrompts, up)
	}
	return userPrompts, nil
}

func (s *NotifyAppServer) updateUserPrompt(ctx context.Context, db Database, prompt *pb.UserPrompt) error {
	stmt, err := db.Prepare(`
		UPDATE user_prompts 
		SET updated=NOW(6), next_prompt_time=? 
		WHERE prompt_id=?`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare")
	}
	currPromptTime, err := time.Parse(timeFormat, prompt.NextPromptTime)
	if err != nil {
		return errors.Wrapf(err, "failed to parse next prompt time")
	}
	frequency, err := time.ParseDuration(prompt.Frequency)
	if err != nil {
		return errors.Wrapf(err, "failed to parse frequency")
	}

	nextPromptTime := currPromptTime.Add(frequency)
	if nextPromptTime.Before(now(db)) {
		//this is the case that prevents notifier from running repeatedly on really old notifications
		nextPromptTime = now(s.DB).Add(frequency)
	}

	if _, err = stmt.Exec(nextPromptTime, prompt.PromptId); err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}
