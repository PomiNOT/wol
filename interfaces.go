package main

import (
	"encoding/binary"
	"errors"
	"net"
	"net/netip"
)

type InterfaceInfo struct {
	Addresses []netip.Addr
	Interface *net.Interface
	Broadcast netip.Addr
	Name      string
}

func getIpv4List(ipNet *net.IPNet) []netip.Addr {
	ips := make([]netip.Addr, 0, 255)
	p := binary.BigEndian.Uint32(ipNet.IP)
	m := binary.BigEndian.Uint32(ipNet.Mask)
	var host uint32 = 0

	for ; host < ^m; host++ {
		ip := make([]byte, 4)
		binary.BigEndian.PutUint32(ip, p|host)
		addr, _ := netip.AddrFromSlice(ip)
		ips = append(ips, addr)
	}
	return ips
}

func getBroadcastAddress(ipNet *net.IPNet) netip.Addr {
	broadcast := make([]byte, 4)
	for i := 0; i < 4; i++ {
		broadcast[i] = ipNet.IP[i] | ^ipNet.Mask[i]
	}
	addr, _ := netip.AddrFromSlice(broadcast)
	return addr
}

func GetInterfaceInfo(networkInterface string) (*InterfaceInfo, error) {
	iface, err := net.InterfaceByName(networkInterface)
	if err != nil {
		return nil, err
	}

	if iface.HardwareAddr == nil {
		return nil, errors.New("interface does not have a MAC address")
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	var ipNet *net.IPNet = nil

	for _, addr := range addrs {
		ip, ipnet, err := net.ParseCIDR(addr.String())
		if err != nil {
			return nil, err
		}

		if ip.To4() != nil {
			ipNet = ipnet
			break
		}
	}

	if ipNet == nil {
		return nil, errors.New("no IPv4 addresses could be found")
	}

	addresses := getIpv4List(ipNet)
	ifaceInfo := &InterfaceInfo{
		Addresses: addresses,
		Interface: iface,
		Broadcast: getBroadcastAddress(ipNet),
		Name:      networkInterface,
	}

	return ifaceInfo, nil
}
