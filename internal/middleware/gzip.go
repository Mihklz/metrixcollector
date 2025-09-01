package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// compressWriter реализует интерфейс http.ResponseWriter и позволяет прозрачно для сервера
// сжимать передаваемые данные и выставлять правильные HTTP-заголовки
type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

func (c *compressWriter) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

func (c *compressWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer и досылает все данные из буфера.
func (c *compressWriter) Close() error {
	return c.zw.Close()
}

// compressReader реализует интерфейс io.ReadCloser и позволяет прозрачно для сервера
// декомпрессировать получаемые от клиента данные
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

// gzipResponseWriter оборачивает http.ResponseWriter и проверяет Content-Type
type gzipResponseWriter struct {
	http.ResponseWriter
	compressWriter *compressWriter
	acceptsGzip    bool
}

func (g *gzipResponseWriter) WriteHeader(statusCode int) {
	// Проверяем Content-Type только при первом вызове WriteHeader
	if g.compressWriter == nil && g.acceptsGzip {
		contentType := g.Header().Get("Content-Type")
		shouldCompress := strings.Contains(contentType, "application/json") ||
			strings.Contains(contentType, "text/html")

		if shouldCompress {
			g.compressWriter = newCompressWriter(g.ResponseWriter)
		}
	}

	if g.compressWriter != nil {
		g.compressWriter.WriteHeader(statusCode)
	} else {
		g.ResponseWriter.WriteHeader(statusCode)
	}
}

func (g *gzipResponseWriter) Write(data []byte) (int, error) {
	// Если WriteHeader не был вызван, вызываем его с кодом 200
	if g.compressWriter == nil && g.acceptsGzip {
		g.WriteHeader(http.StatusOK)
	}

	if g.compressWriter != nil {
		return g.compressWriter.Write(data)
	}
	return g.ResponseWriter.Write(data)
}

func (g *gzipResponseWriter) Close() error {
	if g.compressWriter != nil {
		return g.compressWriter.Close()
	}
	return nil
}

// WithGzip добавляет поддержку gzip сжатия/декомпрессии
func WithGzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// по умолчанию устанавливаем оригинальный http.ResponseWriter как тот,
		// который будем передавать следующей функции
		ow := w

		// проверяем, что клиент умеет получать от сервера сжатые данные в формате gzip
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip {
			// оборачиваем ResponseWriter для отложенной проверки Content-Type
			gzipWriter := &gzipResponseWriter{
				ResponseWriter: w,
				acceptsGzip:    true,
			}
			ow = gzipWriter
			defer gzipWriter.Close()
		}

		// проверяем, что клиент отправил серверу сжатые данные в формате gzip
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			// оборачиваем тело запроса в io.Reader с поддержкой декомпрессии
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// меняем тело запроса на новое
			r.Body = cr
			defer cr.Close()
		}

		// передаём управление хендлеру
		next.ServeHTTP(ow, r)
	})
}
