package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/runletapp/go-console"
	"github.com/sonnt85/goshellbox/lib"
	"github.com/sonnt85/gosystem"
)

func IsTimeoutError(err error) bool {
	// if err == nil {
	// 	return false
	// }
	return websocket.IsCloseError(err, websocket.CloseAbnormalClosure)
}

// PipeLine Connect websocket and childprocess
type PipeLine struct {
	pty console.Console
	skt *websocket.Conn
	sync.Mutex
}

type PipeLineTcp struct {
	con net.Conn
	skt *websocket.Conn
	sync.Mutex
}

// NewPipeLine Malloc PipeLine
func NewPipeLine(conn *websocket.Conn, command string) (*PipeLine, error) {
	proc, err := console.New(120, 60)
	if err != nil {
		return nil, err
	}
	err = proc.Start([]string{command})
	if err != nil {
		return nil, err
	}
	return &PipeLine{proc, conn, sync.Mutex{}}, nil
}

func NewPipeLineTcp(conn *websocket.Conn, addr string) (*PipeLineTcp, error) {
	connTcp, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &PipeLineTcp{connTcp, conn, sync.Mutex{}}, nil
}

// ReadSktAndWritePty read skt and write pty
func (w *PipeLine) ReadSktAndWritePty(logChan chan string) {
	for {
		w.ResetDeadLine()
		mt, payload, err := w.skt.ReadMessage()
		if IsTimeoutError(err) {
			// logChan <- fmt.Sprintf("Error ReadSktAndWritePty websocket ReadMessage timeout: %s", err)
			logChan <- ""
			return
		}
		if err != nil && err != io.EOF {
			logChan <- fmt.Sprintf("Error ReadSktAndWritePty websocket ReadMessage failed: %s", err)
			return
		}
		// for data terminal show
		if mt == websocket.BinaryMessage {
			_, err = w.pty.Write([]byte(payload))
			if err != nil {
				logChan <- fmt.Sprintf("Error ReadSktAndWritePty pty write [binary] failed: %s", err)
				return
			}
			continue
		}
		//for message
		if mt != websocket.TextMessage {
			logChan <- fmt.Sprintf("Error ReadSktAndWritePty Invalid message type %d - %s", mt, gosystem.GetRuntimeCallerInformation())
			return
		}
		var msg lib.Message
		err = json.Unmarshal(payload, &msg)
		if err != nil {
			logChan <- fmt.Sprintf("Error ReadSktAndWritePty Invalid message %s", err)
			return
		}
		switch msg.Type {
		case lib.TypeResize:
			var size []int
			err := json.Unmarshal(msg.Data, &size)
			if err != nil {
				logChan <- fmt.Sprintf("Error ReadSktAndWritePty Invalid resize message: %s", err)
				return
			}
			err = w.pty.SetSize(size[0], size[1])
			if err != nil {
				logChan <- fmt.Sprintf("Error ReadSktAndWritePty pty resize failed: %s", err)
				return
			}
		case lib.TypeData:
			var dat string
			err := json.Unmarshal(msg.Data, &dat)
			if err != nil {
				logChan <- fmt.Sprintf("Error ReadSktAndWritePty Invalid data message %s", err)
				return
			}
			_, err = w.pty.Write([]byte(dat))
			if err != nil {
				logChan <- fmt.Sprintf("Error ReadSktAndWritePty pty write failed: %s", err)
				return
			}
		case lib.KeepConnect:
			continue
		case lib.Proxy:
			// msg.Data
			// remote := sutils.JsonStringFindElement(msg.Data, "remote")
			// ra := strings.Split(remote, ":")
			// raIP := ra[0]
			// raPort, _ := strconv.Atoi(ra[1])

		case lib.RemoteForward:

		default:
			logChan <- fmt.Sprintf("Error ReadSktAndWritePty Invalid message type %d", mt)
			return
		}
	}
}

// ReadPtyAndWriteSkt read pty and write skt
func (w *PipeLine) ReadPtyAndWriteSkt(logChan chan string) {
	buf := make([]byte, 4096)
	lw := sync.Mutex{}
	go func() {
		pingPeriod := 30 * time.Second
		for {
			lw.Lock()
			err := w.skt.WriteMessage(websocket.PingMessage, []byte{})
			lw.Unlock()
			if err != nil {
				return
			}
			time.Sleep(pingPeriod)
		}
	}()
	for {
		w.ResetDeadLine()
		n, err := w.pty.Read(buf)
		if IsTimeoutError(err) {
			// logChan <- fmt.Sprintf("Error ReadSktAndWritePty websocket ReadMessage timeout: %s", err)
			logChan <- ""
			return
		}
		if err != nil {
			logChan <- fmt.Sprintf("Error ReadPtyAndWriteSkt pty read failed: %s", err)
			return
		}
		lw.Lock()
		err = w.skt.WriteMessage(websocket.BinaryMessage, buf[:n]) //BinaryMessage
		lw.Unlock()
		if err != nil {
			logChan <- fmt.Sprintf("Error ReadPtyAndWriteSkt skt write failed: %s", err)
			return
		}
	}
}

func (w *PipeLine) ResetDeadLine(ds ...time.Duration) {
	// return
	w.Lock()
	defer w.Unlock()
	t := time.Now().Add(3 * time.Hour)
	if len(ds) != 0 {
		t = time.Now().Add(ds[0])
	} else {
		if dur := os.Getenv("SB_TIMEOUT"); len(dur) != 0 {
			d, err := time.ParseDuration(dur + "s")
			if err == nil {
				t = time.Now().Add(d)
			}
		}
	}
	w.skt.SetReadDeadline(t)
	w.skt.SetWriteDeadline(t)
}
