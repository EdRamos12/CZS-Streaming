// Tiny state server: control page POSTs state, overlays receive it live via Server-Sent Events.
// Run from the project folder: it reads public/ and state.json from the working directory.
package main

import (
	"bytes"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
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
	pwFile  = "password.json" // delete it (or run with -reset-password) to set a new password
)

var (
	image   = regexp.MustCompile(`(?i)\.(png|jpe?g|webp|gif)$`) // no SVG: it could run script on our origin
	badChar = regexp.MustCompile(`[^\w.-]`)
)

const defaultState = `{"title":"Torneio CZS","subtitle":"","casters":"","startScore":25000,"banners":[],"sponsors":[],"rotateSecs":8,` +
	`"round":{"wind":0,"num":1,"honba":0,"riichi":0},"seats":[{"name":"","score":25000},{"name":"","score":25000},{"name":"","score":25000},{"name":"","score":25000}],"players":[]}`

type server struct {
	mu       sync.Mutex
	state    []byte // compact JSON, kept as-is so unknown fields survive
	file     string
	clients  map[chan struct{}]bool
	pwFile   string
	pw       *password           // nil until the owner sets one
	anySetup bool                // -remote-setup: the first password may be set from any computer
	fails    map[string]failures // wrong logins per IP address
}

type failures struct {
	n     int
	until time.Time
}

// Per address, so someone guessing from the LAN can't lock the owner out.
const (
	maxFails = 5
	lockFor  = 5 * time.Minute
)

// password is stored as a salted PBKDF2 hash. Session is the cookie value; it changes with the password,
// so a reset logs every browser out.
type password struct {
	Salt, Hash, Session string
}

func hashPw(pw, salt string) string {
	k, _ := pbkdf2.Key(sha256.New, pw, []byte(salt), 600_000, 32)
	return hex.EncodeToString(k)
}

func (s *server) authed(r *http.Request) bool {
	s.mu.Lock()
	pw := s.pw
	s.mu.Unlock()
	c, err := r.Cookie("czs")
	return pw != nil && err == nil && subtle.ConstantTimeCompare([]byte(c.Value), []byte(pw.Session)) == 1
}

// guard: APIs answer 401, pages redirect to the login page.
func (s *server) guard(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.authed(r) {
			h(w, r)
		} else if r.Method == "GET" {
			http.Redirect(w, r, "/login.html", http.StatusSeeOther)
		} else {
			http.Error(w, "login required", 401)
		}
	}
}

func loopback(r *http.Request) bool {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// GET /api/auth tells the login page whether to ask for a new password or the existing one.
func (s *server) authInfo(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	setup := s.pw == nil
	s.mu.Unlock()
	fmt.Fprintf(w, `{"setup":%t,"local":%t}`, setup, s.anySetup || loopback(r))
}

// POST /api/login: checks the password, or sets it if none exists yet (only from this computer,
// so nobody on the LAN can claim the panel first).
func (s *server) login(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1000))
	given := string(body)
	if err != nil || given == "" {
		http.Error(w, "empty password", 400)
		return
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fails == nil {
		s.fails = map[string]failures{}
	}
	f := s.fails[ip]
	if time.Now().Before(f.until) {
		http.Error(w, fmt.Sprintf("too many wrong passwords, try again in %d minutes", int(time.Until(f.until).Minutes())+1), 429)
		return
	}
	if s.pw == nil {
		if !s.anySetup && !loopback(r) {
			http.Error(w, "set the password from the computer running the server", 403)
			return
		}
		if len(given) < 6 {
			http.Error(w, "password must have at least 6 characters", 400)
			return
		}
		pw := &password{Salt: rand.Text(), Session: rand.Text()}
		pw.Hash = hashPw(given, pw.Salt)
		b, _ := json.Marshal(pw)
		if err := os.WriteFile(s.pwFile, b, 0o600); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		s.pw = pw
	} else if subtle.ConstantTimeCompare([]byte(hashPw(given, s.pw.Salt)), []byte(s.pw.Hash)) != 1 {
		if f.n++; f.n >= maxFails {
			f = failures{until: time.Now().Add(lockFor)}
		}
		s.fails[ip] = f
		http.Error(w, "wrong password", 403)
		return
	}
	delete(s.fails, ip)
	http.SetCookie(w, &http.Cookie{Name: "czs", Value: s.pw.Session, Path: "/", MaxAge: 365 * 24 * 3600, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	io.WriteString(w, "ok")
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
	mux.HandleFunc("POST /api/state", s.guard(s.postState))
	mux.HandleFunc("GET /api/events", s.events) // overlays (OBS) stay open: they only read
	mux.HandleFunc("POST /api/upload", s.guard(upload))
	mux.HandleFunc("DELETE /api/upload/{name}", s.guard(deleteUpload))
	mux.HandleFunc("GET /api/auth", s.authInfo)
	mux.HandleFunc("POST /api/login", s.login)
	page := func(f string) http.HandlerFunc {
		return s.guard(func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, filepath.Join(public, f)) })
	}
	mux.HandleFunc("GET /{$}", page("control.html"))
	mux.HandleFunc("GET /control.html", page("control.html"))
	mux.HandleFunc("GET /media.html", page("media.html"))
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
	reset := flag.Bool("reset-password", false, "forget the control panel password; the next visit from this computer sets a new one")
	remoteSetup := flag.Bool("remote-setup", false, "let any computer on the network create the control panel password (only while none is set)")
	flag.Parse()
	if *reset {
		if err := os.Remove(pwFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Fatal(err)
		}
		fmt.Println("Password removed. Start the server and open the control panel on this computer to set a new one.")
		return
	}
	port := env("PORT", "8787")
	host := env("HOST", "0.0.0.0") // reachable from the LAN by default; set HOST=127.0.0.1 to allow only this computer
	if err := os.MkdirAll(uploads, 0o755); err != nil {
		log.Fatal(err)
	}

	s := &server{file: file, clients: map[chan struct{}]bool{}, state: []byte(defaultState), pwFile: pwFile, anySetup: *remoteSetup}
	if b, err := os.ReadFile(pwFile); err == nil {
		if json.Unmarshal(b, &s.pw) != nil {
			log.Fatalf("%s is damaged: delete it or run with -reset-password", pwFile)
		}
	}
	var compact bytes.Buffer
	if b, err := os.ReadFile(file); err == nil && json.Compact(&compact, b) == nil {
		s.state = compact.Bytes()
	}

	fmt.Printf("Control panel:    http://localhost:%s/\n", port)
	fmt.Printf("OBS (in-game):    http://localhost:%s/overlay.html\n", port)
	fmt.Printf("OBS (break):      http://localhost:%s/ranking.html\n", port)
	fmt.Printf("OBS (commentary): http://localhost:%s/cams.html\n", port)
	fmt.Printf("OBS (game+chat):  http://localhost:%s/chat1.html (1 camera), /chat2.html (2 cameras)\n", port)
	err := http.ListenAndServe(host+":"+port, s.handler())
	if errors.Is(err, syscall.EADDRINUSE) {
		log.Fatalf("Port %s is already in use by another program. Try: PORT=3000 go run .", port)
	}
	log.Fatal(err)
}
