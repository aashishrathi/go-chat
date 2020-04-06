package main

import (
	"github.com/gorilla/websocket"
)

type client struct {
	// web socket for this client
	socket *websocket.Conn
	// channel for sending messages
	send chan []byte
	// room where connected
	room *room
}

func (c *client) read() {
	defer c.socket.Close()
	for {
		_, msg, err := c.socket.ReadMessage()
		if err != nil {
			return
		}
		c.room.forward <- msg
	}
}

func (c *client) write() {
	defer c.socket.Close()
	for msg := range c.send {
		err := c.socket.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			return
		}
	}
}
