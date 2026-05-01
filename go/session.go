package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/coder/websocket"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

const (
	maxConcurrentCommands = 64
	wsReadTimeout         = 5 * time.Minute
	wsWriteTimeout        = 30 * time.Second
	wsReadLimit           = 135 * 1024 * 1024 // max doc (100MB) * ~1.34 base64 overhead
)

// Command is a request from TS to Go.
type Command struct {
	ID      string                 `json:"id"`
	Session string                 `json:"session,omitempty"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
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
	Type      string      `json:"type"`
	Session   string      `json:"session"`
	EventName string      `json:"event"`
	Data      interface{} `json:"data"`
}

// Session manages a single whatsmeow client.
type Session struct {
	name     string
	client   *whatsmeow.Client
	manager  *SessionManager
	handlers map[string]commandHandler
}

// SessionManager manages multiple sessions and the shared WS connection.
type SessionManager struct {
	container  *sqlstore.Container
	sessions   map[string]*Session
	mu         sync.RWMutex
	conn       *websocket.Conn
	connMu     sync.Mutex
	nameMap    map[string]string // session name → JID string
	nameMapMu  sync.Mutex
	mapFile    string
}

// NewSessionManager creates a new session manager.
func NewSessionManager(container *sqlstore.Container) *SessionManager {
	mapFile := filepath.Join(sessionDir, "sessions.json")
	m := &SessionManager{
		container: container,
		sessions:  make(map[string]*Session),
		nameMap:   make(map[string]string),
		mapFile:   mapFile,
	}
	m.loadNameMap()
	return m
}

// loadNameMap reads the session name → JID mapping from disk.
func (m *SessionManager) loadNameMap() {
	data, err := os.ReadFile(m.mapFile)
	if err != nil {
		return // file doesn't exist yet, that's fine
	}
	if err := json.Unmarshal(data, &m.nameMap); err != nil {
		bridgeLog.Warnf("Failed to parse sessions.json: %v", err)
	}
}

// saveNameMap persists the session name → JID mapping to disk.
// Must be called with nameMapMu held.
func (m *SessionManager) saveNameMap() {
	data, err := json.MarshalIndent(m.nameMap, "", "  ")
	if err != nil {
		bridgeLog.Warnf("Failed to marshal sessions.json: %v", err)
		return
	}
	if err := os.WriteFile(m.mapFile, data, 0600); err != nil {
		bridgeLog.Warnf("Failed to write sessions.json: %v", err)
	}
}

// setNameMapping stores a name → JID mapping and saves to disk.
func (m *SessionManager) setNameMapping(name string, jid string) {
	m.nameMapMu.Lock()
	defer m.nameMapMu.Unlock()
	m.nameMap[name] = jid
	m.saveNameMap()
}

// removeNameMapping deletes a name mapping and saves to disk.
func (m *SessionManager) removeNameMapping(name string) {
	m.nameMapMu.Lock()
	defer m.nameMapMu.Unlock()
	delete(m.nameMap, name)
	m.saveNameMap()
}

// getJIDForName returns the JID string for a session name, if mapped.
func (m *SessionManager) getJIDForName(name string) (string, bool) {
	m.nameMapMu.Lock()
	defer m.nameMapMu.Unlock()
	jid, ok := m.nameMap[name]
	return jid, ok
}

// serveConnected runs the read pump for an already-accepted connection.
func (m *SessionManager) serveConnected(conn *websocket.Conn) {
	defer func() {
		m.connMu.Lock()
		m.conn = nil
		m.connMu.Unlock()
		conn.Close(websocket.StatusNormalClosure, "")
	}()

	conn.SetReadLimit(wsReadLimit)

	sem := make(chan struct{}, maxConcurrentCommands)

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

		if bytes.Contains(data, []byte(`"ping"`)) {
			continue
		}

		var cmd Command
		if err := json.Unmarshal(data, &cmd); err != nil {
			m.sendResponse(Response{
				Error: &ErrorInfo{Code: 1000, Message: "invalid JSON"},
			})
			continue
		}

		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			m.handleCommand(cmd)
		}()
	}
}

func (m *SessionManager) handleCommand(cmd Command) {
	var resp Response

	switch cmd.Method {
	case "connect_session":
		resp = m.cmdConnectSession(cmd)
	case "disconnect_session":
		resp = m.cmdDisconnectSession(cmd)
	case "destroy_session":
		resp = m.cmdDestroySession(cmd)
	case "list_sessions":
		resp = m.cmdListSessions(cmd)
	default:
		// Route to the named session
		sessionName := cmd.Session
		if sessionName == "" {
			sessionName = "default"
		}

		m.mu.RLock()
		session, ok := m.sessions[sessionName]
		m.mu.RUnlock()

		if !ok {
			resp = Response{Error: &ErrorInfo{Code: 1010, Message: fmt.Sprintf("session '%s' not found or not connected", sessionName)}}
		} else {
			if handler, exists := session.handlers[cmd.Method]; exists {
				resp = handler(cmd)
			} else {
				resp = Response{Error: &ErrorInfo{Code: 1002, Message: fmt.Sprintf("unknown method: %s", cmd.Method)}}
			}
		}
	}

	resp.ID = cmd.ID
	m.sendResponse(resp)
}

func (m *SessionManager) cmdConnectSession(cmd Command) Response {
	name, _ := cmd.Params["name"].(string)
	if name == "" {
		return Response{Error: &ErrorInfo{Code: 1003, Message: "missing 'name' parameter"}}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Already connected
	if _, ok := m.sessions[name]; ok {
		return Response{Result: map[string]interface{}{"status": "already_connected"}}
	}

	// Try to find an existing device with this name, or create a new one
	device, err := m.getOrCreateDevice(name)
	if err != nil {
		return Response{Error: &ErrorInfo{Code: 1011, Message: fmt.Sprintf("failed to get device: %v", err)}}
	}

	clientLog := waLog.Stdout(fmt.Sprintf("Client/%s", name), logLevel, true)
	client := whatsmeow.NewClient(device, clientLog)

	session := &Session{
		name:    name,
		client:  client,
		manager: m,
	}
	session.handlers = session.buildCommandHandlers()

	client.AddEventHandler(session.handleWhatsmeowEvent)

	m.sessions[name] = session

	// Connect in a goroutine to not block the command
	go session.connectWhatsmeow()

	return Response{Result: map[string]interface{}{"status": "connecting"}}
}

func (m *SessionManager) cmdDisconnectSession(cmd Command) Response {
	name, _ := cmd.Params["name"].(string)
	if name == "" {
		return Response{Error: &ErrorInfo{Code: 1003, Message: "missing 'name' parameter"}}
	}

	m.mu.Lock()
	session, ok := m.sessions[name]
	if !ok {
		m.mu.Unlock()
		return Response{Error: &ErrorInfo{Code: 1010, Message: fmt.Sprintf("session '%s' not found", name)}}
	}
	delete(m.sessions, name)
	m.mu.Unlock()

	session.client.Disconnect()

	return Response{Result: map[string]interface{}{"ok": true}}
}

func (m *SessionManager) cmdDestroySession(cmd Command) Response {
	name, _ := cmd.Params["name"].(string)
	if name == "" {
		return Response{Error: &ErrorInfo{Code: 1003, Message: "missing 'name' parameter"}}
	}

	m.mu.Lock()
	session, ok := m.sessions[name]
	if ok {
		delete(m.sessions, name)
	}
	m.mu.Unlock()

	if ok {
		session.client.Logout(context.Background())
	} else if jidStr, mapped := m.getJIDForName(name); mapped {
		// Not connected, but we know the JID — find and delete the device from DB
		devices, err := m.container.GetAllDevices(context.Background())
		if err == nil {
			for _, d := range devices {
				if d.ID != nil && d.ID.String() == jidStr {
					if err := d.Delete(context.Background()); err != nil {
						bridgeLog.Warnf("Failed to delete device for session '%s': %v", name, err)
					}
				}
			}
		}
	}

	m.removeNameMapping(name)
	return Response{Result: map[string]interface{}{"ok": true}}
}

func (m *SessionManager) cmdListSessions(cmd Command) Response {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get all devices from DB
	devices, err := m.container.GetAllDevices(context.Background())
	if err != nil {
		return Response{Error: &ErrorInfo{Code: 1011, Message: fmt.Sprintf("failed to list devices: %v", err)}}
	}

	// Build reverse map: JID → session name
	m.nameMapMu.Lock()
	jidToName := make(map[string]string, len(m.nameMap))
	for n, j := range m.nameMap {
		jidToName[j] = n
	}
	m.nameMapMu.Unlock()

	sessions := make([]map[string]interface{}, 0, len(devices))
	for _, d := range devices {
		jid := ""
		name := ""
		if d.ID != nil {
			jid = d.ID.String()
			name = jidToName[jid]
		}
		if name == "" {
			// Unmapped device — use JID user part as fallback
			if d.ID != nil {
				name = d.ID.User
			}
		}

		_, connected := m.sessions[name]
		sessions = append(sessions, map[string]interface{}{
			"name":      name,
			"jid":       jid,
			"connected": connected,
		})
	}

	return Response{Result: map[string]interface{}{"sessions": sessions}}
}

// getOrCreateDevice finds a device by session name (via mapping) or creates a new one.
func (m *SessionManager) getOrCreateDevice(name string) (*store.Device, error) {
	devices, err := m.container.GetAllDevices(context.Background())
	if err != nil {
		return nil, err
	}

	// Check name → JID mapping first
	if jidStr, ok := m.getJIDForName(name); ok {
		for _, d := range devices {
			if d.ID != nil && d.ID.String() == jidStr {
				return d, nil
			}
		}
		// Mapping exists but device is gone — remove stale mapping
		m.removeNameMapping(name)
	}

	// For "default" session, use the first device if it exists (backward compat)
	if name == "default" && len(devices) > 0 {
		return devices[0], nil
	}

	// Create new device
	return m.container.NewDevice(), nil
}

func (m *SessionManager) writeJSON(v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		bridgeLog.Errorf("Failed to marshal: %v", err)
		return
	}

	m.connMu.Lock()
	defer m.connMu.Unlock()
	if m.conn == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), wsWriteTimeout)
	defer cancel()
	if err := m.conn.Write(ctx, websocket.MessageText, data); err != nil {
		bridgeLog.Warnf("Failed to write: %v", err)
	}
}

func (m *SessionManager) sendResponse(resp Response) {
	m.writeJSON(resp)
}

func (m *SessionManager) sendEvent(session string, eventName string, eventData interface{}) {
	m.writeJSON(Event{
		Type:      "event",
		Session:   session,
		EventName: eventName,
		Data:      eventData,
	})
}

// disconnectAll disconnects all active sessions.
func (m *SessionManager) disconnectAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, session := range m.sessions {
		session.client.Disconnect()
		delete(m.sessions, name)
	}
}