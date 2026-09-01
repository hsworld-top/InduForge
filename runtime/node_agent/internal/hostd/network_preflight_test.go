package hostd

import (
	"context"
	"net"
	"testing"
)

func TestCheckLocalPortReportsOccupiedTCPPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	if err := checkLocalPort(context.Background(), "tcp", "127.0.0.1", port, "test"); err == nil {
		t.Fatal("occupied port must fail preflight")
	}
}
