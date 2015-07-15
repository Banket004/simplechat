//包含UDP和TCP服务.UDP负责客户端状态的广播和接收.TCP负责聊天和文件传输
package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"misc"
	"net"
	"os"
	"packet"
	"strconv"
	"sync"
	"time"
)

//UDP服务类
type UDPServer struct {
	tick        int          //心跳包发送时间间隔
	sendAddr    *net.UDPAddr //广播心跳包到局域网的addr(IP+port)
	receiveAddr *net.UDPAddr //接收来自同一局域网的其他客户端心跳包的addr(IP+port)
	bcastAddr   *net.UDPAddr //局域网广播IP的addr(IP+port)
	receiveConn *net.UDPConn //接收心跳包的connect
	sendchan    chan []byte
	receivechan chan []byte
	sexitchan   chan bool //用于关闭UDP发送服务协程
	rexitchan   chan bool //用于关闭UDP接收服务协程
	rlock       sync.Mutex
	slock       sync.Mutex
}

func errorCheck(err error) {
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(2)
	}
}

//NewUDPServer 根据 tick 心跳包时间间隔, tickport 监听广播包端口, sendport 发送广播包端口,
//schan 发送数据来源channel, rchan 数据接收channel,这5个参数生成UDPServer实例
func NewUDPServer(tick int, tickport int, sendport int, schan chan []byte,
	rchan chan []byte) (server *UDPServer) {
	fmt.Println("send port:", sendport, " tickport:", tickport)
	send, err := net.ResolveUDPAddr("udp", misc.GetLocalIP()+":"+strconv.Itoa(sendport))
	errorCheck(err)
	rece, err := net.ResolveUDPAddr("udp", misc.GetLocalIP()+":"+strconv.Itoa(tickport))
	errorCheck(err)
	bcast, err := net.ResolveUDPAddr("udp", net.IPv4bcast.String()+":"+strconv.Itoa(tickport))
	ctrl := make(chan bool, 1)
	server = &UDPServer{tick: tick, sendAddr: send, receiveAddr: rece,
		bcastAddr: bcast, sendchan: schan, receivechan: rchan, sexitchan: ctrl, rexitchan: ctrl}
	fmt.Println("send:" + send.String() + " ticket:" + bcast.String())
	return
}

//Send 发送UDP数据
func (server *UDPServer) Send(buf []byte) (err error) {
	return nil
}

//Receive 接收UDP数据
func (server *UDPServer) Receive(buf *[]byte) (err error) {
	return nil
}

//TickServer 将 c 传入的数据广播到局域网
func (server *UDPServer) TickServer(c <-chan []byte) (err error) {
	var conn *net.UDPConn
	fmt.Println("start TickServer")
	server.slock.Lock()
	for i := 0; i < 3; i++ {
		conn, err = net.DialUDP("udp", server.sendAddr, server.bcastAddr)
		if err == nil {
			break
		} else {
			fmt.Println("DialUDP err:", err)
		}
		time.Sleep(3 * time.Millisecond)
	}

	defer func() {
		if err := recover(); err != nil {
			fmt.Println("MY:", err)
		}
	}()
	stop := false
	p := &packet.NoticePacket{packet.STATE_OFFLINE, []byte(""), []byte("")}
	buf, _ := json.Marshal(p) //make the init packet
	tickchan := make(chan []byte, 128)
	go func(c chan []byte) { //tick function
		defer func() {
			conn.Close()
			server.slock.Unlock()
			fmt.Println("stop udp TickServer FINISH.")
		}()
		for {
			if stop {
				return
			}
			//fmt.Println("send: ", string(buf))
			_, err = conn.Write(buf)
			errorCheck(err)
			time.Sleep(time.Duration(server.tick) * time.Millisecond)
		}
	}(tickchan)
	for {
		timeout := make(chan bool)
		go func() {
			time.Sleep(5000 * time.Millisecond)
			timeout <- true
		}()
		select {
		case buf = <-c:
			//tickchan <- buf
		case <-timeout: //超时检测
		case <-server.sexitchan: //退出
			//server.slock.Lock()
			fmt.Println("stop udp TickServer....")
			stop = true
			return nil
		}
	}

	return nil
}

