package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"misc"
	"packet"
	"runtime"
	"server"
	"time"
)

// ClientTimer 类,管理已经连接的客户端的定时器,定时器超时则设置相应客户端为离线状态
type ClientTimer struct {
	Client
	t *time.Timer
}

func NewClientTimer(c *Client) *ClientTimer {
	t := time.AfterFunc(5*time.Second, func() {
		fmt.Println("timer active: ", string(c.IP))
		c.State = packet.STATE_OFFLINE
	})
	return &ClientTimer{Client: *c, t: t}
}
func (ct *ClientTimer) Reset() {
	ct.t.Reset(5 * time.Second)
	fmt.Println("ClientTimer Reset.")
}

// Manager 管理类
type Manager struct {
	USC       chan []byte            "udpsendchan"    //UDP 发送channel
	URC       chan []byte            "udpreceivechan" //UDP 接收channel
	TSC       chan []byte            "tcpsendchan"    //TCP 发送channel
	TRC       chan []byte            "tcpreceivechan" //TCP 接收channel
	ClientMap map[string]*Client     "clientlist"     //非离线用户列表
	TimerMap  map[string]*time.Timer "timermap"       //非离线用户超时定时器列表
	UDPSrv    *server.UDPServer      "UDPserver"      //UDP类实例
	TCPSrv    *server.TCPServer      "TCPServer"      //TCP类实例
	User      *UserInfo              "user"           //当前登录用户信息
	isOffline bool                   //表示当前客户端账户是否处于离线状态,用于控制UDP和TCP服务的运行和终止
}
type UserInfo struct {
	NickName string
	State    int
	IP       string
}

var ticktime int
var sendUDPPort int
var listenUDPPort int
var sendTCPPort int
var listenTCPPort int
var nickname string

const TICK_SECOND = 10

//LoadConfig 加载配置文件,设置各全局参数
func LoadConfig(filename string) error {
	ticktime = 5000
	sendUDPPort = 998
	listenUDPPort = 999
	sendTCPPort = 2999
	listenTCPPort = 1999
	nickname = "testuser"
	return nil
}

//NewManager 生成 Manager 类实例
func NewManager() *Manager {
	user := new(UserInfo)
	user.IP = misc.GetLocalIP()
	user.State = packet.STATE_ONLINE
	user.NickName = nickname
	usc := make(chan []byte /*, 1024*/)
	urc := make(chan []byte /*, 1024*/)
	tsc := make(chan []byte /*, 1024*/)
	trc := make(chan []byte /*, 1024*/)
	map1 := make(map[string]*Client)
	map2 := make(map[string]*time.Timer)
	udpsrv := server.NewUDPServer(ticktime, sendUDPPort, listenUDPPort, usc, urc)
	tcpsrv := server.NewTCPServer(user.IP, listenTCPPort, trc, tsc, sendTCPPort)
	return &Manager{usc, urc, tsc, trc, map1, map2, udpsrv, tcpsrv, user, false}
}

//UpdateClientList 更新用户列表中用户 c 的状态
func (m *Manager) UpdateClientMap(c *Client) {
	ipstr := base64.StdEncoding.EncodeToString(c.IP)
	item, ok := m.ClientMap[ipstr]
	if ok {
		if c.State == packet.STATE_OFFLINE || c.State == packet.STATE_HIDE {
			delete(m.ClientMap, ipstr)
			delete(m.TimerMap, ipstr)
			m.TCPSrv.CloseSingleChat(string(c.IP))
			fmt.Println("client offline or hide: ", string(c.IP))
		} else {
			fmt.Println("change other state!", string(c.IP), "state: ", c.State)
			item.State = c.State
			item.Nickname = c.Nickname
			if it, ok := m.TimerMap[ipstr]; ok {
				it.Reset(TICK_SECOND * time.Second)
			}

		}
	} else {
		if c.State == packet.STATE_OFFLINE || c.State == packet.STATE_HIDE {
			return
		}
		fmt.Println("found a new client!", string(c.IP))
		m.ClientMap[ipstr] = c
		if it, ok := m.TimerMap[ipstr]; ok { //the timer is exist, just reset.
			it.Reset(TICK_SECOND * time.Second)
		} else { //the time is not exist, create the new one.
			t := time.AfterFunc(TICK_SECOND*time.Second, func() {
				c, ok := m.ClientMap[ipstr]
				if !ok {
					return
				}
				fmt.Println("timer active: ", string(c.IP))
				m.ClientMap[ipstr].State = packet.STATE_OFFLINE
			})
			m.TimerMap[ipstr] = t
		}

	}
}

//MessageProcedure 处理TCP消息包
func (m *Manager) MessageProcedure(p *packet.MessagePacket) {
	fmt.Println("tcp receive:" + p.Content + " from: " + string(p.IP))
	switch p.Class {
	case packet.FILE_JPG:
		fmt.Println("the message type is JPG.")
	case packet.FILE_PNG:
		fmt.Println("the message type is PNG.")
	case packet.FILE_BMP:
		fmt.Println("the message type is BMP.")
	case packet.FILE_TXT:
		fmt.Println("the message type is TXT.")
	case packet.FILE_UNKNOWN:
		fmt.Println("the message type is UNKNOWN.")
	}
}

