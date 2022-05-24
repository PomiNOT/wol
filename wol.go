package main

import (
	"encoding/binary"
	"errors"
	"net"
	"net/netip"
	"regexp"
	"strings"
	"time"

	"github.com/mdlayher/arp"
)

type MachineInfo struct {
	Mac string `json:"mac"`
	Ip string `json:"ip"`
}

func (m *MachineInfo) validMac() (bool, error) {
	lowerMac := strings.ToLower(m.Mac)
	matched, err := regexp.Match("^([a-f0-9]{2}:){5}[a-f0-9]{2}$", []byte(lowerMac))
	return matched, err
}

func getIpv4List(ipNet *net.IPNet) []netip.Addr {
	ips := make([]netip.Addr, 0, 255)
	p := binary.BigEndian.Uint32(ipNet.IP)
	m := binary.BigEndian.Uint32(ipNet.Mask)
	var host uint32 = 0

	for ;host < ^m; host++ {
		ip := make([]byte, 4)
		binary.BigEndian.PutUint32(ip, p | host)
		addr, _ := netip.AddrFromSlice(ip)
		ips = append(ips, addr)
	}
	return ips
}

func getInterfaceInfo(networkInterface string) ([]netip.Addr, *net.Interface, error) {
	iface, err := net.InterfaceByName(networkInterface)
	if err != nil { return nil, nil, err }

	if iface.HardwareAddr == nil {
		return nil, nil, errors.New("interface does not have a MAC address")
	}

	addrs, err := iface.Addrs()
	if err != nil { return nil, nil, err }

	var ipNet *net.IPNet = nil

	for _, addr := range addrs {
		ip, ipnet, err := net.ParseCIDR(addr.String())
		if err != nil { return nil, nil, err }

		if ip.To4() != nil {
			ipNet = ipnet
			break
		}
	}

	if ipNet == nil {
		return nil, nil, errors.New("no IPv4 addresses could be found")
	}

	addresses := getIpv4List(ipNet)

	return addresses, iface, nil
}

func ARPScan(networkInterface string) ([]MachineInfo, error) {
	addresses, iface, err := getInterfaceInfo(networkInterface)
	if err != nil { return nil, err }

	client, err := arp.Dial(iface)
	if err != nil { return nil, err }

	for _, addr := range addresses {
		client.Request(addr)
	}

	client.SetReadDeadline(time.Now().Add(5 * time.Second))

	machines := make([]MachineInfo, 0)

	for {
		packet, _, err := client.Read()
		if err != nil { break }

		machines = append(machines, MachineInfo{
			Mac: packet.TargetHardwareAddr.String(),
			Ip: packet.TargetIP.String(),
		})
	}

	return machines, nil
}