package apiserver

import (
	"net/http"
	"os"

	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
)

func Start(config *Config) error {
	storage, err := Init(config.Url)
	if err != nil {
		return err
	}

	defer storage.client.Disconnect(storage.ctx)
	//defer storage.cancel()

	CSRF := csrf.Protect([]byte(config.Secretkey), csrf.Secure(false))

	server := NewServer(*storage.client, sessions.NewCookieStore([]byte(config.Secretkey)), storage.ctx)

	return http.ListenAndServe(resolveAddress(), CSRF(server.Router))
}
func resolveAddress() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return ":8080"
}
