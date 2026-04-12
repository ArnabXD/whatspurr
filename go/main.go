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
	"syscall"

	"github.com/coder/websocket"
	_ "modernc.org/sqlite"

	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

// bridgeLog is the level-aware logger for all bridge code.
var bridgeLog waLog.Logger

var (
	authToken         string
	sessionDir        string
	dbName            string
	logLevel          string
	autoPresence      bool
	subscribeOutgoing bool
)

func main() {
	flag.StringVar(&authToken, "token", "", "Auth token for WebSocket connections (required)")
	flag.StringVar(&sessionDir, "session-dir", "./session", "Session data directory")
	flag.StringVar(&dbName, "db-name", "whatspurr.db", "SQLite database filename")
	flag.StringVar(&logLevel, "log-level", "info", "Log level: debug, info, warn, error")
	flag.BoolVar(&autoPresence, "auto-presence", false, "Send available presence on connect")
	flag.BoolVar(&subscribeOutgoing, "subscribe-outgoing", false, "Forward outgoing (sent by us) messages to TS")
	flag.Parse()

	if authToken == "" {
		log.Fatal("--token flag is required")
	}

	bridgeLog = waLog.Stdout("Bridge", logLevel, true)

	if strings.ContainsAny(dbName, "/\\") || strings.Contains(dbName, "..") || dbName == "" {
		log.Fatal("--db-name must be a plain filename (no path separators or '..')")
	}

	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		log.Fatalf("Failed to create session dir: %v", err)
	}

	// Init SQLite store — shared across all sessions
	dbPath := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)", filepath.Join(sessionDir, dbName))
	dbLog := waLog.Stdout("Database", logLevel, true)
	container, err := sqlstore.New(context.Background(), "sqlite", dbPath, dbLog)
	if err != nil {
		log.Fatalf("Failed to init store: %v", err)
	}

	// Create session manager (no sessions started yet — on-demand only)
	manager := NewSessionManager(container)

	// Listen on localhost with random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	// HTTP server for WebSocket upgrade
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token") != authToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Single WS connection mode
		manager.connMu.Lock()
		if manager.conn != nil {
			manager.connMu.Unlock()
			http.Error(w, "already connected", http.StatusConflict)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
		if err != nil {
			manager.connMu.Unlock()
			bridgeLog.Errorf("WebSocket accept error: %v", err)
			return
		}

		manager.conn = conn
		manager.connMu.Unlock()

		manager.serveConnected(conn)
	})

	server := &http.Server{Handler: mux}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP serve error: %v", err)
		}
	}()

	fmt.Printf("ready %s\n", listener.Addr().String())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh

	bridgeLog.Infof("Shutting down...")
	manager.disconnectAll()
	server.Shutdown(context.Background())
}
