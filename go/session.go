package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/coder/websocket"
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
	client    interface{} // *whatsmeow.Client — typed in main.go
	conn      *websocket.Conn
	mu        sync.Mutex
	connected bool
}

func (s *Session) isConnected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connected
}

func (s *Session) serve(conn *websocket.Conn) {
	s.mu.Lock()
	s.conn = conn
	s.connected = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.conn = nil
		s.connected = false
		s.mu.Unlock()
		conn.Close(websocket.StatusNormalClosure, "")
	}()

	ctx := context.Background()

	// Connect whatsmeow when TS connects
	s.connectWhatsmeow()

	// Read pump
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			status := websocket.CloseStatus(err)
			if status == websocket.StatusNormalClosure || status == websocket.StatusGoingAway {
				log.Println("WS connection closed")
			} else {
				log.Printf("WS read error: %v", err)
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

		// Handle command in goroutine to not block reads
		go s.handleCommand(cmd)
	}
}

func (s *Session) sendResponse(resp Response) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn == nil {
		return
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	if err := s.conn.Write(context.Background(), websocket.MessageText, data); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

func (s *Session) sendEvent(eventName string, eventData interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn == nil {
		return
	}

	ev := Event{
		Type:      "event",
		EventName: eventName,
		Data:      eventData,
	}

	data, err := json.Marshal(ev)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return
	}

	if err := s.conn.Write(context.Background(), websocket.MessageText, data); err != nil {
		log.Printf("Failed to write event: %v", err)
	}
}