package main

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/coder/websocket"
	"go.mau.fi/whatsmeow"
)

const (
	// Max concurrent command handlers
	maxConcurrentCommands = 64
	// Read deadline: if no message received within this duration, close
	wsReadTimeout = 5 * time.Minute
	// Write deadline: if a write takes longer than this, drop it
	wsWriteTimeout = 30 * time.Second
	// Max incoming WS message size: 100 MB document * 4/3 base64 overhead + JSON envelope
	wsReadLimit = 140 * 1024 * 1024
)

// Command is a request from TS to Go.
type Command struct {
	ID     string                 `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

// Response is Go's reply to a TS command.
type Response struct {
	ID     string                 `json:"id"`
	Result map[string]interface{} `json:"result,omitempty"`
	Error  *ErrorInfo             `json:"error,omitempty"`
}

// ErrorInfo describes a command error.
type ErrorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Event is a push message from Go to TS.
type Event struct {
	Type      string      `json:"type"` // always "event"
	EventName string      `json:"event"`
	Data      interface{} `json:"data"`
}

// Session manages a single WebSocket connection and the whatsmeow client.
type Session struct {
	client   *whatsmeow.Client
	conn     *websocket.Conn
	mu       sync.Mutex
	handlers map[string]commandHandler
}

// serveConnected runs the read pump for an already-accepted connection.
// The caller (HTTP handler) has already set s.conn and s.connected under the lock.
func (s *Session) serveConnected(conn *websocket.Conn) {
	defer func() {
		s.mu.Lock()
		s.conn = nil
		s.mu.Unlock()
		conn.Close(websocket.StatusNormalClosure, "")
	}()

	conn.SetReadLimit(wsReadLimit)

	// Semaphore to bound concurrent command handlers
	sem := make(chan struct{}, maxConcurrentCommands)

	// Connect whatsmeow when TS connects
	s.connectWhatsmeow()

	// Read pump with deadline
	for {
		ctx, cancel := context.WithTimeout(context.Background(), wsReadTimeout)
		_, data, err := conn.Read(ctx)
		cancel()
		if err != nil {
			status := websocket.CloseStatus(err)
			if status == websocket.StatusNormalClosure || status == websocket.StatusGoingAway {
				bridgeLog.Infof("WS connection closed")
			} else {
				bridgeLog.Warnf("WS read error: %v", err)
			}
			return
		}

		var cmd Command
		if err := json.Unmarshal(data, &cmd); err != nil {
			s.sendResponse(Response{
				Error: &ErrorInfo{Code: 1000, Message: "invalid JSON"},
			})
			continue
		}

		// Acquire semaphore slot (bounded concurrency)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			s.handleCommand(cmd)
		}()
	}
}

func (s *Session) writeJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		bridgeLog.Errorf("Failed to marshal: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), wsWriteTimeout)
	defer cancel()
	if err := s.conn.Write(ctx, websocket.MessageText, data); err != nil {
		bridgeLog.Warnf("Failed to write: %v", err)
	}
}

func (s *Session) sendResponse(resp Response) {
	s.writeJSON(resp)
}

func (s *Session) sendEvent(eventName string, eventData interface{}) {
	s.writeJSON(Event{
		Type:      "event",
		EventName: eventName,
		Data:      eventData,
	})
}