package controllers

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	pb "github.com/mikerjacobi/notify-app/server/rpc"
	"github.com/pkg/errors"
)

func (s *NotifyAppServer) populateTemplateByID(ctx context.Context, db Database, notificationID string, payload interface{}) (string, error) {
	notification, err := s.getNotification(ctx, db, notificationID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get notification")
	}
	return s.populateTemplate(ctx, db, notification, payload)
}

func (s *NotifyAppServer) populateTemplate(ctx context.Context, db Database, notification *pb.Notification, payload interface{}) (string, error) {
	switch notification.Type {
	case "registration":
		return s.populateRegistrationTemplate(ctx, db, notification, payload)
	case "reminder":
		return notification.Template, nil
	case "prompt":
		return notification.Template, nil
	default:
		return "", fmt.Errorf("notification type: '%s' is unhandled", notification.Type)
	}
}

type regAckPayload struct {
	Name string
}

func (s *NotifyAppServer) populateRegistrationTemplate(ctx context.Context, db Database, notification *pb.Notification, payload interface{}) (string, error) {
	if notification.Name == "register" {
		return notification.Template, nil
	} else if notification.Name != "register-ack" {
		return "", fmt.Errorf("reg notification name: '%s' is unhandled")
	}

	buf := &bytes.Buffer{}
	if err := template.Must(template.New("regack").Parse(notification.Template)).Execute(buf, payload); err != nil {
		return "", errors.Wrap(err, "failed to execute regack ")
	}
	return buf.String(), nil
}
