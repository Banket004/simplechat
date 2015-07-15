// udpbroadcast project main.go
package main

import (
	"bufio"
	"client"
	"fmt"
	"os"
	"strings"
)

var app *client.Manager

func printhelp() {
	fmt.Println(`
	-help print the command string
	-online
	-offline
	-hide
	-leave
	-busy
	exit
	-chat 0.0.0.0
	-list
	-send message 0.0.0.0
	`)
}
func action(args []string) {
	switch args[0] {
	case "-help":
		printhelp()
	case "-online":
		fmt.Println("switch to online-->")
		app.Online()
	case "-offline":
		fmt.Println("switch to offline-->")
		app.Offline()
	case "-hide":
		fmt.Println("switch to hide-->")
		app.Hide()
	case "-leave":
		fmt.Println("switch to leave-->")
		app.Leave()
	case "-busy":
		fmt.Println("switch to busy-->")
		app.Busy()
	case "-chat":
		fmt.Println(args[1])
		app.CreateSingleChat(args[1])
	case "-send":
		app.SendMessage([]string{args[2]}, args[1])
	case "-list":
		for _, v := range app.ClientMap {
			fmt.Println("IP:" + string(v.IP) + "Nickname:" + string(v.Nickname) + "State:")
		}
	case "exit":
		os.Exit(0)
	}
}
func main() {
	client.LoadConfig("")
	app = client.NewManager()
	app.StartServer()
	printhelp()
	for {
		r := bufio.NewReader(os.Stdin)
		rawline, _, _ := r.ReadLine()
		line := string(rawline)
		tokens := strings.Split(line, " ")
		action(tokens)
	}
}
