package main

import (
	"log"
	"net"
	"time"

	"github.com/hashicorp/mdns"
)

const mdnsService = "_psstd._tcp"
const mdnsDomain  = "local."

// registerMDNS advertises this node on the LAN via mDNS.
// Returns a stop func.
func registerMDNS(hostname string, port int) func() {
	info := []string{"psstd node"}
	svc, err := mdns.NewMDNSService(hostname, mdnsService, mdnsDomain, "", port, nil, info)
	if err != nil {
		log.Printf("mDNS register error: %v", err)
		return func() {}
	}
	server, err := mdns.NewServer(&mdns.Config{Zone: svc})
	if err != nil {
		log.Printf("mDNS server error: %v", err)
		return func() {}
	}
	log.Printf("mDNS: registered %s.%s%s on port %d", hostname, mdnsService, mdnsDomain, port)
	return func() { server.Shutdown() }
}

// discoverPeers scans mDNS for other psstd nodes and returns their addresses.
func discoverPeers(gossipPort int) []string {
	entries := make(chan *mdns.ServiceEntry, 16)
	var peers []string

	go func() {
		params := &mdns.QueryParam{
			Service:     mdnsService,
			Domain:      "local",
			Timeout:     2 * time.Second,
			Entries:     entries,
			DisableIPv6: false,
		}
		if err := mdns.Query(params); err != nil {
			log.Printf("mDNS query error: %v", err)
		}
		close(entries)
	}()

	self := selfIPs()
	for entry := range entries {
		addr := entry.AddrV4
		if addr == nil {
			addr = entry.AddrV6
		}
		if addr == nil {
			continue
		}
		// Don't add ourselves
		if isSelf(addr, self) {
			continue
		}
		peer := fmt.Sprintf("%s:%d", addr.String(), gossipPort)
		log.Printf("mDNS: discovered peer %s (%s)", entry.Name, peer)
		peers = append(peers, peer)
	}
	return peers
}

func selfIPs() []net.IP {
	var ips []net.IP
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				ips = append(ips, ipnet.IP)
			}
		}
	}
	return ips
}

func isSelf(addr net.IP, self []net.IP) bool {
	for _, ip := range self {
		if ip.Equal(addr) {
			return true
		}
	}
	return false
}
