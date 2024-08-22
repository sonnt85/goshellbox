package server

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sonnt85/gosutils/slogrus"
	"github.com/sonnt85/gosutils/vncproxy"
	"github.com/sonnt85/gosystem"

	"github.com/sonnt85/goshellbox/lib"
	"github.com/sonnt85/goshellbox/lib/osauth"

	paseto "aidanwoods.dev/go-paseto"
	"github.com/gorilla/websocket"
)

var sessionKey = paseto.NewV4SymmetricKey()

// HTMLDirHandler FileServer
func HTMLDirHandler() http.Handler {
	return http.FileServer(http.Dir("html"))
}

// GetMethodHandler Only allow GET requests
func GetMethodHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// function checkpwd(){
//    sudo -k; echo "$1" | sudo -S echo sonnt 2>&1 | grep -ie "Sorry, try again" && return 1 || return 0
// }

// func PamCheck(user, pass string) bool { //	"github.com/msteinert/pam/v2"

// 	t, err := pam.StartFunc("", user, func(s pam.Style, msg string) (string, error) {
// 		switch s {
// 		case pam.PromptEchoOff:
// 			return pass, nil
// 		case pam.PromptEchoOn:
// 			return pass, nil
// 		case pam.ErrorMsg:
// 			// fmt.Println(msg)
// 			return "", nil
// 		case pam.TextInfo:
// 			// fmt.Println(msg)
// 			return "", nil
// 		default:
// 			return "", errors.New("unrecognized message style")
// 		}
// 	})

// 	if err != nil {
// 		// fmt.Println("StartFunc:", err)
// 		return false
// 	}

// 	err = t.Authenticate(0)
// 	if err != nil {
// 		// fmt.Println("Authenticate:", err)
// 		return false
// 	}

// 	err = t.AcctMgmt(0)
// 	return err != nil
// }

// VerifyHandler Login verification
func VerifyHandler(username, password string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if username == "" && password == "" {
			// authentication disabled, permit all traffic
		} else {
			token := strings.TrimPrefix(r.URL.Path, "/cmd/")
			if len(token) < 10 {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			p := paseto.NewParser()
			if _, err := p.ParseV4Local(sessionKey, token, nil); err != nil {
				log.Printf("Invalid token: %v", err)
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func LogoutHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		p := paseto.NewParser()
		if _, err := p.ParseV4Local(sessionKey, token, nil); err != nil {
			w.WriteHeader(http.StatusForbidden)
			return
		}
	})
}

// LoginHandler Login interface
func LoginHandler(username, password string) http.Handler {
	if username == "" || password == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Authentication disabled, return a success regardless
			w.Write([]byte("{\"code\":0,\"msg\":\"Logged-in automatically (no authentication required)\",\"path\":\"noauth--login-not-required\"}"))
		})
	}

	md5User := lib.HashCalculation(md5.New(), username)
	md5Pass := lib.HashCalculation(md5.New(), password)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/json; charset=utf-8")

		const halfSecond = int64(time.Second / 2)
		time.Sleep(time.Duration(rand.Int63n(halfSecond)))

		if token := r.URL.Query().Get("token"); token != "" {
			p := paseto.NewParser()
			if _, err := p.ParseV4Local(sessionKey, token, nil); err == nil {
				log.Print("resuming session from stored token")
				w.Write([]byte("{\"code\":0,\"msg\":\"login success!\",\"path\":\"" + token + "\"}"))
				return
			}
		}

		sentUser, sentPass := r.URL.Query().Get("username"), r.URL.Query().Get("password")
		// md5User1 := lib.HashCalculation(md5.New(), "super")
		log.Infof("user %s, pass %s", sentUser, sentPass)
		if (md5User != sentUser || md5Pass != sentPass) && !new(osauth.OSAuth).AuthUser(strings.TrimSuffix(sentUser, "."), sentPass) {
			w.Write([]byte("{\"code\":1,\"msg\":\"Login incorrect!\"}"))
			return
		}

		// login success
		token := paseto.NewToken()

		token.SetIssuedAt(time.Now())
		token.SetNotBefore(time.Now())
		token.SetExpiration(time.Now().Add(24 * time.Hour))
		tokenBytes := token.V4Encrypt(sessionKey, nil)

		w.Write([]byte("{\"code\":0,\"msg\":\"login success!\",\"path\":\"" + tokenBytes + "\"}"))
	})
}

// func pingHandler(conn *websocket.Conn) {
// 	pingPeriod := 30 * time.Second
// 	for {
// 		err := conn.WriteMessage(websocket.PingMessage, []byte{})
// 		if err != nil {
// 			return
// 		}
// 		time.Sleep(pingPeriod)
// 	}
// }

// ConnectionHandler Make websocket and childprocess communicate
func ConnectionHandler(command string) http.Handler {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrader.Upgrade error:", err.Error())
			return
		}
		defer conn.Close()
		log.Printf("Connection request: %s %s", r.Method, r.URL.Path)
		proxyHost := ""
		if portHeader := r.Header.Get("ProxyPort"); portHeader != "" {
			if _, err := strconv.Atoi(portHeader); err == nil {
				proxyHost = fmt.Sprintf("127.0.0.1:%s", portHeader)
			} else {
				proxyHost = portHeader
			}

			var peer *vncproxy.Peer

			peer, err = vncproxy.NewPeer(vncproxy.NewWebSocketConn(conn), proxyHost, "")
			if err != nil {
				log.Print("NewPeer error:", err.Error())
				return
			}
			defer peer.Close()
			// copy input from websocket to proxy
			peer.StartCopy()

			return
		}

		pl, err := NewPipeLine(conn, command)
		if err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
			return
		}
		defer func() {
			if pid, err := pl.pty.Pid(); err == nil {
				gosystem.SendSignalToAllProcess(gosystem.GetKillSignal(), pid)
			}
			pl.pty.Close()
		}()
		// go pingHandler(conn)

		logChan := make(chan string)
		// pre_login := ""
		// for _, v := range []string{"/usr/share/bash-completion/bash_completion", "/etc/bash_completion"} {
		// 	if gosystem.FileIsExist(v) {
		// 		pre_login += ". " + v + "\n"
		// 		break
		// 	}
		// }
		// if _, e := exec.LookPath("bash"); e == nil {
		// 	pre_login += "history -cw\n"
		// }
		// if _, e := exec.LookPath("clear"); e == nil {
		// 	pre_login += "clear\n"
		// }
		// if len(pre_login) != 0 {
		// 	pl.pty.Write([]byte(pre_login))
		// }
		go pl.ReadSktAndWritePty(logChan)
		go pl.ReadPtyAndWriteSkt(logChan)

		errlog := <-logChan
		if len(errlog) != 0 {
			log.Print(errlog)
		}
		go func() {
			<-logChan
			close(logChan)
		}()
	})
}

// ContentPathHandler content path prefix
func ContentPathHandler(contentpath string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, contentpath)
		r.URL.Path = p
		next.ServeHTTP(w, r)
	})
}

// LoggingHandler Log print
func LoggingHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Add("Server", Server)
		next.ServeHTTP(w, r)
		str := fmt.Sprintf(
			"%s Completed %s %s in %v from %s",
			start.Format("2006/01/02 15:04:05"),
			r.Method,
			r.URL.Path,
			time.Since(start),
			r.RemoteAddr)
		log.Info(str)
	})
}
