package client

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/sonnt85/goshellbox/lib"
	"github.com/sonnt85/goshellbox/params"
	log "github.com/sonnt85/gosutils/slogrus"
	"github.com/sonnt85/gosutils/vncproxy"

	"github.com/gorilla/websocket"
)

// LoginServer get websocket path
func LoginServer(https bool, username, password, host, port, contentpath string, get func(url string) (map[string]interface{}, error)) (string, error) {
	protocol := "http"
	if https {
		protocol = "https"
	}
	md5User := lib.HashCalculation(md5.New(), username)
	md5Pass := lib.HashCalculation(md5.New(), password)

	var LoginURL = protocol + "://" + host + ":" + port + contentpath + "/login"
	data, err := get(LoginURL + "?username=" + md5User + "&password=" + md5Pass)
	if err != nil {
		return "", err
	}
	if data["code"] != 0.0 {
		return "", errors.New(data["msg"].(string))
	}
	return data["path"].(string), nil
}

// ConnectSocket c
func ConnectSocket(para *params.Parameter, UserAgent string, path string, conn func(url string, headers ...map[string]string) (*websocket.Conn, error)) {

	// func ConnectSocket(https bool, host, port, contentpath, path, UserAgent string, conn func(url string) (*websocket.Conn, error)) {
	protocol := "ws"
	if para.HTTPS {
		protocol = "wss"
	}
	var headers = make(map[string]string)
	if len(para.ProxyPort) > 0 {
		headers["ProxyPort"] = para.ProxyPort
	}

	skt, err := conn(protocol+"://"+para.Host+":"+para.Port+para.ContentPath+"/cmd/"+path, headers)
	if err != nil {
		log.Error("Connect to WebSocket failed:", err.Error())
		return
	}
	if len(para.ProxyPort) <= 0 {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				err := skt.WriteJSON(lib.Message{
					Type: lib.KeepConnect,
				})
				if err != nil {
					log.Error("Error write keepconnect to WebSocket:", err.Error())
				}
			}
		}()
	}
	pl, _ := NewPipeLine(skt)
	defer skt.Close()
	logChan := make(chan string)
	// var wg sync.WaitGroup
	// wg.Add(1)
	go func() {
		var ct func(*[]byte)

		if len(para.ProxyPort) <= 0 {
			ct = nil
		} else {
			ct = func(*[]byte) {

			}
		}
		pl.ReadSktAndWriteStdio(logChan, ct)
		// wg.Done()
	}()
	var s tcell.Screen

	if len(para.ProxyPort) > 0 {
		wconn := vncproxy.NewWebSocketConn(skt)
		go io.Copy(wconn, os.Stdin)
		io.Copy(os.Stdout, wconn)
		// return
	} else {
		s, err = tcell.NewScreen()
		if err != nil {
			logChan <- fmt.Sprintf("Error ReadStdioAndWriteSkt NewScreen failed: %s", err)
			return
		}
		if err := s.Init(); err != nil {
			logChan <- fmt.Sprintf("Error ReadStdioAndWriteSkt Init failed: %s", err)
			return
		}
		pl.ReadStdioAndWriteSktTcell(s, logChan)
	}

	errlog := <-logChan
	if len(errlog) != 0 {
		log.Error(errlog)
	}
	go func() {
		<-logChan
		close(logChan)
	}()
	if s != nil {
		s.Clear()
		s.Fini()
	}
	// wg.Wait()
	// time.Sleep(time.Second * 10)
	// fmt.Print("EXIT")
}
