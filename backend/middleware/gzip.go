package middleware

import (
	"compress/gzip"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

type gzipResponseWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.writer.Write(b)
}

func (g *gzipResponseWriter) WriteString(s string) (int, error) {
	return g.writer.Write([]byte(s))
}

func Gzip() gin.HandlerFunc {
	return func(c *gin.Context) {
		acceptEncoding := c.GetHeader("Accept-Encoding")
		if !strings.Contains(acceptEncoding, "gzip") {
			c.Next()
			return
		}

		contentType := c.Writer.Header().Get("Content-Type")
		if !shouldCompress(contentType, c.Request.URL.Path) {
			c.Next()
			return
		}

		gz := gzip.NewWriter(c.Writer)
		defer gz.Close()

		c.Writer.Header().Set("Content-Encoding", "gzip")
		c.Writer.Header().Del("Content-Length")

		gzw := &gzipResponseWriter{
			ResponseWriter: c.Writer,
			writer:         gz,
		}
		c.ResponseWriter = gzw

		c.Next()

		gz.Close()
	}
}

func shouldCompress(contentType, path string) bool {
	if strings.HasPrefix(contentType, "text/") {
		return true
	}
	if strings.Contains(contentType, "json") {
		return true
	}
	if strings.Contains(contentType, "javascript") {
		return true
	}
	if strings.Contains(contentType, "css") {
		return true
	}
	if strings.Contains(contentType, "svg") {
		return true
	}
	if strings.Contains(contentType, "xml") {
		return true
	}
	if strings.Contains(contentType, "font") {
		return true
	}

	if strings.HasSuffix(path, ".js") ||
		strings.HasSuffix(path, ".css") ||
		strings.HasSuffix(path, ".html") ||
		strings.HasSuffix(path, ".json") ||
		strings.HasSuffix(path, ".svg") ||
		strings.HasSuffix(path, ".xml") {
		return true
	}

	return false
}

func GzipStatic() gin.HandlerFunc {
	return func(c *gin.Context) {
		acceptEncoding := c.GetHeader("Accept-Encoding")
		if !strings.Contains(acceptEncoding, "gzip") {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		if !isStaticAsset(path) {
			c.Next()
			return
		}

		gzPath := path + ".gz"
		if fileExists(gzPath) {
			c.Request.URL.Path = gzPath
			c.Header("Content-Encoding", "gzip")
			c.Header("Vary", "Accept-Encoding")
			c.Next()
			return
		}

		c.Next()
	}
}

func isStaticAsset(path string) bool {
	ext := strings.ToLower(path)
	return strings.HasSuffix(ext, ".js") ||
		strings.HasSuffix(ext, ".css") ||
		strings.HasSuffix(ext, ".html") ||
		strings.HasSuffix(ext, ".json") ||
		strings.HasSuffix(ext, ".svg") ||
		strings.HasSuffix(ext, ".woff") ||
		strings.HasSuffix(ext, ".woff2") ||
		strings.HasSuffix(ext, ".ttf") ||
		strings.HasSuffix(ext, ".eot")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
