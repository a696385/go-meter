package http

import (
	"bufio"
	"errors"
	"net/textproto"
	"strconv"
	"strings"
)

type Response struct {
	Request    *Request
	Status     string
	StatusCode int

	Header map[string][]string

	ContentLength int64

	BufferSize int64
}

func ReadResponse(r *bufio.Reader, tr *textproto.Reader) (*Response, error) {
	resp := &Response{}

	line, err := tr.ReadLine()
	if err != nil {
		return nil, err
	}
	resp.BufferSize += int64(len(line) + 2)
	f := strings.SplitN(line, " ", 3)

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
		line, err := tr.ReadLine()
		resp.BufferSize += int64(len(line) + 2)
		if err != nil {
			return nil, errors.New("Response Header ERROR")
		}
		if len(line) == 0 {
			break
		} else {
			f := strings.SplitN(line, ":", 2)
			if len(f) == 2 {
				resp.Header[f[0]] = append(resp.Header[strings.TrimSpace(f[0])], strings.TrimSpace(f[1]))
			}
		}
	}

	if cl := resp.Header["Content-Length"]; len(cl) > 0 {
		i, err := strconv.ParseInt(cl[0], 10, 0)
		if err == nil {
			resp.ContentLength = i
		}
	}
	if resp.ContentLength > 0 {

		read := 0
		for {
			p := make([]byte, resp.ContentLength-int64(read))
			n, err := r.Read(p)
			if err != nil {
				return nil, err
			}
			read += n
			if int64(read) == resp.ContentLength {
				break
			}
		}
	}
	resp.BufferSize += int64(resp.ContentLength)
	return resp, nil
}
