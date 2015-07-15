package client

import (
//"packet"
)

type Client struct {
	State    int    "state"
	IP       []byte "ip"
	Nickname []byte "nickname"
}
