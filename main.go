package main

import (
	"container/heap"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	Attempts int = iota
	Retry
)

var serverPool ServerPool
var method string

// Backend holds the data about a server
type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
	Weight       uint64
	Stats        Stats
}

// ServerPool holds information about reachable servers
type ServerPool struct {
	backends BackendHeap
	current  uint64
	weight   uint64
}

// BackendHeap is a type alias for []*Backend with heap.Interface methods
type BackendHeap []*Backend

func (h BackendHeap) Len() int           { return len(h) }
func (h BackendHeap) Less(i, j int) bool { return h[i].Alive && !h[j].Alive } // Alive backends first
func (h BackendHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *BackendHeap) Push(x interface{}) {
	*h = append(*h, x.(*Backend))
}

func (h *BackendHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func (s *ServerPool) AddBackend(backend *Backend) {
	heap.Push(&s.backends, backend)
}

func (s *ServerPool) MarkBackendStatus(backendUrl *url.URL, alive bool) {
	for i, b := range s.backends {
		if b.URL.String() == backendUrl.String() {
			b.SetAlive(alive)
			heap.Fix(&s.backends, i) // Fix the heap after updating the status
			break
		}
	}
}

func GetAttemptsFromContext(r *http.Request) int {
	if attempts, ok := r.Context().Value(Attempts).(int); ok {
		return attempts
	}
	return 1
}

func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(Retry).(int); ok {
		return retry
	}
	return 0
}

func isBackendAlive(u *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		log.Println("Site unreachable, error: ", err)
		return false
	}
	defer conn.Close()
	return true
}

// lb load balances the incoming request
func lb(w http.ResponseWriter, r *http.Request) {
	attempts := GetAttemptsFromContext(r)
	if attempts > 3 {
		log.Printf("%s(%s) Max attempts reached, exiting\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}
	var peer *Backend
	if method == "wrr" {
		peer = serverPool.GetNextWRRPeer()
	} else {
		peer = serverPool.GetNextRRPeer()
	}

	if peer != nil {
		// Increment total requests count
		peer.IncrementRequestCount()

		// Measure request latency
		start := time.Now()
		peer.ReverseProxy.ServeHTTP(w, r)
		latency := time.Since(start)
		peer.AddLatency(latency)

		// Check the response status code
		if w.Header().Get("Status") != "200 OK" {
			// Increment error count
			peer.IncrementErrorCount()
		}
		return
	}

	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

func main() {
	var serverList string
	var port int

	flag.StringVar(&method, "method", "rr", "Select load balancing method (rr|wrr)")
	flag.StringVar(&serverList, "servers", "", "Load balanced servers")
	flag.IntVar(&port, "port", 3030, "Port")
	flag.Parse()

	if len(serverList) == 0 {
		log.Fatal("No servers provided")
	}

	if method != "rr" && method != "wrr" {
		log.Fatal("Method should be either rr or wrr")
	}

	// parse servers
	tokens := strings.Split(serverList, ",")
	for _, tok := range tokens {
		serverUrl, err := url.Parse(tok)
		if err != nil {
			log.Fatal(err)
		}

		proxy := httputil.NewSingleHostReverseProxy(serverUrl)
		proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error) {
			log.Printf("[%s] %s\n", serverUrl.Host, e.Error())
			retries := GetRetryFromContext(request)
			if retries < 3 {
				select {
				case <-time.After(10 * time.Millisecond):
					ctx := context.WithValue(request.Context(), Retry, retries+1)
					proxy.ServeHTTP(writer, request.WithContext(ctx))
				}
				return
			}

			serverPool.MarkBackendStatus(serverUrl, false)

			attempts := GetAttemptsFromContext(request)
			log.Printf("%s(%s) Attempting retry %d\n", request.RemoteAddr, request.URL.Path, attempts)
			ctx := context.WithValue(request.Context(), Attempts, attempts+1)
			lb(writer, request.WithContext(ctx))
		}

		serverPool.AddBackend(&Backend{
			URL:          serverUrl,
			Alive:        true,
			ReverseProxy: proxy,
		})
		log.Printf("Configured server: %s\n", serverUrl)
	}

	go WriteStatsToFile()

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(lb),
	}

	go RunHealthCheck()

	log.Printf("Load Balancer started at :%d\n", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
