package misc

import (
	"log"
	"net"
	"strings"
)

//GetLocalIP 获取本机的IP地址,不包括以0,255结尾的掩码和以1结尾的网关
func GetLocalIP() (ip string) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal(err.Error())
		ip = ""
	} else {
		for _, addr := range addrs {
			str := addr.String()
			last := strings.Split(str, ".")[3]
			if last != "0" && last != "255" && last != "1" {
				ip = str
				break
			}
		}
	}
	return
}
