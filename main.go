// package main starts the server
package main

import (
	"github.com/mvaldes14/twitch-bot/pkgs/secrets"

	"github.com/mvaldes14/twitch-bot/pkgs/server"
)

func main() {
	const port = ":3000"
	s := secrets.NewSecretService()
	s.InitSecrets()

	srv := server.NewServer(port)
	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}

}
