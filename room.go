package main

import (
	"log"
	"net/http"

	"github.com/aashishrathi/trace"

	"github.com/gorilla/websocket"
)

// We don't want to access the map directly for joining and leaving
// the room because two goroutines accessing it concurrently can
// lead to unexpected results

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

type room struct {
	// forward holds incoming messages to be sent to clients
	forward chan []byte
	// join channel for client who wanna join
	join chan *client
	// leave channel for client who wanna leave
	leave chan *client
	// all clients currently in the room
	clients map[*client]bool
	// tracer for trace activity in room
	tracer trace.Tracer
}

func newRoom() *room {
	return &room{
		forward: make(chan []byte),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]bool),
		tracer:  trace.Off(),
	}
}

func (r *room) run() {
	// this will run forever in the bg as a goroutine
	for {
		select {
		case client := <-r.join:
			// joining
			r.clients[client] = true
			r.tracer.Trace("New client joined")
		case client := <-r.leave:
			//leaving
			delete(r.clients, client)
			close(client.send)
			r.tracer.Trace("Client left")
		case msg := <-r.forward:
			r.tracer.Trace("Message received: ", string(msg))
			// forward message to all clients
			for client := range r.clients {
				client.send <- msg
				r.tracer.Trace(" -- sent to client")
			}
		}
	}
}

var upgrader = &websocket.Upgrader{ReadBufferSize: socketBufferSize,
	WriteBufferSize: socketBufferSize}

func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("ServeHTTP:", err)
		return
	}
	client := &client{
		socket: socket,
		send:   make(chan []byte, messageBufferSize),
		room:   r,
	}

	// Add the client to the list
	// Let this thread run the blocking client.read(), and another one
	// for the blocking client.write(). They're blocking because each
	// contains a for loop on a channel not yet closed
	// Once the client disconnects, it is added tot the leave channel
	// to be removed from the map of clients
	r.join <- client
	defer func() { r.leave <- client }()
	go client.write()
	client.read()
}