//ReceiveServer 将接收到的广播数据传给 c
func (server *UDPServer) ReceiveServer(c chan<- []byte) (err error) {
	fmt.Println("start ReceiveServer")
	server.rlock.Lock()
	for i := 0; i < 3; i++ {
		server.receiveConn, err = net.ListenUDP("udp", server.receiveAddr)
		if err == nil {
			break
		} else {
			fmt.Println("ListenUDP error:", err)
		}
		time.Sleep(time.Millisecond * 3)
	}
	errorCheck(err)
	defer func() {
		server.receiveConn.Close()
		server.rlock.Unlock()
		fmt.Println("stop udp ReceiveServer FINISH.")
	}()
	for {
		//fmt.Println("UDP Receive....")
		buf := make([]byte, 1024)
		server.receiveConn.SetReadDeadline(time.Now().Add(time.Duration(3) * time.Second))
		n, _, err := server.receiveConn.ReadFromUDP(buf)
		if err != nil || n < 1 {
			//continue
			n = 0
			//fmt.Println("ReadFromUDP error!")
		}
		//fmt.Println("receive: ", string(buf))
		//timeout := make(chan bool)
		//go func() {
		//	time.Sleep(100 * time.Millisecond)
		//	timeout <- true
		//}()
		select {
		case <-server.rexitchan: //退出
			//server.rlock.Lock()
			fmt.Println("stop udp ReceiveServer....")

			return nil
		//case <-timeout: //超时检测
		case c <- buf[:n]:
			//fmt.Println("read buf....")
		}

	}

	return nil
}

//Start 启动UDP服务:UDP广播和UDP接收.
func (server *UDPServer) Start() (err error) {
	log.Println("Start UDP Server:" + server.receiveAddr.String())
	go server.TickServer(server.sendchan)
	go server.ReceiveServer(server.receivechan)
	return nil
}

//Stop 停止UDP服务
func (server *UDPServer) Stop() {
	fmt.Println("Stop server...")
	server.sexitchan <- true
	server.rexitchan <- true
	fmt.Println("Stop server finish.")
}

//Reset 重启UDP服务
func (server *UDPServer) Reset() {
	server.Stop()
	server.Start()
}

//TCPServer服务类
type TCPServer struct {
	//
	addr       *net.TCPAddr            //监听服务 addr
	clientList map[string]*net.TCPConn //非离线客户端列表
	rmsgChan   chan []byte             //"read the message for client"
	wmsgChan   chan []byte             //"write the message to client"
	clientport int                     //TCP会话端口
	sport      int                     //TCP监听服务端口
	tcplock    sync.Mutex
}

//NewTCPServer 根据 ip 绑定, 监听服务端口 sport, 数据接收channel r, 数据发送channel w,
//TCP会话起始端口 cport 生成 TCPServer 类实例
func NewTCPServer(ip string, sport int, r chan []byte, w chan []byte, cport int) *TCPServer {
	addr, err := net.ResolveTCPAddr("tcp", ip+":"+strconv.Itoa(sport))
	errorCheck(err)
	return &TCPServer{addr: addr, clientList: nil, rmsgChan: r, wmsgChan: w, clientport: cport, sport: sport}
}

//ReadMsg 从 conn 读取数据并传输到 c
func ReadMsg(c chan []byte, conn *net.TCPConn) {
	//defer conn.Close()
	errTime := 0
	log.Println("ReadMsg")
	buf := make([]byte, 1024)
	for {
		if 5 < errTime { //超过5次读取错误直接退出
			break
		}
		n, err := conn.Read(buf)
		if err != nil {
			errTime++
			continue
		}
		c <- buf[0:n]

	}

}

