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

	baseURL := mustParseURL(getEnv("BASE_URL", "http://localhost:9090/"))

	webProxy := createProxy(baseURL, mustParseURL(getEnv("TARGET_WEB", "http://127.0.0.1:8081/")))
	apiProxy := createProxy(baseURL, mustParseURL(getEnv("TARGET_API", "http://127.0.0.1:8082/")))
	authProxy := createProxy(baseURL, mustParseURL(getEnv("TARGET_AUTH", "http://127.0.0.1:8083/")))

	mux.Handle("/", webProxy)

	mux.Handle("/api", apiProxy)
	mux.Handle("/api/", apiProxy)

	mux.Handle("/api/v1/me", authProxy)
	mux.Handle("/api/v1/login", authProxy)
	mux.Handle("/api/v1/csrftoken/", authProxy)

	mux.Handle("/metrics", apiProxy)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("request", "method", r.Method, "url", r.URL.String(), "remote", r.RemoteAddr, "user-agent", r.UserAgent())

		if token := getToken(); token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}

		mux.ServeHTTP(w, r)
	})

	http.ListenAndServe(":9090", h)
}

func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)

	if err != nil {
		panic(err)
	}

	return u
}

func createProxy(baseURL *url.URL, targetURL *url.URL) http.Handler {
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(targetURL)
			r.Out.Host = r.In.Host

			if baseURL.Host != "" {
				r.Out.Host = baseURL.Host
			}
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
