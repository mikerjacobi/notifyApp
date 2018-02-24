package controllers

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	pb "github.com/mikerjacobi/notify-app/server/rpc"
	"github.com/pkg/errors"
)

func (s *NotifyAppServer) populateTemplateByID(ctx context.Context, db Database, promptID string, payload interface{}) (string, error) {
	prompt, err := s.getPrompt(ctx, db, promptID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get prompt")
	}
	return s.populateTemplate(ctx, db, prompt, payload)
}

func (s *NotifyAppServer) populateTemplate(ctx context.Context, db Database, prompt *pb.Prompt, payload interface{}) (string, error) {
	switch prompt.Type {
	case "registration":
		return s.populateRegistrationTemplate(ctx, db, prompt, payload)
	case "reminder":
		return prompt.Template, nil
	case "question":
		return prompt.Template, nil
	default:
		return "", fmt.Errorf("prompt type: '%s' is unhandled", prompt.Type)
	}
}

type regAckPayload struct {
	Name string
}

func (s *NotifyAppServer) populateRegistrationTemplate(ctx context.Context, db Database, prompt *pb.Prompt, payload interface{}) (string, error) {
	if prompt.Name == "register" {
		return prompt.Template, nil
	} else if prompt.Name != "register-ack" {
		return "", fmt.Errorf("reg prompt name: '%s' is unhandled")
	}

	buf := &bytes.Buffer{}
	if err := template.Must(template.New("regack").Parse(prompt.Template)).Execute(buf, payload); err != nil {
		return "", errors.Wrap(err, "failed to execute regack ")
	}
	return buf.String(), nil
}
