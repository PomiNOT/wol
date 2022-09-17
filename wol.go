package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/mdlayher/arp"
)

type MachineInfo struct {
	Mac  net.HardwareAddr `json:"mac"`
	Ipv4 netip.Addr       `json:"ip"`
}

func (m *MachineInfo) UnmarshalJSON(data []byte) error {
	tmp := map[string]string{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	macAddr, err := net.ParseMAC(tmp["mac"])
	if err != nil {
		return err
	}
	m.Mac = macAddr

	ipAddr := net.ParseIP(tmp["ip"])
	if ipAddr != nil {
		netipAddr, _ := netip.AddrFromSlice(ipAddr)
		m.Ipv4 = netipAddr
	}

	return nil
}

func (m *MachineInfo) MarshalJSON() ([]byte, error) {
	return []byte(
		fmt.Sprintf(`{"mac":"%s","ip":"%s"}`, m.Mac.String(), m.Ipv4.String()),
	), nil
}

var existingScanners map[string]*ARPScanner
var scannersMutex sync.Mutex

type ARPScanner struct {
	interfaceInfo    *InterfaceInfo
	cache            map[string]*MachineInfo
	client           *arp.Client
	foundCh          chan *MachineInfo
	accessMutex      sync.Mutex
	waitCond         *sync.Cond
	connectedClients uint
}

func ARPScannerNew(networkInterface string) (*ARPScanner, error) {
	ifaceInfo, err := GetInterfaceInfo(networkInterface)
	if err != nil {
		return nil, err
	}

	client, err := arp.Dial(ifaceInfo.Interface)
	if err != nil {
		return nil, err
	}

	scanner := &ARPScanner{
		interfaceInfo:    ifaceInfo,
		cache:            make(map[string]*MachineInfo),
		client:           client,
		foundCh:          make(chan *MachineInfo),
		connectedClients: 0,
		accessMutex:      sync.Mutex{},
	}
	scanner.waitCond = sync.NewCond(&scanner.accessMutex)

	go scanner.worker()

	return scanner, nil
}

func (s *ARPScanner) worker() {
	for {
		s.accessMutex.Lock()
		for s.connectedClients == 0 {
			log.Println("No listening clients, pausing worker...")
			s.waitCond.Wait()
		}
		s.accessMutex.Unlock()

		for _, addr := range s.interfaceInfo.Addresses {
			s.client.Request(addr)
		}

		s.client.SetReadDeadline(time.Now().Add(10 * time.Second))

		for {
			packet, _, err := s.client.Read()
			if err != nil {
				break
			}

			machine := &MachineInfo{
				Mac:  packet.SenderHardwareAddr,
				Ipv4: packet.SenderIP,
			}

			s.accessMutex.Lock()

			addrStr := packet.SenderHardwareAddr.String()
			//Non blocking send
			select {
			case s.foundCh <- machine:
			default:
			}
			s.cache[addrStr] = machine

			s.accessMutex.Unlock()
		}
	}
}

func (s *ARPScanner) StreamWriter() func(*bufio.Writer) {
	s.accessMutex.Lock()
	s.connectedClients++
	log.Println("A device connected")

	return func(w *bufio.Writer) {
		log.Println("Serving cache")

		for _, machine := range s.cache {
			machineJSON, err := machine.MarshalJSON()
			if err != nil {
				log.Printf("StreamWriter: cannot convert to json")
				continue
			}

			fmt.Fprintf(w, "event: found\n")
			fmt.Fprintf(w, "data: %s\n\n", machineJSON)
			err = w.Flush()
			if err != nil {
				s.connectedClients--
				return
			}
		}

		s.waitCond.Signal()
		s.accessMutex.Unlock()

		log.Println("Listening for new events...")
		for {
			machine := <-s.foundCh
			machineJSON, err := machine.MarshalJSON()
			if err != nil {
				log.Printf("StreamWriter: cannot convert to json")
				continue
			}

			fmt.Fprintf(w, "event: found\n")
			fmt.Fprintf(w, "data: %s\n\n", machineJSON)
			err = w.Flush()
			if err != nil {
				s.accessMutex.Lock()
				log.Println("A device disconnected")
				s.connectedClients--
				s.accessMutex.Unlock()
				return
			}
		}
	}
}

func ARPScannerInstance(networkInterface string) (*ARPScanner, error) {
	scannersMutex.Lock()
	defer scannersMutex.Unlock()

	if existingScanners == nil {
		existingScanners = make(map[string]*ARPScanner)
	}

	scanner := existingScanners[networkInterface]
	if scanner == nil {
		newScanner, err := ARPScannerNew(networkInterface)
		if err != nil {
			return nil, err
		}

		existingScanners[networkInterface] = newScanner
		return newScanner, nil
	} else {
		return scanner, nil
	}
}
