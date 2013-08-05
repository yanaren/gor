package main

import (
	"testing"

	"github.com/buger/gor/listener"
	"github.com/buger/gor/replay"

	"time"

	"fmt"
	"net/http"
	"strconv"
)

func isEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Error("Original and Replayed request not match\n", a, "!=", b)
	}
}

var envs int

type Env struct {
	Verbose bool

	ListenHandler http.HandlerFunc
	ReplayHandler http.HandlerFunc
}

func (e *Env) start() (p int) {
	p = 50000 + envs*10

	go e.startHTTP(p, http.HandlerFunc(e.ListenHandler))
	go e.startHTTP(p+2, http.HandlerFunc(e.ReplayHandler))
	go e.startListener(p, p+1)
	go e.startReplay(p+1, p+2)

	// Time to start http and gor instances
	time.Sleep(time.Millisecond * 100)

	envs++

	return
}

func (e *Env) startListener(port int, replayPort int) {
	listener.Settings.Verbose = e.Verbose
	listener.Settings.Address = "127.0.0.1"
	listener.Settings.ReplayAddress = "127.0.0.1:" + strconv.Itoa(replayPort)
	listener.Settings.Port = port
	listener.Run()
}

func (e *Env) startReplay(port int, forwardPort int) {
	replay.Settings.Verbose = e.Verbose
	replay.Settings.Host = "127.0.0.1"
	replay.Settings.ForwardAddress = "127.0.0.1:" + strconv.Itoa(forwardPort)
	replay.Settings.Port = port
	replay.Run()
}

func (e *Env) startHTTP(port int, handler http.Handler) {
	http.ListenAndServe(":"+strconv.Itoa(port), handler)
}

func getRequest(port int) *http.Request {
	req, _ := http.NewRequest("GET", "http://localhost:"+strconv.Itoa(port)+"/test", nil)
	ck1 := new(http.Cookie)
	ck1.Name = "test"
	ck1.Value = "value"

	req.AddCookie(ck1)

	return req
}

func TestReplay(t *testing.T) {
	var request *http.Request
	received := make(chan int)

	listenHandler := func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "OK", http.StatusNotFound)
	}

	replayHandler := func(w http.ResponseWriter, r *http.Request) {
		isEqual(t, r.URL.Path, request.URL.Path)
		isEqual(t, r.Cookies()[0].Value, request.Cookies()[0].Value)

		http.Error(w, "404 page not found", http.StatusNotFound)

		if t.Failed() {
			fmt.Println("\nReplayed:", r, "\nOriginal:", request)
		}

		received <- 1
	}

	env := &Env{
		Verbose:       true,
		ListenHandler: listenHandler,
		ReplayHandler: replayHandler,
	}
	p := env.start()

	request = getRequest(p)

	_, err := http.DefaultClient.Do(request)

	if err != nil {
		t.Error("Can't make request", err)
	}

	select {
	case <-received:
	case <-time.After(time.Second):
		t.Error("Timeout error")
	}
}

func TestRateLimit(t *testing.T) {
	listenHandler := func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "OK", http.StatusAccepted)
	}

	replayHandler := func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "OK", http.StatusAccepted)
	}

	env := &Env{
		ListenHandler: listenHandler,
		ReplayHandler: replayHandler,
	}

	env.start()
}