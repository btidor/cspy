package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	gonanoid "github.com/matoous/go-nanoid"
)

var upgrader = websocket.Upgrader{
	HandshakeTimeout: timeout,
	CheckOrigin: func(r *http.Request) bool {
		// Open to all origins
		return true
	},
}

func StartWSS() {
	http.HandleFunc("/ws", WSSHandler)
	go func() {
		server := &http.Server{Addr: ":80"}
		err := server.ListenAndServe()
		panic(err)
	}()
}

func WSSHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	id, err := gonanoid.Generate(idchars, 12)
	if err != nil {
		log.Println(err)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()
	clients[id] = conn
	conn.WriteJSON(map[string]string{"id": id})

	go func() {
		time.Sleep(timeout)
		mutex.Lock()
		defer mutex.Unlock()
		if conn, ok := clients[id]; ok {
			conn.WriteControl(websocket.CloseMessage,
				[]byte{}, time.Now().Add(timeout))
			conn.Close()
			delete(clients, id)
		}
	}()
}
