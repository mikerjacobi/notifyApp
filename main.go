package main

import (
	"net/http"
	"time"

	"github.com/husobee/vestigo"
	"github.com/mikerjacobi/notify-app/server/controllers"
	pb "github.com/mikerjacobi/notify-app/server/rpc"
	"github.com/sirupsen/logrus"
)

func logMiddleware(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		f(w, r)
		logrus.Infof("%s: %dns", r.URL.String(), time.Now().Sub(start))
	}
}

func main() {
	config := controllers.Configuration{
		TwilioSecretsPath: "/etc/secrets/twilio.json",
		DBSecretsPath:     "/etc/secrets/notify-db.json",
	}
	c, err := controllers.NewNotifyAppServer(config)
	if err != nil {
		logrus.Panicf("failed to initialize notify service: %+v", err)
	}
	go c.NotifyLoop()
	handler := pb.NewNotifyAppServer(c, nil)
	router := vestigo.NewRouter()

	//frontend routes
	router.Get("/login", c.GetLogin, logMiddleware)
	router.Post("/login", c.PostLogin, logMiddleware)
	router.Post("/twilio", c.TwilioInboundHandler, logMiddleware)
	router.Get("/journal", c.GetJournal, logMiddleware, c.AuthMiddleware)
	router.Post("/journal", c.PostJournal, logMiddleware, c.AuthMiddleware)
	router.Put("/journal/:journal_id", c.PutJournal, logMiddleware, c.AuthMiddleware)
	router.Delete("/journal/:journal_id", c.DeleteJournal, logMiddleware, c.AuthMiddleware)
	router.Get("/configure", c.GetConfigure, logMiddleware, c.AuthMiddleware)
	router.Post("/user-prompt", c.PostUserPrompt, logMiddleware, c.AuthMiddleware)
	router.Post("/user-prompt/:prompt_id/delete", c.DeleteUserPrompt, logMiddleware, c.AuthMiddleware)
	router.Get("/logout", c.Logout, logMiddleware, c.AuthMiddleware)

	//twirp setup
	router.HandleFunc(pb.NotifyAppPathPrefix+"*", handler.ServeHTTP, logMiddleware)

	logrus.Infof("starting server...")
	logrus.Fatal(http.ListenAndServe("0.0.0.0:8080", router))
}
