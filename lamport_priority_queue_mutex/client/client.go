package client

import (
	"container/heap"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/fatih/color"
)

var system = color.New(color.FgHiYellow).Add(color.BgBlack)
var systemresponse = color.New(color.FgHiGreen).Add(color.BgBlack)
var systemtimer = color.New(color.FgHiRed).Add(color.BgBlack)
var systemmodify = color.New(color.FgHiMagenta).Add(color.BgBlack)

// PQEntry represents an entry in the priority queue.
type PQEntry struct {
	Timestamp int
	ID        int
}

type Client struct {
	Lock           sync.Mutex
	Id             int
	Clientlist     map[int]string
	CS 			   int
	LocalClock	   int 
	PQ 			   PriorityQueue
	RequestTimeStamp int
}

// Flag variables:
var Higherid = false

const INF = math.MaxInt16

// Message types
const REQUEST = "request"
const RESPONSE = "response"
const RELEASE = "release"
const JOIN = "join"
const ACK = "ack"

// Timer variable for timing elections:
var Timer = time.Now()
var PQ PriorityQueue

var IncomingChannels []chan Message

// Handle incoming messages from RPC channel
func (client *Client) HandleIncomingMessage(msg Message, reply *Message) error { 
	system.Println("RECEIVE EVENT:", msg)
	// Update clock for receive event
	// client.Vectorclock = client.UpdateVectorClock(msg.Vectorclock, client.Vectorclock, msg.From)
	client.UpdateLocalClock()
	
	switch msg.Type{
		case REQUEST:
			// call the function to approve requests
			responseChannel := make(chan bool)
			go client.ResponseHandler(msg.Vectorclock, msg.From, msg.RemoteTimestamp, responseChannel)
			val := <- responseChannel
			if val{
				client.UpdateLocalClock()
				reply.Type = RESPONSE
				reply.From = client.Id
			}
		case RELEASE:
			system.Println("Updating local value to", msg.UpdatedValue)
			client.CS = msg.UpdatedValue
			heap.Pop(&client.PQ)
			reply.Type = ACK
		case JOIN:
			client.Clientlist[msg.From] = msg.IP
			// client.Vectorclock = append(client.Vectorclock, 0)
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
		responseChannel := make(chan Message)
		go client.CallRPC(message, ip, responseChannel)
		reply := <- responseChannel
		if reply.Type == ACK {
			// client.Vectorclock = append(client.Vectorclock, 0)
			system.Println("Client", id, "has been notified")
		}
	}
	client.PQ = make(PriorityQueue, 0) 
	client.RequestTimeStamp = INF
	IncomingChannels = make([]chan Message, len(client.Clientlist))
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
	client.RequestTimeStamp = client.LocalClock
	systemmodify.Println("Adding my request to the PQ with timestamp", client.RequestTimeStamp)
	heap.Push(&client.PQ, &PQEntry{ID: client.Id, Timestamp: client.RequestTimeStamp})

	// Send the request to all the other nodes in the network
	req := Message{Type: REQUEST, From: client.Id, RemoteTimestamp: client.RequestTimeStamp}
	responseChannels := make([]chan Message, len(client.Clientlist))
	for id, ip := range client.Clientlist{
		if id == client.Id {continue}
		// reply := client.CallRPC(req, ip)
		responseChannel := make(chan Message)
        responseChannels[id] = responseChannel
		go client.CallRPC(req, ip, responseChannel)
	}
	// You only need to wait till my request gets to the head of the queue
    for id, responseChannel := range responseChannels {
        if id == client.Id {
            continue
        }
		reply := <-responseChannel
		// Process the response...
		req.Responselist = append(req.Responselist, reply.From)
		client.UpdateLocalClock()
		systemmodify.Println("Received a response. Response list:", req.Responselist)

    }
	system.Println("here")
	var input int
	for {
		time.Sleep(time.Second * 1)
		if len(req.Responselist) == (len(client.Clientlist)-1) && client.PQ[0].ID == client.Id { 
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


func (client *Client) ResponseHandler(from []int, fromId int, remoteTime int, ch chan bool){
	systemresponse.Println("Adding remote request into PQ with timestamp", remoteTime)
	heap.Push(&client.PQ, &PQEntry{ID: fromId, Timestamp: remoteTime})

	for {
		time.Sleep(1500 * time.Millisecond)
		if client.RequestTimeStamp ==  INF{// If I am no longer in the PQ, means you can send response 
			ch <- true 
			systemresponse.Println(client.Id, "ID is not in PQ, therefore it is sending response right away")
			break
		}
		// Compare vector clocks
		if (client.RequestTimeStamp == remoteTime){
			systemresponse.Println(client.Id, "has encountered concurrent event with", fromId)
			// case 3: events are concurrent, and own id is smaller, don' return true
			// case 4: events are concurrent, and own id is larger, return true
			if client.Id > fromId {
				ch <- true
				system.Println(client.Id, "is greater than", fromId, "<-- IDs. So I am sending response right away")
				break
			}
			// systemresponse.Println("Waiting to send response...")
			continue
		} else {
			// systemresponse.Println("Non Concurrent events. Local:", client.RequestTimeStamp, "Remote:", remoteTime)
			// case 1: vector clock of self is earlier: don't return true
			// case 2: vector clock of self is later: return true
			if client.RequestTimeStamp > remoteTime { // if my own vector clock is larger, return true
				ch <- true
				systemresponse.Println(client.RequestTimeStamp, "is greater than", remoteTime, "<-- Scalar Clocks. So I am sending response right away")
				break
			} 
			continue
		}
	}
}

func (client *Client) ReleaseSender(value int) {
	system.Println(client.Id, "is releasing...")
	entry := heap.Pop(&client.PQ).(*PQEntry) // This should be your own
	system.Println(entry, "popped")
	client.RequestTimeStamp = INF
	responseList := 0
	release := Message{Type: RELEASE, From: client.Id, UpdatedValue: client.CS}
	for id, IP := range client.Clientlist{
		if id == client.Id {continue} // if own id then continue
		responseChannel := make(chan Message)
		go client.CallRPC(release, IP, responseChannel)
		reply := <- responseChannel
		if (reply.Type == ACK) { responseList +=1 }
	}
	for {
		if (responseList == len(client.Clientlist)-1){
			systemtimer.Println("**********\nTime elapsed:", time.Since(Timer), "\n**********")
			break
		}
	}
}