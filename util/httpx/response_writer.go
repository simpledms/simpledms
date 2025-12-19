package httpx

import (
	"log"
	"net/http"

	"github.com/simpledms/simpledms/app/simpledms/renderable"
)

type ResponseWriter interface {
	http.ResponseWriter
	WriteData(value any, statusCode int)
	// necessary for command/query in one request handling
	// because headers cannot set after writing body
	AddRenderables(widgets ...renderable.Renderable)
	Renderables() []renderable.Renderable
	HasRenderables() bool
	HasDataWritten() bool
}

type responseWriter struct {
	http.ResponseWriter
	renderables    []renderable.Renderable
	hasDataWritten bool
}

func NewResponseWriter(rw http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: rw,
	}
}

func (qq *responseWriter) Write(data []byte) (int, error) {
	dataWritten, err := qq.ResponseWriter.Write(data)
	if err != nil {
		return dataWritten, err
	}
	if dataWritten > 0 {
		qq.hasDataWritten = true
	}
	return dataWritten, nil
}

func (qq *responseWriter) WriteData(value interface{}, statusCode int) {
	qq.WriteHeader(statusCode)

	if str, ok := value.(string); ok {
		dataWritten, err := qq.ResponseWriter.Write([]byte(str))
		if err != nil {
			log.Println(err)
			return
		}
		if dataWritten > 0 {
			qq.hasDataWritten = true
		}
		return
	}

	dataWritten, err := qq.ResponseWriter.Write(value.([]byte))
	if err != nil {
		log.Println(err)
		return
	}
	if dataWritten > 0 {
		qq.hasDataWritten = true
	}
	return
}

func (qq *responseWriter) AddRenderables(widgets ...renderable.Renderable) {
	qq.renderables = append(qq.renderables, widgets...)
}

func (qq *responseWriter) Renderables() []renderable.Renderable {
	return qq.renderables
}

func (qq *responseWriter) HasRenderables() bool {
	return len(qq.renderables) > 0
}

// not in use as of 28 April 2025
func (qq *responseWriter) HasDataWritten() bool {
	return qq.hasDataWritten
}
