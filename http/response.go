package http

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
)

type LineReader struct {
	r        io.Reader
	buffer   []byte
	Size     int
	Position int
}

type Response struct {
	Status     string
	StatusCode int

	Header map[string][]string

	ContentLength int64

	BufferSize int64
}

func (this *LineReader) read() error {
	var buff []byte = make([]byte, 1024)
	for {
		n, err := this.r.Read(buff)
		if err != nil {
			return err
		}
		this.Size += n
		this.buffer = append(this.buffer[:], buff[:n]...)
		if n < len(buff) {
			break
		}
	}
	return nil
}

func (this *LineReader) getLine() (string, bool) {
	if i := bytes.IndexByte(this.buffer, '\n'); i >= 0 {
		this.Position += i
		line := this.buffer[:i-1]
		this.buffer = this.buffer[i+1:]
		return string(line), true
	}
	return "", false
}

func (this *LineReader) ReadLine() (string, error) {
	line, ok := this.getLine()
	if ok {
		return line, nil
	}
	for {
		err := this.read()
		if err != nil {
			return "", err
		}
		line, ok := this.getLine()
		if ok {
			return line, nil
		}
	}
}

func (this *LineReader) Seek(n int) error {
	if this.Position+n < this.Size {
		return nil
	}
	n = this.Position + n
	r := this.Position
	for {
		size := 1024
		if r+1024 > n {
			size = n - r
		}
		buff := make([]byte, size)
		readed, err := this.r.Read(buff)
		if err != nil {
			return err
		}
		this.buffer = append(this.buffer[:], buff[:readed]...)
		this.Size += readed
		r += readed
		if r >= n {
			break
		}
	}
	return nil
}

func ReadResponse(r io.Reader) (*Response, error) {
	reader := &LineReader{r: r}
	resp := &Response{}

	line, err := reader.ReadLine()
	if err != nil {
		return nil, err
	}
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
		line, err := reader.ReadLine()
		if err != nil {
			return nil, errors.New("Response Header ERROR")
		}
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
	err = reader.Seek(int(resp.ContentLength))
	if err != nil {
		return nil, err
	}
	resp.BufferSize = int64(reader.Size)
	return resp, nil
}
