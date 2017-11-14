package utils

import (
	"fmt"

	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/urfave/negroni"
)

var LoggerDefaultDateFormat = time.RFC3339

// ALogger interface
type ALogger interface {
	Println(v ...interface{})
	Printf(format string, v ...interface{})
}

// Logger is a middleware handler that logs the request as it goes in and the response as it goes out.
type Logger struct {
	// ALogger implements just enough log.Logger interface to be compatible with other implementations
	ALogger
	dateFormat string
}

// NewLogger returns a new Logger instance
func NewLogger() *Logger {

	logger := &Logger{ALogger: log.New(os.Stdout, "[debug log] ", 0), dateFormat: LoggerDefaultDateFormat}
	return logger
}

func (l *Logger) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()
	dumpReq, err := httputil.DumpRequest(r, true)
	if err != nil {
		http.Error(rw, err.Error(), 500)
	}
	next(rw, r)
	res := rw.(negroni.ResponseWriter)
	l.ALogger.Printf("%v | %v | %v | %v | %v | %v\nRequestBody:\n%v", start.Format(l.dateFormat), res.Status(), time.Since(start), r.Host, r.Method, r.URL.Path, string(dumpReq))
}

// formatRequest generates ascii representation of a request
func formatRequestBody(r *http.Request) (string, error) {
	// dump request
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		return "", err
	}
	reqBody := fmt.Sprintf("%q", dump)
	return reqBody, err
}
