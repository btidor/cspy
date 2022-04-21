package main

import (
	"strings"

	"github.com/miekg/dns"
)

func StartDNS() {
	dns.HandleFunc(domain, DNSHandler)

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
}

func DNSHandler(w dns.ResponseWriter, r *dns.Msg) {
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
}
