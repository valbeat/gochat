package main

import (
	"github.com/gorilla/websocket"
	"net/http"
	"log"
	"github.com/valbeat/trace"
	"time"
	"io"
)

type room struct {
	// forward is a channel that holds incoming messages.
	// that should be forwarded to the other clients.
	forward chan []byte
	// join is a channel for clients wishing to join the room.
	join chan *client
	// leave is a channel form clients wishing to leave from the room.
	leave chan *client
	// clients holds all current clients in this room.
	clients map[*client]bool
	// tracer received log from this room
	tracer *trace.Tracer
}

func newRoom(w io.Writer) *room{
	return &room{
		forward: make(chan []byte),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]bool),
		tracer: trace.New(w),
	}
}

const (
	socketBufferSize = 1024
	messageBufferSize = 256
)

var upgrader = &websocket.Upgrader{ReadBufferSize:socketBufferSize, WriteBufferSize:messageBufferSize}

func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("ServeHTTP:", err)
		return
	}
	client := &client{
		socket: socket,
		send: make(chan []byte, messageBufferSize),
		room: r,
	}
	r.join <- client
	defer func() {r.leave <-client}()
	go client.write()
	client.read()
}


func (r *room) run() {
	for {
		select {
		case client := <-r.join:
			// 入室
			r.clients[client] = true
			r.tracer.Trace("[" + time.Now().Format("2006-01-02 15:04:05") + "] New client joined")
		case client := <-r.leave:
			// 退室
			delete(r.clients, client)
			close(client.send)
			r.tracer.Trace("[" + time.Now().Format("2006-01-02 15:04:05") + "] Client left")
		case msg := <-r.forward:
			// すべてのクライアントにメッセージを転送
			for client := range r.clients {
				select {
				case client.send <-msg:
					// メッセージを送信
					r.tracer.Trace("[" + time.Now().Format("2006-01-02 15:04:05") + "] Sent to client")
				default:
					// 送信に失敗
					// クライアント側でsendチャネルからメッセージを読み込んでチャネルのバッファに空きができる前に、
					// roomからさらにメッセージを送信しようとした時
					delete(r.clients, client)
					close(client.send)
				}
			}
		}
	}
}
