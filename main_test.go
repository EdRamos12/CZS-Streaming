package main

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServer(t *testing.T) {
	s := &server{file: filepath.Join(t.TempDir(), "state.json"), clients: map[chan struct{}]bool{}, state: []byte(defaultState)}
	ts := httptest.NewServer(s.handler())
	defer ts.Close()

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
	for p, want := range map[string]int{"/": 200, "/overlay.html": 200, "/nope.html": 404, "/../state.json": 404} {
		r, _ := http.Get(ts.URL + p)
		if r.Body.Close(); r.StatusCode != want {
			t.Errorf("GET %s: %d, want %d", p, r.StatusCode, want)
		}
	}
}
