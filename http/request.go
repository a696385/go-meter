package http

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/url"
)

type Request struct {
	Method string

	URL *url.URL

	Header map[string][]string

	Body          io.Reader
	ContentLength int64

	Host string

	BufferSize int64
}

func (req *Request) Write(w io.Writer) error {

	bw := &bytes.Buffer{}

	fmt.Fprintf(bw, "%s %s HTTP/1.1\r\n", valueOrDefault(req.Method, "GET"), req.URL.RequestURI())
	fmt.Fprintf(bw, "Host: %s\r\n", req.Host)

	userAgent := ""
	if req.Header != nil {
		if ua := req.Header["User-Agent"]; len(ua) > 0 {
			userAgent = ua[0]
		}
	}
	if userAgent != "" {
		fmt.Fprintf(bw, "User-Agent: %s\r\n", userAgent)
	}

	if req.Method == "POST" || req.Method == "PUT" {
		fmt.Fprintf(bw, "Content-Length: %d\r\n", req.ContentLength)
	}

	if req.Header != nil {
		for key, values := range req.Header {
			if key == "User-Agent" || key == "Content-Length" || key == "Host" {
				continue
			}
			for _, value := range values {
				fmt.Fprintf(bw, "%s: %s\r\n", key, value)
			}
		}
	}

	io.WriteString(bw, "\r\n")

	if req.Method == "POST" || req.Method == "PUT" {
		bodyReader := bufio.NewReader(req.Body)
		_, err := bodyReader.WriteTo(bw)
		if err != nil {
			return err
		}
	}
	req.BufferSize = int64(bw.Len())
	_, err := bw.WriteTo(w)
	return err
}

func valueOrDefault(value string, def string) string {
	if len(value) == 0 {
		return def
	} else {
		return value
	}
}
