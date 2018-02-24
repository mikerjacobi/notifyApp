package controllers

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/schema"
	pb "github.com/mikerjacobi/notify-app/server/rpc"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type TwilioConfig struct {
	API      string `json:"sms_api"`
	From     string `json:"from_number"`
	User     string `json:"username"`
	Password string `json:"password"`
}

type TwilioInboundReq struct {
	From string `schema:"From"`
	Body string `schema:"Body"`
}

func (s *NotifyAppServer) sendSMS(ctx context.Context, to string, body string) error {

	if to[0:3] == "000" {
		logrus.Infof("TEST: sent '%s' to %s", body, to)
		return nil
	}

	v := url.Values{}
	v.Add("To", to)
	v.Add("From", s.config.From)
	v.Add("Body", body)
	req, err := http.NewRequest(http.MethodPost, s.config.API+"/Messages.json", bytes.NewBufferString(v.Encode()))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.config.User, s.config.Password)
	resp, err := s.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to execute request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		if respBody, err := ioutil.ReadAll(resp.Body); err == nil {
			return fmt.Errorf("received http %d from twilio: %s", resp.StatusCode, string(respBody))
		}
		return fmt.Errorf("received http %d from twilio", resp.StatusCode)
	}
	logrus.Infof("sent '%s' to %s", body, to)
	return nil
}

func (s *NotifyAppServer) TwilioInboundHandler(w http.ResponseWriter, r *http.Request) {
	lf := logrus.Fields{}
	ctx := context.Background()
	if err := r.ParseForm(); err != nil {
		logrus.Errorf("failed to parse form: %s", err)
		w.WriteHeader(400)
		return
	}

	payload := TwilioInboundReq{}
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	if err := decoder.Decode(&payload, r.PostForm); err != nil {
		logrus.Errorf("failed to decode body: %s", err)
		w.WriteHeader(400)
		return
	}
	payload.From = strings.Replace(payload.From, "+1", "", -1)
	lf["from"] = payload.From

	recv := &pb.Communication{To: s.config.From, From: payload.From, Message: payload.Body}
	if err := s.insertCommunication(ctx, s.DB, recv); err != nil {
		logrus.WithFields(lf).Warn("failed to insert comms: %s", err)
	}

	user, err := s.getUser(ctx, s.DB, payload.From)
	if err != nil {
		logrus.WithFields(lf).Errorf("failed to get user: %+v", err)
		w.WriteHeader(404)
		return
	}

	//handle the incoming message
	if payload.Body == "reg" {
		if err := s.verifyUser(ctx, user); err != nil {
			logrus.WithFields(lf).Errorf("failed to register user: %s", err)
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(200)
		return
	}

	sent, err := s.getMostRecentSentCommunication(ctx, s.DB, payload.From)
	if err != nil {
		logrus.WithFields(lf).Errorf("failed to get last prompt: %s", err)
		w.WriteHeader(500)
		return
	}
	journal := &pb.Journal{
		CommsId:     recv.CommsId,
		PhoneNumber: payload.From,
		Prompt:      sent.Message,
		Entry:       recv.Message,
	}
	if err := s.insertJournal(ctx, s.DB, journal); err != nil {
		logrus.WithFields(lf).Errorf("failed to insert journal: %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
}
