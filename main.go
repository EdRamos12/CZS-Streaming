// Tiny state server: control page POSTs state, overlays receive it live via Server-Sent Events.
// Run from the project folder: it reads public/ and state.json from the working directory.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const (
	public  = "public"
	file    = "state.json"
	uploads = "public/uploads"
)

var (
	image   = regexp.MustCompile(`(?i)\.(png|jpe?g|webp|gif)$`) // no SVG: it could run script on our origin
	badChar = regexp.MustCompile(`[^\w.-]`)
)

const defaultState = `{"title":"Torneio CZS","subtitle":"","casters":"","startScore":25000,"banners":[],"sponsors":[],"rotateSecs":8,` +
	`"round":{"wind":0,"num":1,"honba":0,"riichi":0},"seats":[{"name":"","score":25000},{"name":"","score":25000},{"name":"","score":25000},{"name":"","score":25000}],"players":[]}`

type server struct {
	mu      sync.Mutex
	state   []byte // compact JSON, kept as-is so unknown fields survive
	file    string
	clients map[chan struct{}]bool
}

// valid mirrors the old check: an object with round, 4 seats and a players array.
func valid(b []byte) bool {
	var s struct {
		Round   json.RawMessage
		Seats   []json.RawMessage
		Players []json.RawMessage
	}
	if json.Unmarshal(b, &s) != nil {
		return false
	}
	r := string(s.Round)
	return r != "" && r != "null" && r != "false" && r != "0" && r != `""` && len(s.Seats) == 4 && s.Players != nil
}

func (s *server) current() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *server) postState(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1e6))
	var compact bytes.Buffer
	if err != nil || !valid(body) || json.Compact(&compact, body) != nil {
		http.Error(w, "invalid state", 400)
		return
	}
	var pretty bytes.Buffer
	json.Indent(&pretty, compact.Bytes(), "", "  ")

	s.mu.Lock()
	s.state = compact.Bytes()
	// temp file + rename so a crash mid-write can't wipe state.json
	if err := os.WriteFile(s.file+".tmp", pretty.Bytes(), 0o644); err != nil {
		log.Println("save failed:", err)
	} else if err := os.Rename(s.file+".tmp", s.file); err != nil {
		log.Println("save failed:", err)
	}
	for c := range s.clients {
		select {
		case c <- struct{}{}:
		default: // already has a pending wake-up; it will read the latest state
		}
	}
	s.mu.Unlock()
	io.WriteString(w, "ok")
}

func (s *server) events(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	c := make(chan struct{}, 1)
	s.mu.Lock()
	s.clients[c] = true
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.clients, c); s.mu.Unlock() }()

	rc := http.NewResponseController(w)
	for {
		if _, err := fmt.Fprintf(w, "data: %s\n\n", s.current()); err != nil || rc.Flush() != nil {
			return
		}
		select {
		case <-c:
		case <-r.Context().Done():
			return
		}
	}
}

// Banner/sponsor images: raw file body, saved to public/uploads/. The control page lists them in state.
func upload(w http.ResponseWriter, r *http.Request) {
	name := strconv.FormatInt(time.Now().UnixMilli(), 10) + "-" + badChar.ReplaceAllString(path.Base(r.URL.Query().Get("name")), "_")
	if !image.MatchString(name) {
		http.Error(w, "only png, jpg, webp or gif", 400)
		return
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10e6))
	if err != nil {
		http.Error(w, "file over 10 MB", 413)
		return
	}
	if err := os.WriteFile(filepath.Join(uploads, name), data, 0o644); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	io.WriteString(w, name)
}

func deleteUpload(w http.ResponseWriter, r *http.Request) {
	name := path.Base(r.PathValue("name"))
	if !image.MatchString(name) {
		w.WriteHeader(400)
		return
	}
	os.Remove(filepath.Join(uploads, name)) // already gone is fine
	io.WriteString(w, "ok")
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/state", s.postState)
	mux.HandleFunc("GET /api/events", s.events)
	mux.HandleFunc("POST /api/upload", upload)
	mux.HandleFunc("DELETE /api/upload/{name}", deleteUpload)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, filepath.Join(public, "control.html")) })
	mux.Handle("GET /", http.FileServer(http.Dir(public)))
	return mux
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	port := env("PORT", "8787")
	host := env("HOST", "127.0.0.1") // set HOST=0.0.0.0 to control from another device on the LAN
	if err := os.MkdirAll(uploads, 0o755); err != nil {
		log.Fatal(err)
	}

	s := &server{file: file, clients: map[chan struct{}]bool{}, state: []byte(defaultState)}
	var compact bytes.Buffer
	if b, err := os.ReadFile(file); err == nil && json.Compact(&compact, b) == nil {
		s.state = compact.Bytes()
	}

	fmt.Printf("Control panel:    http://localhost:%s/\n", port)
	fmt.Printf("OBS (in-game):    http://localhost:%s/overlay.html\n", port)
	fmt.Printf("OBS (break):      http://localhost:%s/ranking.html\n", port)
	fmt.Printf("OBS (commentary): http://localhost:%s/cams.html\n", port)
	err := http.ListenAndServe(host+":"+port, s.handler())
	if errors.Is(err, syscall.EADDRINUSE) {
		log.Fatalf("Port %s is already in use by another program. Try: PORT=3000 go run .", port)
	}
	log.Fatal(err)
}
