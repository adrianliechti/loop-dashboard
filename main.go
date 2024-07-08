package main

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

func main() {
	mux := http.NewServeMux()

	webProxy := createProxy(getEnv("TARGET_WEB", "http://localhost:8081"))
	apiProxy := createProxy(getEnv("TARGET_API", "http://localhost:8082"))
	authProxy := createProxy(getEnv("TARGET_AUTH", "http://localhost:8083"))

	mux.Handle("/*", webProxy)
	mux.Handle("/api/*", apiProxy)
	mux.Handle("/api/v1/me", authProxy)
	mux.Handle("/api/v1/login", authProxy)
	mux.Handle("/api/v1/csrftoken/*", authProxy)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("request", "method", r.Method, "url", r.URL.String(), "remote", r.RemoteAddr, "user-agent", r.UserAgent())

		if token := getToken(); token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}

		mux.ServeHTTP(w, r)
	})

	http.ListenAndServe(":8080", h)
}

func createProxy(target string) http.Handler {
	targetURL, _ := url.Parse(target)

	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(targetURL)
			r.Out.Host = r.In.Host
		},
	}
}

func getToken() string {
	var token string

	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		token = string(data)
	}

	if val := os.Getenv("TOKEN"); val != "" {
		token = val
	}

	return token
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return fallback
}
