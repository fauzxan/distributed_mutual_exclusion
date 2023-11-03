package client

import (
	// "encoding/json"
	"net/rpc"
	// "os"
	"sync"
	"time"

	"github.com/fatih/color"
)

var system = color.New(color.FgCyan).Add(color.BgBlack)

type Client struct {
	Lock           sync.Mutex
	Id             int
	Clientlist     map[int]string
	CS 			   int
	Vectorclock	   []int
}

// Flag variables:
var Higherid = false

// Message types
var REQUEST = "request"
var RESPONSE = "response"
var RELEASE = "correction"
var JOIN = "join"
var ACK = "ack"

// Timer variable for timing elections:
var Timer = time.Now()

func (client *Client) HandleIncomingMessage(msg Message, reply *Message) error { // Handle communication
	switch msg.Type{
	case REQUEST:
		system.Println("Received message of type REQUEST from", msg.From)
		// call the function to approve requests
	case RESPONSE:
		system.Println("Received message of type RESPONSE from", msg.From)
	case RELEASE:
		system.Println("Received message of type RELEASE from", msg.From)
	case JOIN:
		system.Println("Received mesage of type JOIN from", msg.From)
		client.Clientlist[msg.From] = msg.IP
		client.Vectorclock = append(client.Vectorclock, 0)
		reply.Type = ACK
	default:

	}
	return nil
}

func (client *Client) JoinNetwork(){
	for id, ip := range(client.Clientlist){
		if id == client.Id {continue}
		message := Message{Type: JOIN, From: client.Id, IP: client.Clientlist[client.Id]}
		reply := client.CallRPC(message, ip)
		if reply.Type == ACK {
			client.Vectorclock = append(client.Vectorclock, 0)
			system.Println("Client", id, "has been notified")
		}
	}
}


/*
***************************
UTILITY FUNCTIONS
***************************
*/

func (client *Client) Printclients() map[int]string{
	return client.Clientlist
}

func (client *Client) Getmax() int {
	client.Lock.Lock()
	defer client.Lock.Unlock()
	max := 0
	for id := range client.Clientlist {
		if id > max {
			max = id
		}
	}
	return max
}

func (client *Client) Check(n int) bool {
	/*
		true means that id is not taken | id is not there in the clientlist
		false means that id is taken | id is there in the clientlist
	*/
	for id := range client.Clientlist {
		if n == id {
			return false
		}
	}
	return true
}

func (client *Client) GetUniqueRandom() int {
	for i := 0; ; i++ {
		check := client.Check(i)
		if check {
			return i
		}
	}
}

//  Node utility function to call RPC given a request message, and a destination IP address
func (client *Client) CallRPC(msg Message, IP string) Message {
	clnt, err := rpc.Dial("tcp", IP)
	reply := Message{}
	if err != nil {
		system.Println("Error Dialing RPC:", err)
		system.Println("Received reply", reply)
		return reply
	}
	err = clnt.Call("Client.HandleIncomingMessage", msg, &reply)
	if err != nil {
		system.Println("Faced an error trying to call RPC:", err)
		system.Println("Received reply", reply)
		return reply
	}
	system.Println("Received reply", reply, "from", IP)
	return reply
}

func (client *Client) ShowClock() []int{
	return client.Vectorclock
}

// func (client *Client) updateclientlist() {
// 	jsonData, err := json.Marshal(client.Clientlist)
// 	if err != nil {
// 		system.Println("Could not marshal into json data")
// 		panic(err)
// 	}
// 	err = os.WriteFile("clientlist.json", jsonData, os.ModePerm)
// 	if err != nil {
// 		system.Println("Could not write into file")
// 		panic(err)
// 	}
// }
