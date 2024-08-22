package client

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/gorilla/websocket"

	"github.com/sonnt85/goshellbox/lib"
)

// PipeLine Connect websocket and childprocess
type PipeLine struct {
	skt *websocket.Conn
}

// NewPipeLine Malloc PipeLine
func NewPipeLine(conn *websocket.Conn) (*PipeLine, error) {
	return &PipeLine{conn}, nil
}

// ReadSktAndWriteStdio read skt and write stdout
func (w *PipeLine) ReadSktAndWriteStdio(logChan chan string, callbackTextMsg ...func(*[]byte)) {
	for {
		mt, payload, err := w.skt.ReadMessage()
		if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
			// logChan <- "Timeout ! Nothing to read from websocket"
			logChan <- ""
			return
		}
		if err != nil && err != io.EOF {
			logChan <- fmt.Sprintf("Error ReadSktAndWriteTer websocket ReadMessage failed: %s", err)
			return
		}
		if mt == websocket.BinaryMessage {
			os.Stdout.Write(payload)
		} else if mt == websocket.TextMessage {
			if len(callbackTextMsg) != 0 && callbackTextMsg[0] != nil {
				callbackTextMsg[0](&payload)
			} else {
				os.Stdout.Write(payload)
			}
		} else {
			logChan <- fmt.Sprintf("Error ReadSktAndWriteTer Invalid message type %d", mt)
			return
		}
	}
}

// // ReadStdioAndWriteSkt read stdin and write skt
// func (w *PipeLine) ReadStdioAndWriteSktTermbox(logChan chan string) {
// 	err := termbox.Init()
// 	if err != nil {
// 		logChan <- fmt.Sprintf("Error ReadTerAndWriteSkt Init Termbox failed: %s", err)
// 		return
// 	}
// 	defer func() {
// 		termbox.Close()
// 		// exec.Command("reset").CombinedOutput()
// 	}()
// 	for {
// 		var msg lib.MessageClient
// 		switch ev := termbox.PollEvent(); ev.Type {
// 		case termbox.EventKey:
// 			if ev.Ch == 0 {
// 				ev.Ch = rune(ev.Key)
// 			}
// 			msg = lib.MessageClient{Type: lib.TypeData, Data: string(ev.Ch)}
// 		case termbox.EventResize:

// 			msg = lib.MessageClient{Type: lib.TypeResize, Data: []int{ev.Width, ev.Height}}
// 		case termbox.EventError:
// 			logChan <- fmt.Sprintf("Error ReadTerAndWriteSkt Termbox PollEvent failed: %s", err)
// 			return
// 		default:
// 			break
// 		}
// 		data, err := json.Marshal(msg)
// 		if err != nil {
// 			logChan <- fmt.Sprintf("Error ReadTerAndWriteSkt json.Marshal failed: %s", err)
// 			return
// 		}
// 		err = w.skt.WriteMessage(websocket.TextMessage, data)
// 		if err != nil {
// 			logChan <- fmt.Sprintf("Error ReadTerAndWriteSkt skt write failed: %s", err)
// 			return
// 		}
// 	}
// }

func keyToBytes(key *tcell.EventKey) []byte {
	switch key.Key() {
	case tcell.KeyUp:
		return []byte("\x1b[A")
	case tcell.KeyDown:
		return []byte("\x1b[B")
	case tcell.KeyRight:
		return []byte("\x1b[C")
	case tcell.KeyLeft:
		return []byte("\x1b[D")
	default:
		if key.Rune() != 0 {
			return []byte(string(key.Rune()))
		} else {
			return []byte{byte(key.Key())}
		}
	}
}

// ReadStdioAndWriteSkt reads stdin and writes to the websocket
func (w *PipeLine) ReadStdioAndWriteSktTcell(s tcell.Screen, logChan chan string) {
	var err error
	if s == nil {
		s, err = tcell.NewScreen()
		if err != nil {
			logChan <- fmt.Sprintf("Error ReadStdioAndWriteSkt NewScreen failed: %s", err)
			return
		}
		if err := s.Init(); err != nil {
			logChan <- fmt.Sprintf("Error ReadStdioAndWriteSkt Init failed: %s", err)
			return
		}
		defer func() {
			s.Clear()
			s.Fini()
			// time.Sleep(time.Second * 10)
		}()
	}
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	s.SetStyle(defStyle)
	s.EnableMouse()
	s.EnablePaste()
	s.SetCursorStyle(tcell.CursorStyleSteadyBlock)
	s.Clear()
	for {
		var msg lib.MessageClient
		s.Show()
		ev := s.PollEvent()
		msgSendType := -1 //websocket.BinaryMessage
		switch ev := ev.(type) {
		case *tcell.EventKey:
			err = w.skt.WriteMessage(websocket.BinaryMessage, keyToBytes(ev))
			if err != nil {
				logChan <- fmt.Sprintf("Error ReadStdioAndWriteSkt skt write failed: %s", err)
				return
			}
			continue
			var bytes []byte
			msg = lib.MessageClient{Type: lib.TypeData, Data: bytes}
			key := ev.Key()
			var keystring string
			if key == tcell.KeyRune {
				keystring = string(ev.Rune())
			} else {
				// keystring = tcell.KeyNames[key] // key
				// keystring = fmt.Sprintf("%d", key)
				// if key == tcell.KeyUp || key == tcell.KeyDown || key == tcell.KeyLeft || key == tcell.KeyRight {
				// bytes := []byte{byte(key >> 8), byte(key & 0xFF)}
				// keystring = string(bytes)
				// r, _ := utf8.DecodeRuneInString(key)
				// utf8.Rune
				// keystring, _ = utf8.DecodeRune(key)

				// }
				keystring = string(key)
			}
			msg = lib.MessageClient{Type: lib.TypeData, Data: keystring}
			msgSendType = websocket.TextMessage
		case *tcell.EventResize:
			w, h := ev.Size()
			msg = lib.MessageClient{Type: lib.TypeResize, Data: []int{int(w), int(h)}}
			msgSendType = websocket.TextMessage
			s.Sync()
		case *tcell.EventError:
			logChan <- fmt.Sprintf("Error ReadTerAndWriteSkt Tcell PollEvent failed: %s", err)
			return
		case *tcell.EventMouse:
			return
		default:
			msgSendType = websocket.BinaryMessage
		}
		data, err := json.Marshal(msg)
		if err != nil {
			logChan <- fmt.Sprintf("Error ReadStdioAndWriteSkt json.Marshal failed: %s", err)
			return
		}
		err = w.skt.WriteMessage(msgSendType, data)
		if err != nil {
			logChan <- fmt.Sprintf("Error ReadStdioAndWriteSkt skt write failed: %s", err)
			return
		}
	}
}
