package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var failRate = "0"

type contextKey string

const failKey contextKey = "isFailed"

type VersionStatus string

const (
	VersionBlue  VersionStatus = "blue"
	VersionGreen VersionStatus = "green"
)

type APIResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type Middleware = func(http.Handler) http.Handler

func createStack(xs ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(xs) - 1; i >= 0; i-- {
			x := xs[i]
			next = x(next)
		}
		return next
	}
}

type SampleMiddleware struct{}

func NewSampleMiddleware() *SampleMiddleware {
	return &SampleMiddleware{}
}

func (s *SampleMiddleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isFail := shouldFail(failRate)
		code := http.StatusOK
		color := "\033[32m"
		ctx := context.WithValue(r.Context(), failKey, isFail)

		if isFail {
			code = http.StatusInternalServerError
			color = "\033[31m"
		}
		current := time.Now()
		next.ServeHTTP(w, r.WithContext(ctx))
		log.Printf("%s%s %s, Latency=%d,staus=%d\033[0m\n", color, r.Method, r.URL.Path, time.Since(current), code)
	})
}

func shouldFail(rate string) bool {
	// 隨機故障
	r := 0
	fmt.Sscanf(rate, "%d", &r)
	return rand.Intn(100) < r
}

func main() {
	version := os.Getenv("APP_VERSION") // "blue" or "green"
	if version == "" {
		version = string(VersionBlue)
	}
	if version != string(VersionBlue) && version != string(VersionGreen) {
		panic("version not allowed")
	}
	failRate = os.Getenv("FAIL_RATE") // 0-100
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	middleware := NewSampleMiddleware()
	stack := createStack(middleware.Logging)

	router := http.NewServeMux()
	router.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		if err := encoder.Encode(APIResponse{Status: "ok", Version: version}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		isFail, _ := r.Context().Value(failKey).(bool)

		if version == string(VersionGreen) {
			// green 版本搞一些劣化
			time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
		}

		if isFail {
			w.WriteHeader(http.StatusInternalServerError)
			if err := encoder.Encode(APIResponse{Status: "error", Version: version}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		if err := encoder.Encode(APIResponse{Status: "ok", Version: version}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: stack(router),
	}

	fmt.Printf("server start...")
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server : %v\n", err)
			return
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("failed to shutdown server : %v\n", err)
	}
}
