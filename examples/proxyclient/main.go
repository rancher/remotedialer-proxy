package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/rancher/remotedialer-proxy/forward"
	"github.com/rancher/remotedialer-proxy/proxyclient"
	"github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
)

var (
	namespace             = "cattle-system"
	label                 = "app=api-extension"
	certSecretName        = "api-extension-ca-name"
	certServerName        = "api-extension-tls-name"
	connectSecret         = "api-extension"
	connectUrl            = ""
	ports                 = []string{"8443:8443"}
	fakeImperativeAPIAddr = "0.0.0.0:6666"
)

func init() {
	if val, ok := os.LookupEnv("NAMESPACE"); ok {
		namespace = val
	}
	if val, ok := os.LookupEnv("LABEL"); ok {
		label = val
	}
	if val, ok := os.LookupEnv("CERT_SECRET_NAME"); ok {
		certSecretName = val
	}
	if val, ok := os.LookupEnv("CERT_SERVER_NAME"); ok {
		certServerName = val
	}
	if val, ok := os.LookupEnv("CONNECT_SECRET"); ok {
		connectSecret = val
	}
	if val, ok := os.LookupEnv("CONNECT_URL"); ok {
		connectUrl = val
	}
	if val, ok := os.LookupEnv("PORTS"); ok {
		ports = strings.Split(val, ",")
	}
	if val, ok := os.LookupEnv("FAKE_IMPERATIVE_API_ADDR"); ok {
		fakeImperativeAPIAddr = val
	}
}

func handleConnection(ctx context.Context, connFromRDProxy net.Conn) {
	defer connFromRDProxy.Close()

	// Dial the echo-server
	echoServerAddr := fmt.Sprintf("echo-server.%s.svc:12345", namespace)
	connToEchoServer, err := net.Dial("tcp", echoServerAddr)
	if err != nil {
		logrus.Errorf("handleConnection: error dialing echo-server %s: %v", echoServerAddr, err)
		return
	}
	defer connToEchoServer.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	// Pipe data bidirectionally
	go func() {
		defer wg.Done()
		// Copy from RDProxy to Echo Server
		io.Copy(connToEchoServer, connFromRDProxy)
	}()

	go func() {
		defer wg.Done()
		// Copy from Echo Server to RDProxy
		io.Copy(connFromRDProxy, connToEchoServer)
	}()

	// Wait for both copy operations to complete
	wg.Wait()
	logrus.Info("handleConnection: finished piping data")
}

func fakeImperativeAPI(ctx context.Context) error {
	ln, err := net.Listen("tcp", fakeImperativeAPIAddr)
	if err != nil {
		return fmt.Errorf("Error starting server on %s: %w", fakeImperativeAPIAddr, err)
	}
	logrus.Infof("Server listening on %s...", fakeImperativeAPIAddr)

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, acceptErr := ln.Accept()
		if acceptErr != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return fmt.Errorf("fakeImperativeAPI: error accepting connection: %w", acceptErr)
			}
		}
		go handleConnection(ctx, conn)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := fakeImperativeAPI(ctx); err != nil {
			logrus.Errorf("fakeImperativeAPI error: %v", err)
			cancel()
		}
	}()

	cfg, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	options := &core.FactoryOptions{
		Namespace: namespace,
	}

	coreFactory, err := core.NewFactoryFromConfigWithOptions(cfg, options)
	if err != nil {
		logrus.Fatal(err)
	}

	podClient := coreFactory.Core().V1().Pod()
	secretContoller := coreFactory.Core().V1().Secret()

	portForwarder, err := forward.New(cfg, podClient, namespace, label, ports)
	if err != nil {
		logrus.Fatal(err)
	}

	opts := []proxyclient.ProxyClientOpt{}
	if connectUrl != "" {
		opts = append(opts, proxyclient.WithServerURL(connectUrl))
	}

	proxyClient, err := proxyclient.New(
		ctx,
		connectSecret,
		namespace,
		certSecretName,
		certServerName,
		secretContoller,
		portForwarder,
		opts...,
	)
	if err != nil {
		logrus.Fatal(err)
	}

	if err := coreFactory.Start(ctx, 1); err != nil {
		logrus.Fatal(err)
	}

	proxyClient.Run(ctx)

	logrus.Info("RDP Client Started... Waiting for CTRL+C")
	<-sigChan
	logrus.Info("Stopping...")

	cancel()
	proxyClient.Stop()
}
