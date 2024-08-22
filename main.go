package main

import (
	"github.com/sonnt85/goshellbox/client"
	"github.com/sonnt85/goshellbox/params"
	"github.com/sonnt85/goshellbox/server"
)

func main() {
	server.Init()
	parms := new(params.Parameter)
	parms.Init()
	if parms.IsServer {
		server.Run(parms)
	} else {
		client.Run(parms)
	}
}

func Init() {
	// cmd.Init()
}
