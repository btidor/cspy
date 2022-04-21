package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/pion/datachannel"
	"github.com/pion/dtls/v2"
	"github.com/pion/logging"
	"github.com/pion/sctp"
	"github.com/pion/stun"
)

var certificate tls.Certificate
var config *dtls.Config

var req = new(stun.Message)
var rsp = new(stun.Message)

func StartRTC() {
	var err error
	certificate, err = tls.LoadX509KeyPair("/var/cert.pem", "/var/key.pem")
	if err != nil {
		panic(err)
	}
	config = &dtls.Config{
		Certificates:         []tls.Certificate{certificate},
		InsecureSkipVerify:   true,
		ExtendedMasterSecret: dtls.RequestExtendedMasterSecret,
	}

	go func() {
		host, err := net.ResolveUDPAddr("udp", "fly-global-services:1234")
		if err != nil {
			panic(err)
		}
		socket, err := net.ListenUDP("udp", host)
		if err != nil {
			panic(err)
		}
		buf := make([]byte, 16384)
		for {
			n, addr, err := socket.ReadFrom(buf)
			if err != nil {
				panic(err)
			} else if n == 0 {
				return
			}

			if stun.IsMessage(buf[:n]) {
				STUNHandler(socket, addr, buf[:n])
			} else {
				alt := make([]byte, n)
				copy(alt, buf[:n])
				go DTLSHandler(socket, addr, alt)
			}
		}
	}()
}

func STUNHandler(socket *net.UDPConn, addr net.Addr, buf []byte) {
	err := stun.Decode(buf, req)
	if err != nil || req.Type != stun.BindingRequest {
		return
	}

	user, ok := req.Attributes.Get(stun.AttrUsername)
	parts := strings.Split(string(user.Value), ":")
	if !ok {
		return
	}
	priority, ok := req.Attributes.Get(stun.AttrPriority)
	if !ok {
		return
	}

	if strings.HasPrefix(parts[0], "CSPY-") {
		slug := strings.TrimPrefix(parts[0], "CSPY-")
		go func() {
			mutex.Lock()
			defer mutex.Unlock()
			if conn, ok := clients[slug]; ok {
				conn.WriteJSON(map[string]bool{"rtcuser": true})
			}
		}()
		return
	}

	err = rsp.Build(
		&stun.BindingSuccess,
		&stun.XORMappedAddress{
			IP:   addr.(*net.UDPAddr).IP,
			Port: addr.(*net.UDPAddr).Port,
		},
	)
	if err != nil {
		panic(err)
	}
	err = req.AddTo(rsp)
	if err != nil {
		return
	}
	err = stun.NewShortTermIntegrity(icePassword).AddTo(rsp)
	if err != nil {
		return
	}
	err = stun.Fingerprint.AddTo(rsp)
	if err != nil {
		return
	}
	_, err = socket.WriteTo(rsp.Raw, addr)
	if err != nil {
		return
	}

	username := stun.NewUsername(fmt.Sprintf("%s:%s", parts[1], parts[0]))
	req.Build(
		&stun.BindingRequest,
		&username,
		&stun.RawAttribute{
			Type:   stun.AttrICEControlled,
			Length: 8,
			Value:  []byte{0, 0, 0, 0, 0, 0, 0, 1},
		},
		&priority,
	)
	err = stun.NewShortTermIntegrity(parts[0]).AddTo(req)
	if err != nil {
		return
	}
	err = stun.Fingerprint.AddTo(req)
	if err != nil {
		return
	}
	_, err = socket.WriteTo(req.Raw, addr)
	if err != nil {
		return
	}

	DTLSStart(socket, addr)
}

func DTLSHandler(socket *net.UDPConn, addr net.Addr, buf []byte) {
	mutex.Lock()
	defer mutex.Unlock()
	if conn, ok := sessions[addr.String()]; ok {
		(*conn).Write(buf)
	}
}

func DTLSStart(socket *net.UDPConn, addr net.Addr) {
	saddr := addr.String()
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := sessions[saddr]; ok {
		// Session in progress
		return
	}

	external, internal := net.Pipe()
	sessions[saddr] = &external

	go func() {
		defer func() {
			mutex.Lock()
			defer mutex.Unlock()
			internal.Close()
			if conn, ok := sessions[saddr]; ok {
				(*conn).Close()
				delete(sessions, saddr)
			}
		}()

		conn, err := dtls.Client(internal, config)
		if err != nil {
			return
		}

		loggerFactory := logging.NewDefaultLoggerFactory()
		client, err := sctp.Client(sctp.Config{
			NetConn:       conn,
			LoggerFactory: loggerFactory,
		})
		if err != nil {
			return
		}
		defer client.Close()
		channel, err := datachannel.Accept(client, &datachannel.Config{
			ChannelType:   datachannel.ChannelTypeReliable,
			Negotiated:    false,
			Label:         "label",
			LoggerFactory: loggerFactory,
		})
		if err != nil {
			return
		}
		defer channel.Close()

		go func() {
			time.Sleep(timeout)
			channel.Close()
		}()

		buf := make([]byte, 1024)
		n, _, err := channel.ReadDataChannel(buf)
		if err != nil {
			return
		}

		mutex.Lock()
		defer mutex.Unlock()
		if conn, ok := clients[string(buf[:n])]; ok {
			conn.WriteJSON(map[string]bool{"rtcdata": true})
		}
	}()

	go func() {
		buf := make([]byte, 16384)
		for {
			n, err := external.Read(buf)
			if err != nil {
				return
			}
			socket.WriteTo(buf[:n], addr)
		}
	}()
}
