// package main starts the server
package main

import (
	"net/http"

	"github.com/mvaldes14/twitch-bot/pkgs/secrets"
	"github.com/mvaldes14/twitch-bot/pkgs/service"

	"github.com/mvaldes14/twitch-bot/pkgs/server"
)

const port = ":3000"

type app struct {
	Server  *http.Server
	Service *service.Service
	Secrets *secrets.SecretService
}

func newApp(port string) *app {
	service := service.NewService("main")
	server := server.NewServer(port)
	secrets := secrets.NewSecretService()
	return &app{
		Server:  server,
		Service: service,
		Secrets: secrets,
	}
}

func (a *app) initApp() error {
	err := a.Secrets.InitSecrets()
	return err
}

func main() {
	app := newApp(port)
	if err := app.initApp(); err != nil {
		app.Service.Logger.Error(err)
	}
	if err := app.Server.ListenAndServe(); err != nil {
		panic(err)
	}
}
