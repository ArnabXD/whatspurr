package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/coder/websocket"
	_ "modernc.org/sqlite"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// bridgeLog is the level-aware logger for all bridge code.
// Initialized in main() after flag parsing.
var bridgeLog waLog.Logger

var (
	authToken    string
	sessionDir   string
	dbName       string
	logLevel     string
	autoPresence bool
)

func main() {
	flag.StringVar(&authToken, "token", "", "Auth token for WebSocket connections (required)")
	flag.StringVar(&sessionDir, "session-dir", "./session", "Session data directory")
	flag.StringVar(&dbName, "db-name", "whatspurr.db", "SQLite database filename")
	flag.StringVar(&logLevel, "log-level", "info", "Log level: debug, info, warn, error")
	flag.BoolVar(&autoPresence, "auto-presence", false, "Send available presence on connect")
	flag.Parse()

	if authToken == "" {
		log.Fatal("--token flag is required")
	}

	// Init level-aware logger for bridge code
	bridgeLog = waLog.Stdout("Bridge", logLevel, true)

	// Validate dbName: must not contain path separators or traversal
	if strings.ContainsAny(dbName, "/\\") || strings.Contains(dbName, "..") || dbName == "" {
		log.Fatal("--db-name must be a plain filename (no path separators or '..')")
	}

	// Ensure session directory exists
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		log.Fatalf("Failed to create session dir: %v", err)
	}

	// Init SQLite store for whatsmeow auth
	dbPath := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)", filepath.Join(sessionDir, dbName))
	dbLog := waLog.Stdout("Database", logLevel, true)
	container, err := sqlstore.New(context.Background(), "sqlite", dbPath, dbLog)
	if err != nil {
		log.Fatalf("Failed to init store: %v", err)
	}

	// Get or create device store
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		log.Fatalf("Failed to get device: %v", err)
	}

	// Create whatsmeow client
	clientLog := waLog.Stdout("Client", logLevel, true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// Create session to manage the single WS connection
	session := &Session{
		client: client,
		mu:     sync.Mutex{},
	}

	// Register whatsmeow event handler
	client.AddEventHandler(session.handleWhatsmeowEvent)

	// Listen on localhost with random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	// HTTP server for WebSocket upgrade
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Verify auth token from query param
		if r.URL.Query().Get("token") != authToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Single-client mode: acquire lock for atomic check-and-accept
		session.mu.Lock()
		if session.connected {
			session.mu.Unlock()
			http.Error(w, "already connected", http.StatusConflict)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			session.mu.Unlock()
			bridgeLog.Errorf("WebSocket accept error: %v", err)
			return
		}

		session.conn = conn
		session.connected = true
		session.mu.Unlock()

		session.serveConnected(conn)
	})

	server := &http.Server{Handler: mux}

	// Start serving in background
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP serve error: %v", err)
		}
	}()

	// Signal readiness: "ready <addr>" so TS knows the port
	fmt.Printf("ready %s\n", listener.Addr().String())

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh

	bridgeLog.Infof("Shutting down...")
	client.Disconnect()
	server.Shutdown(context.Background())
}
