package controllers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

type authSession struct {
	PhoneNumber string `json:"phone_number"`
	ID          string `json:"session_id"`
}

func (s *NotifyAppServer) AuthMiddleware(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session")
		if err != nil {
			logrus.Errorf("failed to get session cookie: %s", err)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		} else if sessionCookie == nil {
			//not logged in
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		sessionValue, err := base64.StdEncoding.DecodeString(sessionCookie.Value)
		if err != nil {
			logrus.Errorf("failed to decode session: %s", err)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		session := authSession{}
		if err := json.Unmarshal(sessionValue, &session); err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		ctx := context.Background()
		user, err := s.getUser(ctx, s.DB, session.PhoneNumber)
		if err != nil {
			logrus.Errorf("failed to get user: %s", err)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		if user.SessionId != session.ID {
			logrus.Errorf("session mismatch: '%s' != '%s'", user.SessionId, session.ID)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		ctx = context.WithValue(ctx, userKey, user)
		f(w, r.WithContext(ctx))
	}
}
