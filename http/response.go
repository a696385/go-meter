package http

import (
	"bufio"
	"errors"
	"strconv"
	"strings"
)

type Response struct {
	Status     string
	StatusCode int

	Header map[string][]string

	ContentLength int64

	BufferSize int64
}

func readLine(r *bufio.Reader) (string, error) {
	var line []byte
	for {
		l, more, err := r.ReadLine()
		if err != nil {
			return "", err
		}
		if line == nil && !more {
			return string(l), nil
		}
		line = append(line, l...)
		if !more {
			break
		}
	}
	return string(line), nil
}

func ReadResponse(r *bufio.Reader) (*Response, error) {
	resp := &Response{}

	line, err := readLine(r)
	if err != nil {
		return nil, err
	}
	f := strings.SplitN(line, " ", 3)
	resp.BufferSize += int64(len(line) + 2)

	if len(f) < 2 {
		return nil, errors.New("Response Header ERROR")
	}

	reasonPhrase := ""
	if len(f) > 2 {
		reasonPhrase = f[2]
	}
	resp.Status = f[1] + " " + reasonPhrase
	resp.StatusCode, err = strconv.Atoi(f[1])
	if err != nil {
		return nil, errors.New("malformed HTTP status code")
	}

	resp.Header = make(map[string][]string)
	for {
		line, err := readLine(r)
		if err != nil {
			return nil, errors.New("Response Header ERROR")
		}
		resp.BufferSize += int64(len(line) + 2)
		if len(line) == 0 {
			break
		} else {
			f := strings.SplitN(line, ":", 2)
			resp.Header[f[0]] = append(resp.Header[strings.TrimSpace(f[0])], strings.TrimSpace(f[1]))
		}
	}

	if cl := resp.Header["Content-Length"]; len(cl) > 0 {
		i, err := strconv.ParseInt(cl[0], 10, 0)
		if err == nil {
			resp.ContentLength = i
		}
	}

	_, err = r.Peek(int(resp.ContentLength))
	if err != nil {
		return nil, err
	}
	resp.BufferSize += int64(resp.ContentLength)

	return resp, nil
}