//StartServer 启动客户端的服务
func (m *Manager) StartServer() error {
	go func() { //UDP消息处理协程
		oldstate := packet.STATE_OFFLINE
		for {
			buf := <-m.URC
			if len(buf) == 0 {
				continue
			}
			var c Client
			err := json.Unmarshal(buf, &c)
			if nil != err {
				fmt.Println("unknown format! buf:", string(buf))
				continue
			}
			if string(c.IP) == m.User.IP { //收到自己发给自己的消息包,直接忽略
				continue
			}
			if c.State != oldstate {
				fmt.Println("change state.")
				m.UpdateClientMap(&c)
				oldstate = c.State
			} //else {
			fmt.Println("no state need update. ", string(c.IP), "state: ", c.State)
			ipstr := base64.StdEncoding.EncodeToString(c.IP)
			if it, ok := m.TimerMap[ipstr]; ok { //reset tick
				it.Reset(TICK_SECOND * time.Second)
			}
			//}

		}
	}()
	go func() { //TCP消息处理协程
		for {
			buf := <-m.TRC
			var p packet.MessagePacket
			err := json.Unmarshal(buf, &p)
			if nil != err {
				continue
			}
			m.MessageProcedure(&p)
			//fmt.Println("tcp receive:" + p.Content + " from: " + string(p.IP))
		}
	}()
	m.UDPSrv.Start()
	m.TCPSrv.Start()
	time.Sleep(5 * time.Millisecond)
	m.ChangeState(packet.STATE_ONLINE)
	go func() { //清除离线客户端
		for {
			time.Sleep(5 * time.Second)
			for i, v := range m.ClientMap {
				if v.State == packet.STATE_OFFLINE {
					fmt.Println("clear client: ", string(v.IP))
					delete(m.ClientMap, i)
					break
				}
			}
			runtime.Gosched()
		}

	}()
	return nil
}
func (m *Manager) StopServer() error {
	m.UDPSrv.Stop()
	return nil
}

//ChangeState 更改用户自身状态
func (m *Manager) ChangeState(state int) {
	fmt.Println("change own state.")
	if m.isOffline && state != packet.STATE_OFFLINE {
		fmt.Println("start the UDP server.")
		m.UDPSrv.Start()
		m.isOffline = false
	}
	m.User.State = state
	p := &packet.NoticePacket{state, []byte(m.User.IP), []byte(m.User.NickName)}
	buf, err := json.Marshal(p)
	if err == nil {
		t := make(chan bool)
		go func() {
			time.Sleep(5 * time.Second)
			t <- true
		}()
		select {
		case m.USC <- buf:
		case <-t:
			fmt.Println("ChangeState timeout.")
		}

	}
	switch state {
	case packet.STATE_ONLINE:
	case packet.STATE_OFFLINE:
		m.UDPSrv.Stop()
		for k := range m.ClientMap {
			delete(m.ClientMap, k)
		}
		for k := range m.TimerMap {
			delete(m.TimerMap, k)
		}
		m.isOffline = true
	case packet.STATE_BUSY:
	case packet.STATE_LEAVE:
	case packet.STATE_HIDE:
	default:
	}
}

//Online 切换到"在线"状态
func (m *Manager) Online() {
	m.ChangeState(packet.STATE_ONLINE)
}

//Offline 切换到"离线"状态
func (m *Manager) Offline() {
	m.ChangeState(packet.STATE_OFFLINE)
}

//Leave 切换到"离开"状态
func (m *Manager) Leave() {
	m.ChangeState(packet.STATE_LEAVE)
}

//Busy 切换到"忙碌"状态
func (m *Manager) Busy() {
	m.ChangeState(packet.STATE_BUSY)
}

//Hide 切换到"隐身"状态
func (m *Manager) Hide() {
	m.ChangeState(packet.STATE_HIDE)
}

//CreateSingleChat 创建与地址为 ip 的用户之间的链接
func (m *Manager) CreateSingleChat(ip string) (ret bool) {
	m.TCPSrv.CreateSingleChat(ip)
	return true
}

//SendMessage 发送消息 msg 给 iplist 列表中的用户.
func (m *Manager) SendMessage(iplist []string, msg string) {
	p := &packet.MessagePacket{IP: []byte(m.User.IP), Class: packet.FILE_UNKNOWN, Content: msg, Path: ""}
	m.TCPSrv.Send(iplist, p)
}

//SendFile 发送绝对路径为 path 的文件给 iplist 列表中的用户
func (m *Manager) SendFile(iplist []string, path string) {
	p := &packet.MessagePacket{IP: []byte(m.User.IP), Class: packet.FILE_TXT, Content: "", Path: path}
	m.TCPSrv.Send(iplist, p)
}
