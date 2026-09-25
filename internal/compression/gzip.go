// Package compression provides response compression middleware
package compression

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// GzipWriter wraps http.ResponseWriter with gzip compression
type GzipWriter struct {
	http.ResponseWriter
	gzip *gzip.Writer
}

// Write compresses and writes data
func (w *GzipWriter) Write(b []byte) (int, error) {
	return w.gzip.Write(b)
}

// Close flushes gzip writer
func (w *GzipWriter) Close() error {
	return w.gzip.Close()
}

// Middleware adds gzip compression to responses
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client accepts gzip
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Create gzip writer
		gz := gzip.NewWriter(w)
		defer gz.Close()

		// Wrap response writer
		gzw := &GzipWriter{
			ResponseWriter: w,
			gzip:           gz,
		}

		// Set content encoding header
		gzw.Header().Set("Content-Encoding", "gzip")

		// Remove content-length since it will change
		gzw.Header().Del("Content-Length")

		// Call next handler
		next.ServeHTTP(gzw, r)

		// Flush remaining data
		gz.Flush()
	})
}

// SkipBodyCompression returns true if response body should not be compressed
func SkipBodyCompression(w http.ResponseWriter) bool {
	contentType := w.Header().Get("Content-Type")

	// Skip compression for already compressed formats
	skipTypes := []string{
		"image/",
		"video/",
		"audio/",
		"application/octet-stream",
		"application/zip",
		"application/gzip",
		"application/x-gzip",
	}

	for _, skipType := range skipTypes {
		if strings.HasPrefix(contentType, skipType) {
			return true
		}
	}

	return false
}
