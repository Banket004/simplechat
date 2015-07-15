package server

import (
	"encoding/json"
	"misc"
	"packet"
	//"fmt"
	"testing"
	"time"
)

func TestOps(t *testing.T) {
	//UDPTest(t)
	TCPTest(t)
	time.Sleep(5000 * time.Millisecond)
	//var str string
	//fmt.Scan(&str)
}
func UDPTest(t *testing.T) {
	receChan := make(chan []byte, 1024)
	sendChan := make(chan []byte, 1024)
	broadcast := NewUDPServer(1000, 998, 999)
	broadcast.Start(sendChan, receChan)
	go func() {
		for {
			rbuf := <-receChan
			var notice packet.NoticePacket
			err := json.Unmarshal(rbuf, &notice)
			if err != nil {
				t.Error(err.Error())
				return
			}
			check(t, &notice)
		}
	}()
	ip := []byte(misc.GetLocalIP())
	p := packet.NewNoticePacket(packet.STATE_ONLINE, ip, []byte("Jack Lee"))
	sbuf, err := json.Marshal(p)
	if err != nil {
		t.Error(err.Error())
		return
	}
	sendChan <- sbuf
	sendChan <- []byte("{\"State\":1,\"IP\":\"MTAuMC4yMC4xNjg=\"}")
	sendChan <- []byte("{\"State\":2,\"IP\":\"MTAuMC4yMC4xNjg=\"}")
	sendChan <- []byte("{\"State\":3,\"IP\":\"MTAuMC4yMC4xNjg=\"}")
	sendChan <- []byte("{\"State\":4,\"IP\":\"MTAuMC4yMC4xNjg=\"}")
	sendChan <- []byte("{\"State\":5,\"IP\":\"MTAuMC4yMC4xNjg=\"}")
	//time.Sleep(5000 * time.Millisecond)
}

var sport int = 1999
var cport int = 2999

func CreateTCPInst(t *testing.T) {
	receChan := make(chan []byte, 1024)
	sendChan := make(chan []byte, 1024)
	server := NewTCPServer(misc.GetLocalIP(), sport, receChan, sendChan, cport)
	sport += 10
	cport += 500
	server.Start()
	server.CreateSingleChat(misc.GetLocalIP())
	server.Send([]string{misc.GetLocalIP()}, &packet.MessagePacket{IP: []byte(misc.GetLocalIP()), Class: packet.FILE_UNKNOWN, Content: "hello", Path: "0000"})
	go func() {
		for {
			rbuf := <-receChan
			var p packet.MessagePacket
			err := json.Unmarshal(rbuf, &p)
			if err != nil {
				t.Error(err.Error())
				return
			}
			t.Log("IP:", string(p.IP), ":", p.Content)
		}
	}()
}
func TCPTest(t *testing.T) {
	for i := 0; i < 5; i++ {
		CreateTCPInst(t)
	}
}
func check(t *testing.T, p *packet.NoticePacket) {
	t.Log(string(p.IP))
	switch p.State {
	case packet.STATE_ONLINE:
		t.Log("online.")
	case packet.STATE_OFFLINE:
		t.Log("offline.")
	case packet.STATE_BUSY:
		t.Log("busy.")
	case packet.STATE_LEAVE:
		t.Log("leave.")
	case packet.STATE_HIDE:
		t.Log("hide")
	default:
		t.Log("unknown state")
	}
}
