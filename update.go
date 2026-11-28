// Updates from GitHub through git: page-only releases are pulled on their own, server releases by hand.
// Versions come from CHANGELOG.md: the copy built into the binary (server), the one on disk (pages)
// and the one on the upstream branch (latest).
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

//go:embed CHANGELOG.md
var builtLog string

type release struct {
	Version string   `json:"version"`
	Server  bool     `json:"server"` // needs a server rebuild/restart, not only new pages
	Notes   []string `json:"notes"`
}

var heading = regexp.MustCompile(`^## (\S+)( \(server\))?`)

func parseLog(s string) (rs []release) {
	for _, l := range strings.Split(s, "\n") {
		if m := heading.FindStringSubmatch(l); m != nil {
			rs = append(rs, release{Version: m[1], Server: m[2] != ""})
		} else if n, ok := strings.CutPrefix(l, "- "); ok && len(rs) > 0 {
			rs[len(rs)-1].Notes = append(rs[len(rs)-1].Notes, n)
		}
	}
	return
}

// newer: releases listed above v. An unknown v (e.g. a local version not pushed yet) gives none.
func newer(rs []release, v string) []release {
	for i, r := range rs {
		if r.Version == v {
			return rs[:i]
		}
	}
	return nil
}

func anyServer(rs []release) bool {
	for _, r := range rs {
		if r.Server {
			return true
		}
	}
	return false
}

func top(rs []release) string {
	if len(rs) == 0 {
		return "?"
	}
	return rs[0].Version
}

type updateInfo struct {
	Server      string    `json:"server"`      // version this binary was built from
	Pages       string    `json:"pages"`       // version of the files on disk
	Restart     bool      `json:"restart"`     // files on disk need a newer server: rebuild/restart
	New         []release `json:"new"`         // on GitHub, not pulled yet
	NeedsServer bool      `json:"needsServer"` // one of New changes the server, so it is not pulled on its own
	Updated     bool      `json:"updated"`     // pages were just pulled
	Err         string    `json:"err,omitempty"`
}

var (
	updMu   sync.Mutex
	updLast = updateInfo{Server: top(parseLog(builtLog))}
)

func git(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", args...).CombinedOutput()
	if err != nil && len(out) > 0 {
		return "", errors.New(strings.TrimSpace(string(out)))
	}
	return string(out), err
}

// checkUpdate fetches upstream and, with apply, pulls it when it only holds page releases.
func checkUpdate(apply bool) updateInfo {
	updMu.Lock()
	defer updMu.Unlock()
	built := parseLog(builtLog)
	info := updateInfo{Server: top(built)}
	status := func() {
		disk, _ := os.ReadFile("CHANGELOG.md")
		d := parseLog(string(disk))
		info.Pages, info.Restart = top(d), anyServer(newer(d, info.Server))
	}
	status()
	defer func() { updLast = info }()
	if _, err := git("fetch", "--quiet"); err != nil {
		info.Err = "git fetch: " + err.Error()
		return info
	}
	remote, err := git("show", "@{u}:CHANGELOG.md")
	if err != nil {
		info.Err = "git show: " + err.Error()
		return info
	}
	info.New = newer(parseLog(remote), info.Pages)
	info.NeedsServer = anyServer(info.New)
	if !apply || len(info.New) == 0 || info.NeedsServer {
		return info
	}
	if _, err := git("pull", "--ff-only", "--quiet"); err != nil {
		info.Err = "git pull: " + err.Error() // local edits or commits: nothing is overwritten
		return info
	}
	log.Println("pages updated to", info.New[0].Version)
	info.New, info.Updated = nil, true
	status()
	return info
}

// GET /api/version: last check. POST /api/update: check again and pull pages if possible.
func versionInfo(w http.ResponseWriter, r *http.Request) {
	updMu.Lock()
	info := updLast
	updMu.Unlock()
	json.NewEncoder(w).Encode(info)
}

func runUpdate(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(checkUpdate(true))
}
