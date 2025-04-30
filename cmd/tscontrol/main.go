package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"strconv"
	"testing"

	"tailscale.com/tstest/integration"
	"tailscale.com/tstest/integration/testcontrol"
	"tailscale.com/types/logger"
)

var (
	flagAddress = flag.String("host", "0.0.0.0", "network address to bind to")
	flagPort    = flag.Int("port", 9911, "api port to listen on")
)

func main() {
	var t fakeTB
	derpMap := integration.RunDERPAndSTUN(t, logger.Discard, *flagAddress)

	addr := net.JoinHostPort(*flagAddress, strconv.Itoa(*flagPort))
	control := &testcontrol.Server{
		DERPMap:         derpMap,
		ExplicitBaseURL: "http://" + addr,
	}

	mux := http.NewServeMux()
	mux.Handle("GET /health", http.HandlerFunc(health))
	mux.Handle("/", control)
	log.Printf("listening on %s", addr)

	err := http.ListenAndServe(addr, mux)
	log.Fatal(err)
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

type fakeTB struct {
	*testing.T
}

func (t fakeTB) Cleanup(_ func()) {}
func (t fakeTB) Error(args ...any) {
	t.Fatal(args...)
}
func (t fakeTB) Errorf(format string, args ...any) {
	t.Fatalf(format, args...)
}
func (t fakeTB) Fail() {
	t.Fatal("failed")
}
func (t fakeTB) FailNow() {
	t.Fatal("failed")
}
func (t fakeTB) Failed() bool {
	return false
}
func (t fakeTB) Fatal(args ...any) {
	log.Fatal(args...)
}
func (t fakeTB) Fatalf(format string, args ...any) {
	log.Fatalf(format, args...)
}
func (t fakeTB) Helper() {}
func (t fakeTB) Log(args ...any) {
	log.Print(args...)
}
func (t fakeTB) Logf(format string, args ...any) {
	log.Printf(format, args...)
}
func (t fakeTB) Name() string {
	return "faketest"
}
func (t fakeTB) Setenv(key string, value string) {
	panic("not implemented")
}
func (t fakeTB) Skip(args ...any) {
	t.Fatal("skipped")
}
func (t fakeTB) SkipNow() {
	t.Fatal("skipnow")
}
func (t fakeTB) Skipf(format string, args ...any) {
	t.Logf(format, args...)
	t.Fatal("skipped")
}
func (t fakeTB) Skipped() bool {
	return false
}
func (t fakeTB) TempDir() string {
	panic("not implemented")
}
