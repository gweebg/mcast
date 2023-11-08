package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/netip"
)

func MustNormalizeAddr(addr net.Addr) (newAddr netip.Addr) {

	tcp, err := net.ResolveTCPAddr("tcp", addr.String())
	if err != nil {
		log.Fatalf("cannot resolve (parse) address '%v'\n", addr.String())
	}

	ip := tcp.IP
	newAddr, ok := netip.AddrFromSlice(ip)
	if !ok {
		fmt.Println("Failed to convert net.IP to netip.Addr")
		log.Fatalf("cannot convert '%v' from net.IP to netip.Addr", addr.String())
	}

	return newAddr
}

func PrintStruct(s interface{}) {

	sJson, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Printf("%s\n", string(sJson))

}
