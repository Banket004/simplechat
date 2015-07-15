package packet

import (
//"net"
)

const (
	STATE_ONLINE  = 0
	STATE_OFFLINE = 1
	STATE_BUSY    = 2
	STATE_LEAVE   = 3
	STATE_HIDE    = 4
)
const (
	TYPE_CONTENT = 0
	TYPE_FILE    = 1
)
const (
	FILE_JPG     = 0
	FILE_PNG     = 1
	FILE_BMP     = 2
	FILE_TXT     = 3
	FILE_UNKNOWN = 4
)

//结构体会转化为JSON对象，并且只有结构体里边以大写字母开头的可被导出的字段才会
//被转化输出，而这些可导出的字段会作为JSON对象的字符串索引。
//心跳包结构
type NoticePacket struct {
	State    int    "state"    //账号状态
	IP       []byte "ip"       //账号所登录PC的IP
	Nickname []byte "nickname" //账号昵称
}

//消息包结构
type MessagePacket struct {
	IP      []byte
	Class   int    //消息包数据类型,为"纯文本类型"或者"文本+文件"
	Content string //文本消息
	Path    string //文件所在的绝对路径
}

func NewNoticePacket(state int, ip []byte, nickname []byte) (packet *NoticePacket) {
	packet = &NoticePacket{state, ip, nickname[0:64]}
	return
}
