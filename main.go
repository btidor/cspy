package main

import (
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	gonanoid "github.com/matoous/go-nanoid"
	"github.com/miekg/dns"
	"github.com/pion/stun"
)

const domain = "cspy.btidor.dev"
const suffix = "." + domain + "."

var ip = net.ParseIP("37.16.6.18")

const dummyUser = "abcdef" // for WebRTC
const timeout = 15 * time.Second
const idchars = "abcdefghijklmnopqrstuvwxyz0123456789"

var clients = make(map[string]*websocket.Conn)
var mutex = sync.Mutex{}

func main() {
	dns.HandleFunc(domain, func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)

		name := r.Question[0].Name
		slug := strings.TrimSuffix(name, suffix)
		parts := strings.Split(strings.TrimSpace(slug), "-")
		if len(parts) > 1 {
			mutex.Lock()
			defer mutex.Unlock()
			if conn, ok := clients[parts[len(parts)-1]]; ok {
				conn.WriteJSON(map[string][]string{"query": parts})
			}
		}

		a := new(dns.A)
		a.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60}
		a.A = ip
		m.Answer = append(m.Answer, a)
		w.WriteMsg(m)
	})

	go func() {
		// We must bind on port 53. If we bind on another port and rely on Fly
		// to proxy packets, they'll be dropped somewhere in the Go stack due to
		// the port mismatch.
		server := &dns.Server{Addr: "fly-global-services:53", Net: "udp"}
		err := server.ListenAndServe()
		panic(err)
	}()

	go func() {
		// This might be magic, but there seem to be fewer connection resets if
		// we run TCP on a different internal port from UDP.
		server := &dns.Server{Addr: ":5353", Net: "tcp"}
		err := server.ListenAndServe()
		panic(err)
	}()

	go func() {
		socket, err := net.ListenPacket("udp", "fly-global-services:1234")
		if err != nil {
			panic(err)
		}
		buf := make([]byte, 1024)
		msg := new(stun.Message)
		for {
			n, _, err := socket.ReadFrom(buf)
			if err != nil {
				panic(err)
			}
			err = stun.Decode(buf[:n], msg)
			if err != nil {
				continue
			}

			user, err := msg.Get(stun.AttrUsername)
			if err != nil || string(user) == dummyUser {
				continue
			}

			go func() {
				parts := strings.Split(strings.TrimSpace(string(user)), ":")
				if len(parts) > 1 {
					mutex.Lock()
					defer mutex.Unlock()
					if conn, ok := clients[parts[0]]; ok {
						conn.WriteJSON(map[string]bool{"webrtc": true})
					}
				}
			}()
		}
	}()

	upgrader := websocket.Upgrader{
		HandshakeTimeout: timeout,
		CheckOrigin: func(r *http.Request) bool {
			// Open to all origins
			return true
		},
	}
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
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
	})
	server := &http.Server{Addr: ":80"}
	err := server.ListenAndServe()
	panic(err)
}
