package client

import (
	// "encoding/json"
	// "os"
	"container/heap"
	// "fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/fatih/color"
)

var system = color.New(color.FgHiYellow).Add(color.BgBlack)
var systemresponse = color.New(color.FgHiGreen).Add(color.BgBlack)
var systemtimer = color.New(color.FgHiRed).Add(color.BgBlack)

type PqEntry struct{
	timestamp int
	pid int
}

type Client struct {
	Lock           sync.Mutex
	Id             int
	Clientlist     map[int]string
	CS 			   int
	Vectorclock	   []int
	LocalClock	   int 
	PQ 			   PriorityQueue
	RequestTimeStamp []int
}

// Flag variables:
var Higherid = false

// Message types
var REQUEST = "request"
var RESPONSE = "response"
var RELEASE = "release"
var JOIN = "join"
var ACK = "ack"

// Timer variable for timing elections:
var Timer = time.Now()
var PQ PriorityQueue
var ch = make(chan bool)


// Handle incoming messages from RPC channel
func (client *Client) HandleIncomingMessage(msg Message, reply *Message) error { 
	system.Println("RECEIVE EVENT:", msg)
	// Update clock for receive event
	client.Vectorclock = client.UpdateVectorClock(msg.Vectorclock, client.Vectorclock, msg.From)
	client.UpdateLocalClock()
	
	switch msg.Type{
		case REQUEST:
			// call the function to approve requests
			go client.ResponseHandler(msg.Vectorclock, msg.From, msg.RemoteTimestamp)
			val := <- ch
			if val{
					// Update clock for send event
					client.UpdateLocalClock()
					reply.Type = RESPONSE
					reply.From = client.Id
					reply.Vectorclock = client.Vectorclock
			}
		case RELEASE:
			system.Println("Updating local value to", msg.UpdatedValue)
			client.CS = msg.UpdatedValue
			heap.Pop(&client.PQ)
		case JOIN:
			client.Clientlist[msg.From] = msg.IP
			client.Vectorclock = append(client.Vectorclock, 0)
			reply.Type = ACK
		default:
			system.Println("Invalid message type?? How?") // will never reach this scenario...
	}
	return nil
}

// Enables everyone to create an entry in their clientlist, as well as their vector clocks.
func (client *Client) JoinNetwork(){
	for id, ip := range client.Clientlist{
		if id == client.Id {continue}
		message := Message{Type: JOIN, From: client.Id, IP: client.Clientlist[client.Id]}
		reply := client.CallRPC(message, ip)
		if reply.Type == ACK {
			client.Vectorclock = append(client.Vectorclock, 0)
			system.Println("Client", id, "has been notified")
		}
	}
	client.PQ = make(PriorityQueue, 0) 
	heap.Init(&client.PQ)
}


// Should be a go routine, because the priority queue should be able to get modified when this is occuring. 
func (client *Client) ModifyCriticalSection(){
	for {
		timeNow := time.Now()
		if timeNow.Second() % 10 == 0 {break}
	} // Even if I press one at slightly different times, they will all send concurrently at the same time.

	// Put yourself in your own priority queue
	Timer = time.Now()
	client.UpdateLocalClock()
	client.RequestTimeStamp = client.Vectorclock
	heap.Push(&client.PQ, PqEntry{pid: client.Id, timestamp: client.LocalClock})

	// Send the request to all the other nodes in the network
	req := Message{Type: REQUEST, From: client.Id, Vectorclock: client.RequestTimeStamp, RemoteTimestamp: client.LocalClock}
	for id, ip := range client.Clientlist{
		if id == client.Id {continue}
		reply := client.CallRPC(req, ip)
		if (reply.Type == RESPONSE){
			req.Responselist = append(req.Responselist, reply.From)
			client.Vectorclock = client.UpdateVectorClock(reply.Vectorclock, client.Vectorclock, reply.From)
			client.UpdateLocalClock()
			system.Println("Received a response. Response list:", req.Responselist)
		}
	}
	// By the time the client comes to this point, it will for sure have all the responses from everyone. Need to ensure the timeout for rpc call is sufficiently large enough.
	// You only need to wait till my request gets to the head of the queue
	
	var input int
	for {
		time.Sleep(time.Second * 1)
		if client.PQ[0].pid == client.Id { 
			system.Println("I now have permission from everyone else, and I am at the head of the queue.")
			system.Println("Enter value")
			// fmt.Scanln(&input) // <- this might be too slow when trying to compare the time, so just randomize it!
			input = rand.Intn(1000)
			system.Println("Entered", input)
			system.Println("Setting CS...")
			client.CS = input
			break
		}
	}

	// Once done, you will need to send a release lock message to everyone in the clientlist.
	go client.ReleaseSender(input)
}


func (client *Client) ResponseHandler(from []int, fromId int, remoteTime int){
	heap.Push(&client.PQ, PqEntry{pid: fromId, timestamp: remoteTime})
	system.Println("Inside response handler, PQ:", client.ShowPriorityQueue())
	var comparision []int 
	if client.RequestTimeStamp != nil{ // If I made a request, then my request timestamp is updated. I should compare with this 
		comparision = client.RequestTimeStamp
	} else{
		comparision = client.Vectorclock
	}
	for {
		if !client.CheckIfInPQ() {// If I am no longer in the PQ, means you can send response 
			ch <- true 
			systemresponse.Println(client.Id, "is not in PQ, therefore it is sending response right away")
			break
		}
		// Compare vector clocks
		if (!client.IsSmaller(comparision, from) && !client.IsSmaller(from, comparision) ){
			systemresponse.Println(client.Id, "has encountered concurrent event with", fromId)
			// case 3: events are concurrent, and own id is smaller, don' return true
			// case 4: events are concurrent, and own id is larger, return true
			if client.Id > fromId {
				ch <- true
				system.Println(client.Id, "is greater than", fromId, comparision, from, "So I am sending response right away")
				break
			}
			systemresponse.Println("Waiting to send response...")
			continue
		} else {
			// case 1: vector clock of self is earlier: don't return true
			// case 2: vector clock of self is later: return true
			if client.IsSmaller(from, comparision) { // if my own vector clock is larger, return true
				ch <- true
				systemresponse.Println(client.Id, "is greater than", fromId, comparision, from)
				break
			} 
			systemresponse.Println("Waiting to send response...")
			continue
		}
	}
}

func (client *Client) ReleaseSender(value int) {
	system.Println(client.Id, "is releasing...")
	heap.Pop(&client.PQ) // This should be your own
	client.RequestTimeStamp = nil

	release := Message{Type: RELEASE, From: client.Id, UpdatedValue: client.CS}
	for id, IP := range client.Clientlist{
		if id == client.Id {continue} // if own id then continue
		client.CallRPC(release, IP)
	}
	systemtimer.Println("**********\nTime elapsed:", time.Since(Timer), "\n**********")
}