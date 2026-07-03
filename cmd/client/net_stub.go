//go:build !js || !wasm

package main

import (
	"fmt"

	"packov/internal/protocol"
)

type wsClient struct{}

func newWSClient() (*wsClient, error)                    { return nil, fmt.Errorf("browser websocket unavailable") }
func (c *wsClient) send(protocol.ClientMessage)          {}
func (c *wsClient) next() (protocol.ServerMessage, bool) { return protocol.ServerMessage{}, false }
func (c *wsClient) isOpen() bool                         { return false }
func (c *wsClient) isClosed() bool                       { return true }
func browserToken() string                               { return "local-pilot" }
