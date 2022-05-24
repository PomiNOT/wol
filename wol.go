package main

import (
	"encoding/binary"
	"errors"
	"net"
	"net/netip"
	"time"

	"github.com/mdlayher/arp"
)

type MachineInfo struct {
	Mac net.HardwareAddr `json:"mac"`
	Ipv4 netip.Addr `json:"ip"`
}

func NewMachineInfoFromBody(body MachineInfoBody) (*MachineInfo, error) {
	macAddr, err := net.ParseMAC(body.Mac)
	if err != nil { return nil, err }

	return &MachineInfo{
		Mac: macAddr,
	}, nil
}

type InterfaceInfo struct {
	Addresses []netip.Addr
	Interface *net.Interface
	Broadcast netip.Addr
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
	if err != nil { return nil, err }

	if iface.HardwareAddr == nil {
		return nil, errors.New("interface does not have a MAC address")
	}

	addrs, err := iface.Addrs()
	if err != nil { return nil, err }

	var ipNet *net.IPNet = nil

	for _, addr := range addrs {
		ip, ipnet, err := net.ParseCIDR(addr.String())
		if err != nil { return nil, err }

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
	}

	return ifaceInfo, nil
}

func ARPScan(networkInterface string) ([]MachineInfo, error) {
	ifaceInfo, err := GetInterfaceInfo(networkInterface)
	if err != nil { return nil, err }

	client, err := arp.Dial(ifaceInfo.Interface)
	if err != nil { return nil, err }

	for _, addr := range ifaceInfo.Addresses {
		client.Request(addr)
	}

	client.SetReadDeadline(time.Now().Add(5 * time.Second))

	temp := map[netip.Addr]MachineInfo {}
	
	for {
		packet, _, err := client.Read()
		if err != nil { break }
		
		temp[packet.SenderIP] = MachineInfo{
			Mac: packet.SenderHardwareAddr,
			Ipv4: packet.SenderIP,
		}
	}
	
	machines := make([]MachineInfo, 0, len(temp))
	for k := range temp {
		machines = append(machines, temp[k])
	}

	return machines, nil
}