//WriteMsg 将 c 中的数据写入到 conn
func WriteMsg(c chan []byte, conn *net.TCPConn) {
	//defer conn.Close()
	log.Println("WriteMsg")
	for {
		buf := make([]byte, 1024)
		buf = <-c
		_, err := conn.Write(buf)
		if err != nil {
			break
		}
	}
}

//Start 启动TCP监听服务
func (server *TCPServer) Start() (err error) {
	server.tcplock.Lock()
	listener, err := net.ListenTCP("tcp", server.addr)
	defer server.tcplock.Unlock()
	errorCheck(err)
	server.clientList = make(map[string]*net.TCPConn)
	log.Println("Start TCP Server: ", server.addr.String())
	go func(l *net.TCPListener) {
		defer func() {
			log.Println("End Server.")
			l.Close()
		}()
		for {
			conn, err := l.AcceptTCP()
			errorCheck(err)
			raddr := conn.RemoteAddr()
			key := base64.StdEncoding.EncodeToString([]byte(raddr.String()))
			if _, ok := server.clientList[key]; !ok {
				server.clientList[key] = conn
			}
			go ReadMsg(server.rmsgChan, conn)
			log.Println("ip: ", raddr.String(), "is connect Server.")
		}
	}(listener)

	return nil
}

//CreateSingleChat 与 ip 的客户端建立TCP会话
func (server *TCPServer) CreateSingleChat(ip string) (key string) {
	log.Println("CreateSingleChat")
	key = base64.StdEncoding.EncodeToString([]byte(ip))
	conn, ok := server.clientList[key]
	//查看此IP是否已经在非离线客户端列表中,如果不存在则创建会话并添加进去,存在则直接返回此IP在客户列表中的索引
	if !ok {
		laddr, err := net.ResolveTCPAddr("tcp", misc.GetLocalIP()+":"+strconv.Itoa(server.clientport))
		errorCheck(err)
		raddr, err := net.ResolveTCPAddr("tcp", ip+":"+strconv.Itoa(server.sport))
		server.clientport++
		conn, err = net.DialTCP("tcp", laddr, raddr)
		errorCheck(err)
		server.clientList[key] = conn
		go ReadMsg(server.rmsgChan, conn)
		log.Println("CreateSingleChat ", ip, " success!")
	}
	return
}
func (server *TCPServer) CreateMultiChats(iplist []string) {
	for _, ip := range iplist {
		server.CreateSingleChat(ip)
	}
}

//CloseSingleChat 关闭与 ip 客户端的TCP会话
func (server *TCPServer) CloseSingleChat(ip string) (key string) {
	key = base64.StdEncoding.EncodeToString([]byte(ip))
	conn, ok := server.clientList[key]
	if ok {
		conn.Close()
		delete(server.clientList, key)
	}
	return
}
func (server *TCPServer) CloseMultiChats(iplist []string) {
	for _, ip := range iplist {
		server.CloseSingleChat(ip)
	}
}

//Send 发送消息包 p 到 iplist 中所有的客户端
func (server *TCPServer) Send(iplist []string, p *packet.MessagePacket) (err error) {
	//just for text now.
	log.Println("Send msg-->", string(p.IP), ":", p.Content)
	switch p.Class {
	case packet.FILE_UNKNOWN: //未知文件类型,当作纯文本消息处理
		buf, err := json.Marshal(p)
		errorCheck(err)
		for _, ip := range iplist {
			key := base64.StdEncoding.EncodeToString([]byte(ip))
			conn, ok := server.clientList[key]
			if ok {
				conn.Write(buf)
				//WriteMsg(buf, conn)
			} else {
				log.Println("can not find the connection.")
			}
		}

	case packet.FILE_JPG:
	case packet.FILE_PNG:
	case packet.FILE_BMP:
	case packet.FILE_TXT:

	}
	log.Println("Send msg <--")
	return nil
}
