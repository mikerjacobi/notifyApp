package controllers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/husobee/vestigo"
	pb "github.com/mikerjacobi/notify-app/server/rpc"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type BaseTemplate struct {
	Name    string
	Tab     string
	Payload interface{}
}

func (s *NotifyAppServer) GetLogin(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "login", nil)
}

func (s *NotifyAppServer) Logout(w http.ResponseWriter, r *http.Request) {
	c := &http.Cookie{Name: "session"}
	http.SetCookie(w, c)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (s *NotifyAppServer) PostLogin(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if err := r.ParseForm(); err != nil {
		logrus.Errorf("failed to parse form: %s", err)
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	phoneNumber := r.PostForm.Get("phone_number")
	user, err := s.getUser(ctx, s.DB, phoneNumber)
	if err != nil {
		logrus.Errorf("failed to get user: %s", err)
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	inputPassword := []byte(r.PostForm.Get("password"))
	storedPassword, err := base64.StdEncoding.DecodeString(user.Password)
	if err != nil {
		logrus.Errorf("failed to decode hashword: %s", err)
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if err := bcrypt.CompareHashAndPassword(storedPassword, inputPassword); err != nil {
		logrus.Errorf("password mismatch: %s", err)
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	user.SessionId = uuid.NewV4().String()
	session := authSession{
		PhoneNumber: phoneNumber,
		ID:          user.SessionId,
	}

	payload, err := json.Marshal(session)
	if err != nil {
		logrus.Errorf("failed to marshal session: %s", err)
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if err := s.updateUser(ctx, s.DB, user); err != nil {
		logrus.Errorf("failed to upate user: %s", err)
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	c := &http.Cookie{
		Name:  "session",
		Value: base64.StdEncoding.EncodeToString(payload),
	}
	http.SetCookie(w, c)
	http.Redirect(w, r, "/journal", http.StatusFound)
}

func (s *NotifyAppServer) GetJournal(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(userKey).(*pb.User)
	if !ok {
		logrus.Errorf("failed to get user")
		http.Redirect(w, r, "/logout", http.StatusFound)
		return
	}
	entries, err := s.getJournalEntries(r.Context(), s.DB, user.PhoneNumber)
	if err != nil {
		logrus.Errorf("failed to get entries: %s", err)
		renderTemplate(w, r, "error", nil)
		return
	}
	payload := struct {
		Entries []*pb.Journal
	}{entries}
	renderTemplate(w, r, "journal", &payload)
}

func (s *NotifyAppServer) GetConfigure(w http.ResponseWriter, r *http.Request) {

	user, ok := r.Context().Value(userKey).(*pb.User)
	if !ok {
		logrus.Errorf("failed to get user")
		http.Redirect(w, r, "/logout", http.StatusFound)
		return
	}

	notifications, err := s.getNotifications(r.Context(), s.DB)
	if err != nil {
		logrus.Errorf("failed to get notifications: %s", err)
		renderTemplate(w, r, "error", nil)
		return
	}

	userNotifications, err := s.getUserNotifications(r.Context(), s.DB, user.PhoneNumber)
	if err != nil {
		logrus.Errorf("failed to get notifications: %s", err)
		renderTemplate(w, r, "error", nil)
		return
	}

	payload := struct {
		Notifications     []*pb.Notification
		UserNotifications []*pb.UserNotification
	}{notifications, userNotifications}
	renderTemplate(w, r, "configure", payload)
}

func renderTemplate(w http.ResponseWriter, r *http.Request, page string, payload interface{}) {
	tmplDir := "../client/"
	paths := []string{
		filepath.Join(tmplDir, "base.html"),
		filepath.Join(tmplDir, page+".html"),
	}
	tmpl := template.Must(template.ParseFiles(paths...))

	user, ok := r.Context().Value(userKey).(*pb.User)
	if !ok || user == nil {
		user = &pb.User{}
	}

	if page == "error" {
		w.WriteHeader(500)
	}

	base := BaseTemplate{
		Name:    user.Name,
		Tab:     page,
		Payload: payload,
	}
	if err := tmpl.Execute(w, base); err != nil {
		logrus.Errorf("failed to execute tmpl: %s", err)
		w.WriteHeader(500)
		return
	}
}

func (s *NotifyAppServer) PostUserNotification(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		logrus.Errorf("failed to parse form: %s", err)
		renderTemplate(w, r, "error", nil)
		return
	}

	user, ok := r.Context().Value(userKey).(*pb.User)
	if !ok {
		logrus.Errorf("failed to get user")
		http.Redirect(w, r, "/logout", http.StatusFound)
		return
	}

	up := &pb.UserNotification{
		PhoneNumber:          user.PhoneNumber,
		Frequency:            r.PostForm.Get("frequency"),
		NextNotificationTime: r.PostForm.Get("notification_time"),
	}

	switch r.PostForm.Get("radios") {
	case "prompt":
		newPrompt := r.PostForm.Get("new_prompt")
		if newPrompt != "" {
			notification := &pb.Notification{Type: "prompt", Template: newPrompt}
			err = s.insertNotification(r.Context(), s.DB, notification)
			up.NotificationId = notification.NotificationId
		} else {
			up.NotificationId = r.PostForm.Get("select_notification")
			_, err = s.getNotification(r.Context(), s.DB, up.NotificationId)
		}
	case "reminder":
		notification := &pb.Notification{Type: "reminder", Template: r.PostForm.Get("new_reminder")}
		err = s.insertNotification(r.Context(), s.DB, notification)
		up.NotificationId = notification.NotificationId
	default:
		logrus.Errorf("invalid notification type: %s", r.PostForm.Get("radios"))
		renderTemplate(w, r, "error", nil)
		return
	}
	if err != nil {
		logrus.Errorf("failed to insert/get notification: %s", err)
		renderTemplate(w, r, "error", nil)
		return
	}

	if _, err := s.AddUserNotification(r.Context(), up); err != nil {
		logrus.Errorf("failed to insert user notification: %s", err)
		renderTemplate(w, r, "error", nil)
		return
	}

	http.Redirect(w, r, "/configure", http.StatusFound)
}

func (s *NotifyAppServer) DeleteUserNotification(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		logrus.Errorf("failed to parse form: %s", err)
		renderTemplate(w, r, "error", nil)
		return
	}

	user, ok := r.Context().Value(userKey).(*pb.User)
	if !ok {
		logrus.Errorf("failed to get user")
		http.Redirect(w, r, "/logout", http.StatusFound)
		return
	}

	notificationID := vestigo.Param(r, "notification_id")
	if err := s.deleteUserNotification(r.Context(), s.DB, user.PhoneNumber, notificationID); err != nil {
		logrus.Errorf("failed to deleteuser notification: %s", err)
		renderTemplate(w, r, "error", nil)
		return
	}

	http.Redirect(w, r, "/configure", http.StatusFound)
}

func (s *NotifyAppServer) PostJournal(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		logrus.Errorf("failed to parse form: %s", err)
		renderTemplate(w, r, "error", nil)
		return
	}

	user, ok := r.Context().Value(userKey).(*pb.User)
	if !ok {
		logrus.Errorf("failed to get user")
		http.Redirect(w, r, "/logout", http.StatusFound)
		return
	}

	journal := &pb.Journal{
		PhoneNumber: user.PhoneNumber,
		Title:       r.PostForm.Get("journal_title"),
		Entry:       r.PostForm.Get("journal_entry"),
	}

	if err := s.insertJournal(r.Context(), s.DB, journal); err != nil {
		logrus.Errorf("failed to insert user notification: %s", err)
		renderTemplate(w, r, "error", nil)
		return
	}

	http.Redirect(w, r, "/journal", http.StatusFound)
}

func (s *NotifyAppServer) PutJournal(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(userKey).(*pb.User)
	if !ok {
		logrus.Errorf("failed to get user")
		w.WriteHeader(500)
		w.Write([]byte("{}"))
		return
	}

	journal := &pb.Journal{
		PhoneNumber: user.PhoneNumber,
		JournalId:   vestigo.Param(r, "journal_id"),
	}
	if err := json.NewDecoder(r.Body).Decode(journal); err != nil {
		logrus.Errorf("failed to decode put journal: %s", err)
		w.WriteHeader(500)
		w.Write([]byte("{}"))
		return
	}

	if err := s.updateJournal(r.Context(), s.DB, journal); err != nil {
		logrus.Errorf("failed to update journal: %s", err)
		w.WriteHeader(500)
		w.Write([]byte("{}"))
		return
	}

	w.Write([]byte("{}"))
}

func (s *NotifyAppServer) DeleteJournal(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(userKey).(*pb.User)
	if !ok {
		logrus.Errorf("failed to get user")
		w.WriteHeader(500)
		w.Write([]byte("{}"))
		return
	}

	journal := &pb.Journal{
		PhoneNumber: user.PhoneNumber,
		JournalId:   vestigo.Param(r, "journal_id"),
	}
	if err := s.deleteJournal(r.Context(), s.DB, journal); err != nil {
		logrus.Errorf("failed to update journal: %s", err)
		w.WriteHeader(500)
		w.Write([]byte("{}"))
		return
	}

	w.Write([]byte("{}"))
}
