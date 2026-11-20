package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	grpcInfopage "github.com/patrickbrandao/traefix/infopage/grpc"
	"google.golang.org/grpc"
)

//go:embed openapi.yaml
var openAPISpec []byte

//go:embed static
var staticFS embed.FS

// Units.
const (
	_        = iota
	KB int64 = 1 << (10 * iota)
	MB
	GB
	TB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

var (
	cert    string
	key     string
	ca      string
	port    string
	name    string
	prefix  string
	verbose bool

	withLists bool
)

func init() {
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.StringVar(&cert, "cert", "", "give me a certificate")
	flag.StringVar(&key, "key", "", "give me a key")
	flag.StringVar(&ca, "cacert", "", "give me a CA chain, enforces mutual TLS")
	flag.StringVar(&port, "port", getEnv("INFOPAGE_PORT_NUMBER", "80"), "give me a port number")
	flag.StringVar(&name, "name", os.Getenv("INFOPAGE_NAME"), "give me a name")
	flag.StringVar(&prefix, "prefix", getEnv("INFOPAGE_HEADER_PREFIX", "X-Traefix-"), "prefix of the headers added by Traefix")
	flag.BoolVar(&withLists, "lists", getEnv("INFOPAGE_LISTS", "1") != "0", "download the address lists of robots, clouds and iCloud Private Relay")
}

// Data infopage information.
type Data struct {
	Hostname   string            `json:"hostname,omitempty"`
	IP         []string          `json:"ip,omitempty"`
	Headers    http.Header       `json:"headers,omitempty"`
	URL        string            `json:"url,omitempty"`
	Host       string            `json:"host,omitempty"`
	Method     string            `json:"method,omitempty"`
	Name       string            `json:"name,omitempty"`
	RemoteAddr string            `json:"remoteAddr,omitempty"`
	Environ    map[string]string `json:"environ,omitempty"`
	Proto      string            `json:"proto,omitempty"`
	Prefix     string            `json:"prefix,omitempty"`
	Time       string            `json:"time,omitempty"`
	Score      *Score            `json:"score,omitempty"`
}

func main() {
	flag.Parse()

	// The container has 64 MB: keep the heap well under it while the lists are refreshed.
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(40 << 20)
	}

	if withLists {
		lists = newListStore()
		go lists.run(context.Background())
	}

	mux := http.NewServeMux()
	mux.Handle("/data", handle(dataHandler, verbose))
	mux.Handle("/echo", handle(echoHandler, verbose))
	mux.Handle("/bench", handle(benchHandler, verbose))
	mux.Handle("/api", handle(apiHandler, verbose))
	mux.Handle("/health", handle(healthHandler, verbose))
	mux.Handle("/openapi.yaml", handle(openAPIHandler, verbose))
	mux.Handle("/assets/", handle(assetsHandler(), verbose))
	mux.Handle("/", handle(infopageHandler, verbose))

	serverGRPC := grpc.NewServer()
	grpcInfopage.RegisterInfopageServer(serverGRPC, infopageServer{})
	mux.Handle("/infopage.Infopage/", serverGRPC)

	h := handle(mux.ServeHTTP, verbose)

	if cert == "" || key == "" {
		log.Printf("Starting up on port %s", port)

		var protocols http.Protocols
		protocols.SetHTTP1(true)
		protocols.SetUnencryptedHTTP2(true)

		server := &http.Server{
			Addr:      ":" + port,
			Handler:   h,
			Protocols: &protocols,
		}

		log.Fatal(server.ListenAndServe())
	}

	server := &http.Server{
		Addr:      ":" + port,
		TLSConfig: &tls.Config{ClientAuth: tls.RequestClientCert},
		Handler:   h,
	}

	if ca != "" {
		server.TLSConfig = setupMutualTLS(ca)
	}

	log.Printf("Starting up with TLS on port %s", port)

	log.Fatal(server.ListenAndServeTLS(cert, key))
}

func setupMutualTLS(ca string) *tls.Config {
	clientCACert, err := os.ReadFile(ca)
	if err != nil {
		log.Fatal(err)
	}

	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(clientCACert)

	tlsConfig := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  clientCertPool,
		MinVersion: tls.VersionTLS12,
	}

	return tlsConfig
}

func handle(next http.HandlerFunc, verbose bool) http.Handler {
	if !verbose {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next(w, r)

		// <remote_IP_address> - [<timestamp>] "<request_method> <request_path> <request_protocol>" -
		log.Printf("%s - - [%s] \"%s %s %s\" - -", r.RemoteAddr, time.Now().Format("02/Jan/2006:15:04:05 -0700"), r.Method, r.URL.Path, r.Proto)
	})
}

func benchHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprint(w, "1")
}

func openAPIHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(openAPISpec)
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			return
		}

		printBinary(p)
		err = conn.WriteMessage(messageType, p)
		if err != nil {
			return
		}
	}
}

func printBinary(s []byte) {
	fmt.Printf("Received b:")
	for n := range s {
		fmt.Printf("%d,", s[n])
	}
	fmt.Printf("\n")
}

func dataHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	size, err := strconv.ParseInt(queryParams.Get("size"), 10, 64)
	if err != nil {
		size = 1
	}
	if size < 0 {
		size = 0
	}

	unit := queryParams.Get("unit")
	switch strings.ToLower(unit) {
	case "kb":
		size *= KB
	case "mb":
		size *= MB
	case "gb":
		size *= GB
	case "tb":
		size *= TB
	}

	attachment, err := strconv.ParseBool(queryParams.Get("attachment"))
	if err != nil {
		attachment = false
	}

	content := &contentReader{size: size}

	if attachment {
		w.Header().Set("Content-Disposition", "Attachment")
		http.ServeContent(w, r, "data.txt", time.Now(), content)
		return
	}

	if _, err := io.Copy(w, content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func infopageHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	wait := queryParams.Get("wait")
	if wait != "" {
		duration, err := time.ParseDuration(wait)
		if err == nil {
			time.Sleep(duration)
		}
	}

	w.Header().Add("Vary", "Accept")

	if wantsHTML(r) {
		pageHandler(w, r)
		return
	}

	if name != "" {
		_, _ = fmt.Fprintln(w, "Name:", name)
	}

	hostname, _ := os.Hostname()
	_, _ = fmt.Fprintln(w, "Hostname:", hostname)

	for _, ip := range getIPs() {
		_, _ = fmt.Fprintln(w, "IP:", ip)
	}

	_, _ = fmt.Fprintln(w, "RemoteAddr:", r.RemoteAddr)
	writeScoreText(w, computeScore(r.Context(), r, prefix, false, lists))

	if r.TLS != nil {
		for i, cert := range r.TLS.PeerCertificates {
			_, _ = fmt.Fprintf(w, "Certificate[%d] Subject: %v\n", i, cert.Subject)
		}
	}

	if err := r.Write(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if ok, _ := strconv.ParseBool(queryParams.Get("env")); ok {
		for _, env := range os.Environ() {
			_, _ = fmt.Fprintln(w, env)
		}
	}
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(newData(r, false)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func newData(r *http.Request, nav bool) Data {
	queryParams := r.URL.Query()

	hostname, _ := os.Hostname()

	environ := make(map[string]string)

	if ok, _ := strconv.ParseBool(queryParams.Get("env")); ok {
		for _, env := range os.Environ() {
			before, after, _ := strings.Cut(env, "=")
			environ[before] = after
		}
	}

	return Data{
		Hostname:   hostname,
		IP:         getIPs(),
		Headers:    r.Header,
		URL:        r.URL.RequestURI(),
		Host:       r.Host,
		Method:     r.Method,
		Name:       name,
		RemoteAddr: r.RemoteAddr,
		Environ:    environ,
		Proto:      r.Proto,
		Prefix:     prefix,
		Time:       time.Now().UTC().Format(time.RFC3339Nano),
		Score:      computeScore(r.Context(), r, prefix, nav, lists),
	}
}

type healthState struct {
	StatusCode int
}

var (
	currentHealthState = healthState{http.StatusOK}
	mutexHealthState   = &sync.RWMutex{}
)

func healthHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPost {
		var statusCode int

		if err := json.NewDecoder(req.Body).Decode(&statusCode); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		fmt.Printf("Update health check status code [%d]\n", statusCode)

		mutexHealthState.Lock()
		defer mutexHealthState.Unlock()
		currentHealthState.StatusCode = statusCode
	} else {
		mutexHealthState.RLock()
		defer mutexHealthState.RUnlock()
		w.WriteHeader(currentHealthState.StatusCode)
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getIPs() []string {
	var ips []string

	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil {
				ips = append(ips, ip.String())
			}
		}
	}

	return ips
}

type infopageServer struct {
	grpcInfopage.UnimplementedInfopageServer
}

func (g infopageServer) Bench(_ context.Context, _ *grpcInfopage.BenchRequest) (*grpcInfopage.BenchReply, error) {
	return &grpcInfopage.BenchReply{Data: 1}, nil
}

func (g infopageServer) Infopage(_ context.Context, _ *grpcInfopage.InfopageRequest) (*grpcInfopage.InfopageReply, error) {
	reply := &grpcInfopage.InfopageReply{}
	if name != "" {
		reply.Name = name
	}

	reply.Hostname, _ = os.Hostname()

	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			reply.Iface = append(reply.Iface, ip.String())
		}
	}

	return reply, nil
}
