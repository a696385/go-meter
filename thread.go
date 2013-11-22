package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/a696385/go-meter/http"
	"net/url"
	"os"
	"sync/atomic"
	"time"
)

func NewThread(config *Config) {
	timerAllow := time.NewTicker(time.Duration(250) * time.Millisecond)
	allow := int32(config.MRQ / 4 / config.Threads)
	if config.MRQ == -1 {
		allow = 2147483647
	} else if allow <= 0 {
		allow = 1
	}
	var connectionErrors int32 = 0
	currentAllow := allow
	for {
		select {
		//Set allow requests per time timer
		case <-timerAllow.C:
			currentAllow = allow
		//Get free tcp connection
		case connection := <-config.ConnectionManager.conns:
			currentAllow--
			//Return connection to pool if allowed request is 0
			if currentAllow < 0 {
				connection.Return()
			} else {
				//Create request object
				req := getRequest(config.Method, config.Url, config.Source.GetNext())
				//For reconnect mode need disconnect
				if config.Reconnect && connection.IsConnected() {
					connection.Disconnect()
				}
				//Connect to server if connection lost
				if !connection.IsConnected() {
					if connection.Dial() != nil {
						connectionErrors++
					}
				}
				//Send request if we connected
				if connection.IsConnected() {
					go writeSocket(connection, req, config.RequestStats)
				} else {
					connection.Return()
				}
			}
		//Wait exit event
		case <-config.WorkerQuit:
			//Store errors
			atomic.AddInt32(&ConnectionErrors, connectionErrors)
			//Complete exit
			config.WorkerQuited <- true
			return
		}
	}
}

func getRequest(method string, URL *url.URL, body *[]byte) *http.Request {
	if method == "POST" || method == "PUT" {
		return &http.Request{
			Method: method,
			URL:    URL,
			Header: map[string][]string{
				"Connection": {"keep-alive"},
			},
			Body:          bytes.NewBuffer(*body),
			ContentLength: int64(len(*body)),
			Host:          URL.Host,
		}
	} else {
		//Use source data as URL request or original URL
		var (
			r   *url.URL
			err error
		)
		if body != nil {
			r, err = url.Parse(string(*body))
			if err != nil {
				fmt.Printf("ERROR: URL is broken %s\n", string(*body))
				os.Exit(1)
			}
		} else {
			r = URL
		}
		return &http.Request{
			Method: method,
			URL:    r,
			Header: map[string][]string{
				"Connection": {"keep-alive"},
			},
			Host: URL.Host,
		}
	}
}

func writeSocket(connection *Connection, req *http.Request, read chan *RequestStats) {
	result := &RequestStats{}
	//Anyway return connection and send respose
	defer func() {
		connection.Return()
		read <- result
	}()

	now := time.Now()
	conn := connection.conn
	bw := bufio.NewWriter(conn)
	//Write request
	err := req.Write(bw)
	if err != nil {
		result.WriteError = err
		return
	}
	err = bw.Flush()
	if err != nil {
		result.WriteError = err
		return
	}
	//Read response
	res, err := http.ReadResponse(bufio.NewReader(conn))
	if err != nil {
		result.ReadError = err
		return
	}
	//Store info
	result.Duration = time.Now().Sub(now)
	result.NetOut = req.BufferSize
	result.NetIn = res.BufferSize
	result.ResponseCode = res.StatusCode
	//Free memory
	req.Body = nil
}
