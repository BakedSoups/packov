//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"packov/internal/protocol"
)

type wsClient struct {
	conn      js.Value
	inbox     chan protocol.ServerMessage
	open      bool
	closed    bool
	callbacks []js.Func
}

func newWSClient() (*wsClient, error) {
	url := wsURL()
	conn := js.Global().Get("WebSocket").New(url)
	c := &wsClient{conn: conn, inbox: make(chan protocol.ServerMessage, 256)}
	onOpen := js.FuncOf(func(this js.Value, args []js.Value) any {
		c.open = true
		return nil
	})
	onClose := js.FuncOf(func(this js.Value, args []js.Value) any {
		c.closed = true
		c.open = false
		return nil
	})
	onMessage := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		data := args[0].Get("data").String()
		var msg protocol.ServerMessage
		if err := json.Unmarshal([]byte(data), &msg); err == nil {
			select {
			case c.inbox <- msg:
			default:
			}
		}
		return nil
	})
	onError := js.FuncOf(func(this js.Value, args []js.Value) any {
		c.closed = true
		c.open = false
		return nil
	})
	c.callbacks = []js.Func{onOpen, onClose, onMessage, onError}
	conn.Set("onopen", onOpen)
	conn.Set("onclose", onClose)
	conn.Set("onmessage", onMessage)
	conn.Set("onerror", onError)
	return c, nil
}

func (c *wsClient) send(msg protocol.ClientMessage) {
	if c == nil || !c.isOpen() {
		return
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}
	c.conn.Call("send", string(b))
}

func (c *wsClient) next() (protocol.ServerMessage, bool) {
	select {
	case msg := <-c.inbox:
		return msg, true
	default:
		return protocol.ServerMessage{}, false
	}
}

func (c *wsClient) isOpen() bool {
	return c != nil && c.open && c.conn.Get("readyState").Int() == 1
}

func (c *wsClient) isClosed() bool {
	return c == nil || c.closed || c.conn.Get("readyState").Int() >= 2
}

func wsURL() string {
	location := js.Global().Get("location")
	proto := "ws:"
	if location.Get("protocol").String() == "https:" {
		proto = "wss:"
	}
	return fmt.Sprintf("%s//%s/ws", proto, location.Get("host").String())
}

func browserToken() string {
	storage := js.Global().Get("localStorage")
	token := storage.Call("getItem", "packov_token").String()
	if token != "" && token != "<null>" && token != "null" {
		return token
	}
	token = fmt.Sprintf("pilot-%d", time.Now().UnixNano())
	storage.Call("setItem", "packov_token", token)
	return token
}
