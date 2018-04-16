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

func (s *NotifyAppServer) getNotification(ctx context.Context, db Database, notificationID string) (*pb.Notification, error) {
	stmt, err := db.Prepare(`
		SELECT name,type,template
		FROM notifications 
		WHERE notification_id=?`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare")
	}
	rows, err := stmt.Query(notificationID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()
	notification := &pb.Notification{NotificationId: notificationID}
	if rows.Next() {
		if err := rows.Scan(&notification.Name, &notification.Type, &notification.Template); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
	} else {
		return nil, fmt.Errorf("notification id '%s' not found", notificationID)
	}
	return notification, nil
}

func (s *NotifyAppServer) getNotifications(ctx context.Context, db Database) ([]*pb.Notification, error) {
	stmt, err := db.Prepare(`
		SELECT notification_id,name,type,template
		FROM notifications`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare")
	}
	rows, err := stmt.Query()
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()
	notifications := []*pb.Notification{}
	for rows.Next() {
		p := &pb.Notification{}
		if err := rows.Scan(&p.NotificationId, &p.Name, &p.Type, &p.Template); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		notifications = append(notifications, p)
	}
	return notifications, nil
}

func (s *NotifyAppServer) insertNotification(ctx context.Context, db Database, p *pb.Notification) error {
	p.NotificationId = uuid.NewV4().String()
	stmt, err := db.Prepare(`
		INSERT INTO notifications (notification_id, name, type, template, created, updated)
		VALUES (?, ?, ?, ?, NOW(6), NOW(6))
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare")
	}
	if _, err = stmt.Exec(p.NotificationId, p.Name, p.Type, p.Template); err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (s *NotifyAppServer) AddUserNotification(ctx context.Context, req *pb.UserNotification) (*gpb.Empty, error) {
	if arg, err := s.validateAddUserNotification(ctx, req); err != nil {
		logrus.Errorf("failed validation: %s", err)
		return nil, twirp.InvalidArgumentError(arg, "invalid")
	}

	if err := s.insertUserNotification(ctx, s.DB, req); err != nil {
		logrus.Error("failed to insert user notification: %s", err)
		return nil, twirp.InternalError("failed to add user notification")
	}
	return &gpb.Empty{}, nil
}

func (s *NotifyAppServer) validateAddUserNotification(ctx context.Context, req *pb.UserNotification) (string, error) {
	if !govalidator.IsUUID(req.NotificationId) {
		return "notification_id", fmt.Errorf("notification_id '%s' is invalid", req.NotificationId)
	}
	if len(req.PhoneNumber) != 10 || !govalidator.IsNumeric(req.PhoneNumber) {
		return "phone_number", fmt.Errorf("phone_number '%s' is invalid", req.PhoneNumber)
	}
	if _, err := time.Parse(timeFormat, req.NextNotificationTime); err != nil {
		return "next_notification_time", errors.Wrapf(err, "next_notification_time '%s' is invalid", req.NextNotificationTime)
	}

	//parse frequency
	if _, err := parseDuration(req.Frequency); err != nil {
		return "frequency", errors.Wrapf(err, "frequency '%s' is invalid", req.Frequency)
	}
	return "", nil
}

func (s *NotifyAppServer) insertUserNotification(ctx context.Context, db Database, up *pb.UserNotification) error {
	stmt, err := db.Prepare(`
		INSERT INTO user_notifications (notification_id, phone_number, next_notification_time, frequency, created, updated)
		VALUES (?, ?, ?, ?, NOW(6), NOW(6))
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare")
	}
	if _, err = stmt.Exec(up.NotificationId, up.PhoneNumber, up.NextNotificationTime, up.Frequency); err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (s *NotifyAppServer) deleteUserNotification(ctx context.Context, db Database, phoneNumber, notificationID string) error {
	stmt, err := db.Prepare(`
		DELETE FROM user_notifications
		WHERE phone_number=?
		AND notification_id=?
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare")
	}
	if _, err = stmt.Exec(phoneNumber, notificationID); err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (s *NotifyAppServer) getUserNotifications(ctx context.Context, db Database, phoneNumber string) ([]*pb.UserNotification, error) {
	stmt, err := db.Prepare(`
		SELECT up.notification_id,up.phone_number,up.next_notification_time,up.frequency,p.template,p.type,p.name
		FROM user_notifications up, notifications p
		WHERE up.phone_number = ?
		AND up.notification_id=p.notification_id`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare")
	}
	rows, err := stmt.Query(phoneNumber)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()
	userNotifications := []*pb.UserNotification{}
	for rows.Next() {
		up := &pb.UserNotification{Notification: &pb.Notification{}}
		if err := rows.Scan(&up.NotificationId, &up.PhoneNumber, &up.NextNotificationTime, &up.Frequency, &up.Notification.Template, &up.Notification.Type, &up.Notification.Name); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		userNotifications = append(userNotifications, up)
	}
	return userNotifications, nil
}

func (s *NotifyAppServer) getAllUserNotifications(ctx context.Context, db Database) ([]*pb.UserNotification, error) {
	stmt, err := db.Prepare(`
		SELECT up.notification_id,up.phone_number,up.next_notification_time,up.frequency,p.template,p.type,p.name
		FROM user_notifications up, notifications p
		WHERE up.next_notification_time <= DATE_SUB(NOW(6), INTERVAL 15 SECOND)
		AND up.notification_id=p.notification_id`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare")
	}
	rows, err := stmt.Query()
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()
	userNotifications := []*pb.UserNotification{}
	for rows.Next() {
		up := &pb.UserNotification{Notification: &pb.Notification{}}
		if err := rows.Scan(&up.NotificationId, &up.PhoneNumber, &up.NextNotificationTime, &up.Frequency, &up.Notification.Template, &up.Notification.Type, &up.Notification.Name); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
		userNotifications = append(userNotifications, up)
	}
	return userNotifications, nil
}

func (s *NotifyAppServer) updateUserNotification(ctx context.Context, db Database, notification *pb.UserNotification) error {
	stmt, err := db.Prepare(`
		UPDATE user_notifications 
		SET updated=NOW(6), next_notification_time=? 
		WHERE notification_id=?`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare")
	}
	currNotificationTime, err := time.Parse(timeFormat, notification.NextNotificationTime)
	if err != nil {
		return errors.Wrapf(err, "failed to parse next notification time")
	}
	if notification.Frequency == "" {
		//notification is not recurring
		if err := s.deleteUserNotification(ctx, db, notification.PhoneNumber, notification.NotificationId); err != nil {
			return errors.Wrapf(err, "failed to delete one time notification: %+v", notification)
		}
		return nil
	}

	frequency, err := parseDuration(notification.Frequency)
	if err != nil {
		return errors.Wrapf(err, "failed to parse frequency")
	}

	nextNotificationTime := currNotificationTime.Add(frequency)
	if nextNotificationTime.Before(now(db)) {
		//this is the case that prevents notifier from running repeatedly on really old notifications
		nextNotificationTime = now(s.DB).Add(frequency)
	}

	if _, err = stmt.Exec(nextNotificationTime, notification.NotificationId); err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}
