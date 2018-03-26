package controllers

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	_ "github.com/go-sql-driver/mysql"
	gpb "github.com/golang/protobuf/ptypes/empty"
	pb "github.com/mikerjacobi/notify-app/server/rpc"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/twitchtv/twirp"
	"golang.org/x/crypto/bcrypt"
)

type Configuration struct {
	DBSecretsPath     string
	TwilioSecretsPath string
	TwilioConfig
}

type NotifyAppServer struct {
	config Configuration
	client *http.Client
	*sql.DB
}

var (
	birthdayFormat = "2006-01-02"
	timeFormat     = "2006-01-02 15:04:05"
)

func NewNotifyAppServer(config Configuration) (*NotifyAppServer, error) {
	twilio, err := ioutil.ReadFile(config.TwilioSecretsPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read twilio")
	}
	if err := json.Unmarshal(twilio, &config); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal twilio")
	}
	//connect to database
	dbData := struct {
		Username string `json:"user"`
		Password string `json:"password"`
		Host     string `json:"host"`
	}{}
	dbFile, err := ioutil.ReadFile(config.DBSecretsPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read db file")
	}
	if err := json.Unmarshal(dbFile, &dbData); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal db data")
	}
	connStr := fmt.Sprintf("%s:%s@tcp(%s)/notify", dbData.Username, dbData.Password, dbData.Host)
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open dbconn to %s", dbData.Host)
	}
	db.SetConnMaxLifetime(time.Second * 10)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	c := &NotifyAppServer{
		config: config,
		client: http.DefaultClient,
		DB:     db,
	}
	return c, nil
}

func (s *NotifyAppServer) CreateAccount(ctx context.Context, req *pb.CreateAccountReq) (*pb.CreateAccountResp, error) {
	if arg, err := s.validateCreateAccount(ctx, req); err != nil {
		logrus.Errorf("failed validation: %s", err)
		return nil, twirp.InvalidArgumentError(arg, "invalid")
	}

	if err := s.insertUser(ctx, s.DB, req.User); err != nil {
		logrus.Error("failed to insert account: %s", err)
		return nil, twirp.InternalError("failed to create account")
	}

	registerNotificationID := "deaabd59-0d15-4f44-a3a8-1e3f920a3710"
	msg, err := s.populateTemplateByID(ctx, s.DB, registerNotificationID, nil)
	if err != nil {
		logrus.Error("failed to populate template: %s", err)
		return nil, twirp.InternalError("failed to create account")
	}

	if err := s.sendSMS(ctx, req.User.PhoneNumber, msg); err != nil {
		logrus.Error("failed to send sms: %s", err)
		return nil, twirp.InternalError("failed to create account")
	}
	comm := &pb.Communication{From: s.config.From, To: req.User.PhoneNumber, Message: msg}
	if err := s.insertCommunication(ctx, s.DB, comm); err != nil {
		logrus.Warn("failed to insert comms: %s", err)
	}
	return &pb.CreateAccountResp{Success: true}, nil
}

func (s *NotifyAppServer) validateCreateAccount(ctx context.Context, req *pb.CreateAccountReq) (string, error) {
	if req.User == nil {
		return "user", fmt.Errorf("user object missing")
	}
	if len(req.User.PhoneNumber) != 10 || !govalidator.IsNumeric(req.User.PhoneNumber) {
		return "phone_number", fmt.Errorf("phone_number '%s' is invalid", req.User.PhoneNumber)
	}
	if req.User.Password != req.PasswordRepeat {
		return "password", fmt.Errorf("passwords don't match")
	}
	if len(req.User.Password) < 6 {
		return "password", fmt.Errorf("password too short")
	}
	if req.User.Name == "" {
		return "name", fmt.Errorf("name '%s' is invalid", req.User.Name)
	}
	if _, err := time.Parse(birthdayFormat, req.User.Birthday); err != nil {
		return "birthday", errors.Wrapf(err, "birthday '%s' is invalid", req.User.Birthday)
	}
	return "", nil
}

