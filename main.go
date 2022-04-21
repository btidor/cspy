package main

import (
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const domain = "cspy.btidor.dev"
const suffix = "." + domain + "."

var ip = net.ParseIP("37.16.6.18")

const icePassword = "STATIC-PASSWORD-123456"
const timeout = 15 * time.Second
const idchars = "abcdefghijklmnopqrstuvwxyz0123456789"

var mutex = sync.Mutex{}
var clients = make(map[string]*websocket.Conn)
var sessions = make(map[string]*net.Conn)

func main() {
	StartDNS()
	StartRTC()
	StartWSS()
	select {}
}
