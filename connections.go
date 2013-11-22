package main

import "net"

type Connection struct {
	conn    net.Conn
	manager *ConnectionManager
}

type ConnectionManager struct {
	conns  chan *Connection
	config *Config
}

func NewConnectionManager(config *Config) (result *ConnectionManager) {

	result = &ConnectionManager{config: config, conns: make(chan *Connection, config.Connections)}
	for i := 0; i < config.Connections; i++ {
		connection := &Connection{manager: result}
		if connection.Dial() != nil {
			ConnectionErrors++
		}
		result.conns <- connection
	}
	return
}

func (this *ConnectionManager) Get() *Connection {
	return <-this.conns
}

func (this *Connection) Dial() error {
	if this.IsConnected() {
		this.Disconnect()
	}
	conn, err := net.Dial("tcp4", this.manager.config.Url.Host)
	if err == nil {
		this.conn = conn
	}
	return err
}

func (this *Connection) Disconnect() {
	this.conn.Close()
	this.conn = nil
}

func (this *Connection) IsConnected() bool {
	return this.conn != nil
}

func (this *Connection) Return() {
	this.manager.conns <- this
}