func (s *NotifyAppServer) insertUser(ctx context.Context, db Database, user *pb.User) error {
	stmt, err := db.Prepare(`
		INSERT INTO users (phone_number, name, hashword, birthday, verified, created, updated)
		VALUES (?, ?, ?, ?, 0, NOW(6), NOW(6))
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare")
	}
	hashwordBytes, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.Wrap(err, "failed to generate hashword")
	}
	hashword := base64.StdEncoding.EncodeToString(hashwordBytes)
	if _, err = stmt.Exec(user.PhoneNumber, user.Name, hashword, user.Birthday); err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (s *NotifyAppServer) getUser(ctx context.Context, db Database, phoneNumber string) (*pb.User, error) {
	stmt, err := db.Prepare(`SELECT hashword,name,birthday,verified,session_id FROM users WHERE phone_number=?`)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare")
	}
	rows, err := stmt.Query(phoneNumber)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query")
	}
	defer rows.Close()
	user := &pb.User{PhoneNumber: phoneNumber}
	if rows.Next() {
		if err := rows.Scan(&user.Password, &user.Name, &user.Birthday, &user.Verified, &user.SessionId); err != nil {
			return nil, errors.Wrap(err, "failed to scan")
		}
	} else {
		return nil, fmt.Errorf("number '%s' not found", phoneNumber)
	}
	return user, nil
}

func (s *NotifyAppServer) updateUser(ctx context.Context, db Database, user *pb.User) error {
	stmt, err := db.Prepare(`
		UPDATE users SET verified=?,session_id=?,updated=NOW(6)
		WHERE phone_number=?
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare")
	}
	if _, err = stmt.Exec(user.Verified, user.SessionId, user.PhoneNumber); err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (s *NotifyAppServer) verifyUser(ctx context.Context, user *pb.User) error {
	if user.Verified {
		logrus.Infof("user %s already registered", user.PhoneNumber)
		return nil
	}

	user.Verified = true
	if err := s.updateUser(ctx, s.DB, user); err != nil {
		return errors.Wrap(err, "failed to verify user")
	}

	payload := &regAckPayload{user.Name}
	regAckNotificationID := "81a36dd3-8301-410c-af35-0b2a87cdd921"
	msg, err := s.populateTemplateByID(ctx, s.DB, regAckNotificationID, payload)
	if err != nil {
		return errors.Wrap(err, "failed to populate regack tmpl")
	}

	if err := s.sendSMS(ctx, user.PhoneNumber, msg); err != nil {
		return errors.Wrap(err, "failed to send sms")
	}
	comm := &pb.Communication{From: s.config.From, To: user.PhoneNumber, Message: msg, NotificationId: regAckNotificationID}
	if err := s.insertCommunication(ctx, s.DB, comm); err != nil {
		logrus.Warn("failed to insert comms: %s", err)
	}
	return nil
}

func (s *NotifyAppServer) NotifyLoop() {
	ctx := context.Background()
	for {
		time.Sleep(15 * time.Second)

		if err := s.triggerNotifications(ctx); err != nil {
			logrus.Errorf("failed to auto trigger notifications: %s", err)
		}
	}
}

func (s *NotifyAppServer) TriggerNotifications(ctx context.Context, empty *gpb.Empty) (*gpb.Empty, error) {
	if err := s.triggerNotifications(ctx); err != nil {
		logrus.Errorf("failed to http trigger notifications: %s", err)
		return nil, twirp.InternalError("failed to trigger notifications")
	}
	return &gpb.Empty{}, nil
}

func (s *NotifyAppServer) triggerNotifications(ctx context.Context) error {
	notifications, err := s.getAllUserNotifications(ctx, s.DB)
	if err != nil {
		return errors.Wrapf(err, "failed to get user notifications")
	}

	for _, notification := range notifications {
		if err := s.handleUserNotification(ctx, notification); err != nil {
			logrus.Warnf("failed to handle user notification: %+v.  %s", notification, err)
		}
	}
	return nil
}

func (s *NotifyAppServer) handleUserNotification(ctx context.Context, up *pb.UserNotification) error {
	txn, err := s.DB.Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin txn")
	}
	defer txn.Rollback()

	//update user notifications
	if err := s.updateUserNotification(context.Background(), txn, up); err != nil {
		return errors.Wrap(err, "failed to update user notification")
	}

	msg, err := s.populateTemplate(ctx, s.DB, up.Notification, nil)
	if err != nil {
		return errors.Wrap(err, "failed to populate regack tmpl")
	}

	//send sms
	if err := s.sendSMS(ctx, up.PhoneNumber, msg); err != nil {
		return errors.Wrap(err, "failed to send sms")
	}

	comm := &pb.Communication{From: s.config.From, To: up.PhoneNumber, Message: msg, NotificationId: up.NotificationId}
	if err := s.insertCommunication(ctx, txn, comm); err != nil {
		logrus.Warn("failed to insert comms: %s", err)
	}
	return errors.Wrap(txn.Commit(), "failed to commit")
}

func (s *NotifyAppServer) insertCommunication(ctx context.Context, db Database, comm *pb.Communication) error {
	comm.CommsId = uuid.NewV4().String()

	stmt, err := db.Prepare(`
		INSERT INTO communications (comms_id, notification_id, from_phone, to_phone, message, created)
		VALUES (?, ?, ?, ?, ?, NOW(6))
	`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare")
	}
	from := strings.Replace(comm.From, "+1", "", -1)
	to := strings.Replace(comm.To, "+1", "", -1)
	if _, err = stmt.Exec(comm.CommsId, comm.NotificationId, from, to, comm.Message); err != nil {
		return errors.Wrap(err, "failed to exec")
	}
	return nil
}

func (s *NotifyAppServer) getMostRecentPrompt(ctx context.Context, db Database, phoneNumber string) (string, error) {
	stmt, err := db.Prepare(`
		SELECT n.template
		FROM communications c, notifications n 
		WHERE c.to_phone=? 
		AND c.notification_id = n.notification_id
		AND n.type="prompt"
		ORDER BY c.created DESC LIMIT 1`)
	if err != nil {
		return "", errors.Wrap(err, "failed to prepare")
	}
	rows, err := stmt.Query(phoneNumber)
	if err != nil {
		return "", errors.Wrap(err, "failed to query")
	}
	defer rows.Close()
	if !rows.Next() {
		return "", fmt.Errorf("sent message not found to %s", phoneNumber)
	}

	notification := &pb.Notification{}
	if err := rows.Scan(&notification.Template); err != nil {
		return "", errors.Wrap(err, "failed to scan")
	}
	return notification.Template, nil
}
