package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rancher/remotedialer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunProxyListener(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// remotedialer server
	authorizer := func(req *http.Request) (string, bool, error) {
		return "client-id", true, nil
	}
	remoteDialerServer := remotedialer.New(authorizer, remotedialer.DefaultErrorWriter)
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		remoteDialerServer.ServeHTTP(w, req)
	})
	wsServer := httptest.NewServer(handler)
	defer wsServer.Close()

	// peer server that the remotedialer client will connect to
	peerServer, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to start peer server")
	defer peerServer.Close()

	go func() {
		conn, err := peerServer.Accept()
		if err != nil {
			return // Listener closed
		}
		defer conn.Close()
		io.Copy(conn, conn)
	}()

	// remotedialer client
	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http") + "/connect"

	onConnect := func(sessionCtx context.Context, session *remotedialer.Session) error {
		return nil
	}

	// allow all
	connectAuthorizer := func(proto, address string) bool {
		return true
	}

	go func() {
		// ClientConnect will exit when the context is cancelled or the connection is otherwise lost.
		headers := http.Header{}
		headers.Set("X-API-Tunnel-Secret", "test-secret")
		err := remotedialer.ClientConnect(ctx, wsURL, headers, websocket.DefaultDialer, connectAuthorizer, onConnect)
		// No error on clean context cancellation
		if ctx.Err() == nil && err != nil {
			t.Errorf("remotedialer client connect error: %v", err)
		}
	}()

	// Wait for the client
	for i := 0; i < 100; i++ {
		if len(remoteDialerServer.ListClients()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	require.Greater(t, len(remoteDialerServer.ListClients()), 0, "remotedialer client did not connect in time")

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to find a free port")
	proxyPort := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	// Start the proxy listener
	cfg := &Config{
		ProxyPort: proxyPort,
		PeerPort:  peerServer.Addr().(*net.TCPAddr).Port,
	}

	go func() {
		_ = runProxyListener(ctx, cfg, remoteDialerServer)
	}()

	// Allow time for the listener to start
	time.Sleep(100 * time.Millisecond)

	// Connect to the proxy
	proxyAddr := fmt.Sprintf("127.0.0.1:%d", cfg.ProxyPort)
	proxyConn, err := net.Dial("tcp", proxyAddr)
	require.NoError(t, err, "failed to connect to proxy")
	defer proxyConn.Close()

	// Test data transfer
	const message = "hello proxy"
	_, err = proxyConn.Write([]byte(message))
	require.NoError(t, err, "failed to write to proxy")

	buf := make([]byte, len(message))
	proxyConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = proxyConn.Read(buf)
	require.NoError(t, err, "failed to read from proxy")

	assert.Equal(t, message, string(buf), "expected to read '%s', but got '%s'", message, string(buf))
}
