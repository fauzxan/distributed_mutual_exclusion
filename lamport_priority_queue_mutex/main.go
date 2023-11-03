package main

import (
	"core/client"
	"encoding/json"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/fatih/color"
)

var system = color.New(color.FgCyan).Add(color.BgBlack)

func showmenu(){
	system.Println("********************************")
	system.Println("\t\tMENU")
	system.Println("Press 1 to make request to enter CS")
	system.Println("Press 2 to show local vector clock")
	system.Println("Press 3 to show local clientlist")
	system.Println("Press m to see the menu")
	system.Println("********************************")
}

func main(){
	system.Println("yo")

	me := client.Client{Lock: sync.Mutex{}}
	
	// Read data from "nearby neighbour" aka clientlist.json, then set my clientlist
	data, err := os.ReadFile("clientlist.json")
	if err != nil {
		system.Println("Error reading file clientlist.json")
	}
	var clientList map[int]string
	err = json.Unmarshal(data, &clientList)
	if err != nil {
		panic(err)
	}
	system.Println("Clientlist obtained!")
	me.Clientlist = clientList

	// Update the "nearby neighbour" of the new clientlist.json
	if len(me.Clientlist) == 0 {
		// If I am the only client so far, then I will set my self as both the coordinator, as well as communicate using the port
		me.Id = 0
		me.Clientlist[me.Id] = "127.0.0.1:3000"
		me.Vectorclock = make([]int, 1)
	} else {
		// Otherwise, I will create a new clientid, and also a new port from which I will communicate with
		me.Id = me.Getmax() + 1
		me.Clientlist[me.Id] = "127.0.0.1:" + strconv.Itoa(3000+me.Id)
		me.Vectorclock = make([]int, 1)
	}

	// Write back to json file
	jsonData, err := json.Marshal(me.Clientlist)
	if err != nil {
		system.Println("Could not marshal into json data")
		panic(err)
	}
	err = os.WriteFile("clientlist.json", jsonData, os.ModePerm)
	if err != nil {
		system.Println("Could not write into file")
		panic(err)
	}

	system.Println(me.Clientlist)
	// Resolve and listen to address of self
	address, err := net.ResolveTCPAddr("tcp", me.Clientlist[me.Id])
	if err != nil {
		panic("Error resolving TCP address")
	}
	inbound, err := net.ListenTCP("tcp", address)
	if err != nil {
		panic("Could not listen to TCP address")
	}
	rpc.Register(&me)
	system.Println("Client is runnning at IP address", address)
	go rpc.Accept(inbound)

	// method to join network called. 
	go me.JoinNetwork()

	var input string
	for {
		showmenu()
		fmt.Scanln(&input)

		switch input{
		case "1":
			
		case "2":
			system.Println("My vector clock:", me.ShowClock())
		case "3":
			system.Println("My clientlist:", me.Printclients())
		default:
			system.Println("Enter a valid option!!!")
		}
		time.Sleep(100 * time.Millisecond)
	}
}