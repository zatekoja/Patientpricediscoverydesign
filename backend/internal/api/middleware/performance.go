package middleware

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

// Compression middleware with gzip support
func Compression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client accepts gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Wrap response writer with gzip writer
		gz := gzipWriterPool.Get().(*gzip.Writer)
		defer gzipWriterPool.Put(gz)
		gz.Reset(w)
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length") // Let gzip set the correct length

		gzw := &gzipResponseWriter{
			ResponseWriter: w,
			Writer:         gz,
		}

		next.ServeHTTP(gzw, r)
	})
}

// Pool of gzip writers to reduce allocations
var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		// Compression level 5 is a good balance between speed and compression ratio
		gz, _ := gzip.NewWriterLevel(io.Discard, 5)
		return gz
	},
}

// gzipResponseWriter wraps http.ResponseWriter to compress the response
type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *gzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("ResponseWriter does not support Hijack")
}

// ETag middleware for conditional requests (304 Not Modified)
func ETag(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only apply to GET and HEAD requests
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}

		// Create a response recorder to capture the response
		rec := &etagResponseRecorder{
			ResponseWriter: w,
			buffer:         &bytes.Buffer{},
		}

		// Serve the request
		next.ServeHTTP(rec, r)

		// Only add ETag for successful responses
		if rec.statusCode == 0 || rec.statusCode == http.StatusOK {
			// Generate ETag from response body
			hash := sha256.Sum256(rec.buffer.Bytes())
			etag := `"` + hex.EncodeToString(hash[:16]) + `"` // Use first 16 bytes for shorter ETag

			// Check if client has matching ETag
			clientETag := r.Header.Get("If-None-Match")
			if clientETag == etag {
				// Client has the same version, return 304 Not Modified
				w.Header().Set("ETag", etag)
				w.WriteHeader(http.StatusNotModified)
				return
			}

			// Send the response with ETag
			w.Header().Set("ETag", etag)
			w.Header().Set("Cache-Control", "private, must-revalidate")
			if rec.statusCode > 0 {
				w.WriteHeader(rec.statusCode)
			}
			w.Write(rec.buffer.Bytes())
		} else {
			// Non-OK status, just write the buffered response
			if rec.statusCode > 0 {
				w.WriteHeader(rec.statusCode)
			}
			w.Write(rec.buffer.Bytes())
		}
	})
}

// etagResponseRecorder captures the response for ETag generation
type etagResponseRecorder struct {
	http.ResponseWriter
	buffer     *bytes.Buffer
	statusCode int
}

func (r *etagResponseRecorder) Write(b []byte) (int, error) {
	return r.buffer.Write(b)
}

func (r *etagResponseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

// CacheControl middleware adds cache headers based on path patterns
func CacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set cache headers based on endpoint
		path := r.URL.Path

		switch {
		case strings.HasPrefix(path, "/api/facilities/") && !strings.Contains(path, "search"):
			// Individual facilities: cache for 5 minutes
			w.Header().Set("Cache-Control", "public, max-age=300, must-revalidate")
		case strings.Contains(path, "/api/search") || strings.Contains(path, "/api/facilities/search"):
			// Search results: cache for 2 minutes
			w.Header().Set("Cache-Control", "public, max-age=120, must-revalidate")
		case strings.HasPrefix(path, "/api/procedures"):
			// Procedures: cache for 10 minutes (changes rarely)
			w.Header().Set("Cache-Control", "public, max-age=600, must-revalidate")
		case strings.HasPrefix(path, "/api/insurance"):
			// Insurance providers: cache for 15 minutes (changes rarely)
			w.Header().Set("Cache-Control", "public, max-age=900, must-revalidate")
		default:
			// Default: no cache for dynamic content
			w.Header().Set("Cache-Control", "private, no-cache, must-revalidate")
		}

		next.ServeHTTP(w, r)
	})
}

// LastModified middleware adds Last-Modified header
func LastModified(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only apply to GET and HEAD requests
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}

		// Create a response recorder to capture headers
		rec := &lastModifiedRecorder{
			ResponseWriter: w,
		}

		next.ServeHTTP(rec, r)

		// If the handler set a Last-Modified header, check If-Modified-Since
		if lastModified := rec.Header().Get("Last-Modified"); lastModified != "" {
			if ifModifiedSince := r.Header.Get("If-Modified-Since"); ifModifiedSince != "" {
				// Compare timestamps
				if ifModifiedSince == lastModified {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
		}
	})
}

type lastModifiedRecorder struct {
	http.ResponseWriter
}

// ResponseOptimization combines compression, ETag, and cache control
func ResponseOptimization(next http.Handler) http.Handler {
	// Chain middleware in order: CacheControl -> ETag -> Compression
	return CacheControl(ETag(Compression(next)))
}
