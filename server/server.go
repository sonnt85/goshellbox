package server

import (
	"crypto/tls"
	"log"
	"net/http"

	"github.com/sonnt85/goshellbox/lib"
	"github.com/sonnt85/goshellbox/params"
)

// Version WebShell Server current version
const Version = "2.0"

// Server Response header[Server]
const Server = "goshellbox-" + Version

// ShellBoxServer Main Server
type ShellBoxServer struct {
	http.ServeMux
}

// StaticHandler reserved for static_gen.go
var StaticHandler http.Handler

// Init WebShell. register handlers
func (s *ShellBoxServer) Init(Username, Password, Command, ContentPath string) {
	if StaticHandler == nil {
		StaticHandler = HTMLDirHandler()
	}
	s.Handle(ContentPath+"/", s.upgrade(ContentPath, StaticHandler))
	s.Handle(ContentPath+"/cmd/", s.upgrade(ContentPath, VerifyHandler(Username, Password, ConnectionHandler(Command))))
	s.Handle(ContentPath+"/login", s.upgrade(ContentPath, LoginHandler(Username, Password)))
	s.Handle(ContentPath+"/logout", s.upgrade(ContentPath, LogoutHandler()))
}

// packaging and upgrading http.Handler
func (s *ShellBoxServer) upgrade(ContentPath string, h http.Handler) http.Handler {
	return LoggingHandler(GetMethodHandler(ContentPathHandler(ContentPath, h)))
}

// Run WebShell server
func (s *ShellBoxServer) Run(params *params.Parameter) {
	var err error
	server := &http.Server{Addr: ":" + params.Port, Handler: s}
	if params.IsHttps {
		if params.RootCrtFile != "" {
			server.TLSConfig = &tls.Config{
				ClientCAs:  lib.ReadCertPool(params.RootCrtFile),
				ClientAuth: tls.RequireAndVerifyClientCert,
			}
		}
		err = server.ListenAndServeTLS(params.CrtFile, params.KeyFile)
	} else {
		err = server.ListenAndServe()
	}
	if err != nil {
		log.Fatal(err.Error())
	}
}

func Run(params *params.Parameter) {
	s := new(ShellBoxServer)
	s.Init(params.Username, params.Password, params.Command, params.ContentPath)
	s.Run(params)
}
