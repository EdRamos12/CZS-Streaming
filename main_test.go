package main

import (
	"bufio"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	dir := t.TempDir()
	s := &server{file: filepath.Join(dir, "state.json"), pwFile: filepath.Join(dir, "password.json"), clients: map[chan struct{}]bool{}, state: []byte(defaultState)}
	ts := httptest.NewServer(s.handler())
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	http.DefaultClient.Jar = jar
	defer func() { http.DefaultClient.Jar = nil }()

	ev, err := http.Get(ts.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer ev.Body.Close()
	lines := bufio.NewScanner(ev.Body)
	next := func() string {
		for lines.Scan() {
			if l := lines.Text(); l != "" {
				return l
			}
		}
		return ""
	}
	if l := next(); l != "data: "+defaultState {
		t.Fatalf("initial event: %q", l)
	}

	post := func(p, body string) int {
		r, err := http.Post(ts.URL+p, "", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		return r.StatusCode
	}
	// password: locked until set, setup only from this computer, then login works and wrong passwords fail
	if c := post("/api/state", defaultState); c != 401 {
		t.Fatalf("state before login: %d", c)
	}
	remote := httptest.NewRequest("POST", "/api/login", strings.NewReader("secret1"))
	remote.RemoteAddr = "192.0.2.1:1234"
	rec := httptest.NewRecorder()
	if s.handler().ServeHTTP(rec, remote); rec.Code != 403 {
		t.Fatalf("setup from LAN: %d", rec.Code)
	}
	// with -remote-setup the LAN may create it (checked on a throwaway server so the rest of the test is unchanged)
	any := &server{pwFile: filepath.Join(dir, "any.json"), anySetup: true}
	remote = httptest.NewRequest("POST", "/api/login", strings.NewReader("secret1"))
	remote.RemoteAddr = "192.0.2.1:1234"
	rec = httptest.NewRecorder()
	if any.login(rec, remote); rec.Code != 200 {
		t.Fatalf("remote setup with -remote-setup: %d", rec.Code)
	}
	if c := post("/api/login", "short"); c != 400 {
		t.Fatalf("short password: %d", c)
	}
	if c := post("/api/login", "secret1"); c != 200 {
		t.Fatalf("setup: %d", c)
	}
	jar, _ = cookiejar.New(nil)
	http.DefaultClient.Jar = jar
	if c := post("/api/login", "wrong!!"); c != 403 {
		t.Fatalf("wrong password: %d", c)
	}
	if c := post("/api/state", defaultState); c != 401 {
		t.Fatalf("state after wrong password: %d", c)
	}
	if c := post("/api/login", "secret1"); c != 200 {
		t.Fatalf("login: %d", c)
	}
	// lockout: 5 wrong tries from one address block it, even with the right password; other addresses still work
	try := func(ip, pw string) int {
		req := httptest.NewRequest("POST", "/api/login", strings.NewReader(pw))
		req.RemoteAddr = ip + ":1234"
		rec := httptest.NewRecorder()
		s.handler().ServeHTTP(rec, req)
		return rec.Code
	}
	for i := 0; i < maxFails; i++ {
		if c := try("192.0.2.9", "wrong!!"); c != 403 {
			t.Fatalf("wrong try %d: %d", i, c)
		}
	}
	if c := try("192.0.2.9", "secret1"); c != 429 {
		t.Fatalf("locked address: %d", c)
	}
	if c := try("192.0.2.8", "secret1"); c != 200 {
		t.Fatalf("other address: %d", c)
	}
	s.fails["192.0.2.9"] = failures{until: time.Now().Add(-time.Second)} // lock expired
	if c := try("192.0.2.9", "secret1"); c != 200 {
		t.Fatalf("after lock expired: %d", c)
	}
	if _, ok := s.fails["192.0.2.9"]; ok {
		t.Fatal("success should clear the counter")
	}
	if b, _ := os.ReadFile(s.pwFile); strings.Contains(string(b), "secret1") {
		t.Fatal("password stored in plain text")
	}
	anon := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for _, p := range []string{"/", "/control.html", "/media.html"} {
		r, _ := anon.Get(ts.URL + p)
		if r.Body.Close(); r.StatusCode != 303 || r.Header.Get("Location") != "/login.html" {
			t.Errorf("GET %s logged out: %d %s", p, r.StatusCode, r.Header.Get("Location"))
		}
	}

	for _, bad := range []string{`nope`, `null`, `[]`, `{"round":{},"seats":[1,2,3],"players":[]}`, `{"round":null,"seats":[1,2,3,4],"players":[]}`, `{"round":{},"seats":[1,2,3,4]}`} {
		if c := post("/api/state", bad); c != 400 {
			t.Errorf("%s: got %d, want 400", bad, c)
		}
	}
	good := `{"round": {"num": 1}, "seats": [1,2,3,4], "players": [], "extra": "kept"}`
	if c := post("/api/state", good); c != 200 {
		t.Fatalf("valid state: %d", c)
	}
	if l := next(); l != `data: {"round":{"num":1},"seats":[1,2,3,4],"players":[],"extra":"kept"}` {
		t.Fatalf("broadcast: %q", l)
	}
	if b, _ := os.ReadFile(s.file); !strings.Contains(string(b), "\n  \"extra\": \"kept\"") {
		t.Fatalf("saved file: %s", b)
	}

	if c := post("/api/upload?name=x.svg", "<svg/>"); c != 400 {
		t.Errorf("svg upload: %d", c)
	}
	for p, want := range map[string]int{"/": 200, "/media.html": 200, "/overlay.html": 200, "/cams1.html": 200, "/cams2.html": 200, "/chat1.html": 200, "/chat2.html": 200, "/info.html": 200, "/login.html": 200, "/nope.html": 404, "/../state.json": 404} {
		r, _ := http.Get(ts.URL + p)
		if r.Body.Close(); r.StatusCode != want {
			t.Errorf("GET %s: %d, want %d", p, r.StatusCode, want)
		}
	}
}

func TestChangelog(t *testing.T) {
	rs := parseLog("# Changelog\n- not a note\n## 1.2.0\n- pages\n## 1.1.0 (server)\n- api\n- more\n## 1.0.0\n")
	if len(rs) != 3 || rs[0].Version != "1.2.0" || rs[0].Server || !rs[1].Server || len(rs[1].Notes) != 2 {
		t.Fatalf("parse: %+v", rs)
	}
	if n := newer(rs, "1.1.0"); len(n) != 1 || anyServer(n) {
		t.Errorf("from 1.1.0: %+v", n) // pages only: pulled on its own
	}
	if n := newer(rs, "1.0.0"); len(n) != 2 || !anyServer(n) {
		t.Errorf("from 1.0.0: %+v", n) // includes a server release: pulled by hand
	}
	if n := newer(rs, "9.9.9"); n != nil {
		t.Errorf("unknown version: %+v", n)
	}
	if top(parseLog(builtLog)) == "?" {
		t.Error("CHANGELOG.md has no version heading")
	}
}
