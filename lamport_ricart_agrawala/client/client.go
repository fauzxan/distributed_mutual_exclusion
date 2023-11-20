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
	Lock       sync.Mutex
	Id         int
	Clientlist map[int]string
	CS         int
	// Vectorclock	   []int
	LocalClock       int
	PQ               PriorityQueue
	RequestTimeStamp *int
	listOfResponses  sync.Map
}

// Flag variables:
var Higherid = false

const INF = math.MaxInt16

// Message types
const (
	REQUEST  = "request"
	RESPONSE = "response"
	// RELEASE  = "release"
	JOIN = "join"
	ACK  = "ack"
)

// Timer variable for timing elections:
var Timer = time.Now()
var PQ PriorityQueue

var IncomingChannels []chan Message

// Handle incoming messages from RPC channel
func (client *Client) HandleIncomingMessage(msg Message, reply *Message) error {
	// Update clock for receive event
	// client.Vectorclock = client.UpdateVectorClock(msg.Vectorclock, client.Vectorclock, msg.From)
	client.UpdateLocalClock()

	switch msg.Type {
	case REQUEST:
		// call the function to approve requests
		responseChannel := make(chan bool)
		go client.ResponseHandler(msg.Vectorclock, msg.From, msg.RemoteTimestamp, responseChannel)

		val := <-responseChannel
		if val {
			client.UpdateLocalClock()
			reply.Type = RESPONSE
			reply.From = client.Id
		} else {
			client.UpdateLocalClock()
			reply.Type = ACK
			reply.From = client.Id
		}
	case RESPONSE:
		systemmodify.Println("Received a response")
		if *client.RequestTimeStamp < INF { // Only update list of responses if you have an active request
			client.listOfResponses.Store(msg.From, 1)
			systemmodify.Println("Received a response from", msg.From, "from RPC.")
		}
		client.CS = msg.UpdatedValue
		reply.Type = ACK + RESPONSE
	case JOIN:
		client.Clientlist[msg.From] = msg.IP
		reply.Type = ACK + JOIN
	default:
		system.Println("Invalid message type?? How?") // will never reach this scenario...
	}
	return nil
}

// Enables everyone to create an entry in their clientlist, as well as their vector clocks.
func (client *Client) JoinNetwork() {
	for id, ip := range client.Clientlist {
		if id == client.Id {
			continue
		}
		message := Message{Type: JOIN, From: client.Id, IP: client.Clientlist[client.Id]}
		responseChannel := make(chan Message)
		go client.CallRPC(message, ip, responseChannel)
		reply := <-responseChannel
		if reply.Type == ACK {
			system.Println("Client", id, "has been notified")
		}
	}
	client.PQ = make(PriorityQueue, 0)
	client.RequestTimeStamp = new(int)
	*client.RequestTimeStamp = INF
	IncomingChannels = make([]chan Message, len(client.Clientlist))
}

// Should be a go routine, because the priority queue should be able to get modified when this is occuring.
func (client *Client) ModifyCriticalSection() {
	for {
		timeNow := time.Now()
		if timeNow.Second()%10 == 0 {
			break
		}
	} // Even if I press one at slightly different times, they will all send concurrently at the same time.

	// Put yourself in your own priority queue
	Timer = time.Now()
	client.UpdateLocalClock()
	*client.RequestTimeStamp = client.LocalClock
	systemmodify.Println("Adding my request to the PQ with timestamp", *client.RequestTimeStamp)
	heap.Push(&client.PQ, &PQEntry{ID: client.Id, Timestamp: *client.RequestTimeStamp})

	// Send the request to all the other nodes in the network
	req := Message{Type: REQUEST, From: client.Id, RemoteTimestamp: *client.RequestTimeStamp}
	responseChannels := make([]chan Message, len(client.Clientlist))
	for id, ip := range client.Clientlist {
		if id == client.Id {
			continue
		}
		responseChannel := make(chan Message)
		responseChannels[id] = responseChannel
		go client.CallRPC(req, ip, responseChannel)
	}

	for id, responseChannel := range responseChannels {
		if id == client.Id {
			continue
		}
		reply := <-responseChannel
		if reply.Type == RESPONSE {
			client.listOfResponses.Store(reply.From, 1)
			systemmodify.Println("Received a response from", reply.From, " through local call.")
		}
		client.UpdateLocalClock()
	}
	client.listOfResponses.Store(client.Id, 1) // You have permission from yourself.
	
	for countEntries(&client.listOfResponses) != len(client.Clientlist) { // Wait till you receive all the responses.
		time.Sleep(1500 * time.Millisecond)
	}
	system.Println("LENGTHS:", countEntries(&client.listOfResponses), len(client.Clientlist))
	printListOfResponses(client)
	
	var input int
	systemmodify.Println("I now have permission from everyone else. Response list")
	systemmodify.Println("Enter value")
	// fmt.Scanln(&input) // <- this might be too slow when trying to compare the time, so just randomize it!
	input = rand.Intn(1000)
	systemmodify.Println("Entered", input)
	systemmodify.Println("Setting CS...")
	client.CS = input

	// Once done, you will need to send a release lock message to everyone in the clientlist.
	entry := heap.Pop(&client.PQ).(*PQEntry)   // This should be your own
	system.Println(entry, "<-- popped myself") // by this time you are guaranteed to be at the head of your own queue, so you simply call Pop to remove yourself.
	clearMap(&client.listOfResponses)
	go client.SendResponse(input)
}

func (client *Client) ResponseHandler(from []int, fromId int, remoteTime int, ch chan bool) {
	// (*client.listOfResponses.listOfResponses) = make([]int, len(client.Clientlist))
	if *client.RequestTimeStamp == INF { // If I don't have any requests of my own, then send a response right away.
		system.Println("I don't have any requests of my own. I am sending response to", fromId)
		ch <- true
		return
	}

	// If the incoming request has a smaller timestamp, then I send a response right away.
	if remoteTime < *client.RequestTimeStamp {
		system.Println("I have request of my own, but it's larger. I am sending response to", fromId)
		ch <- true
	} else

	// If the incoming request has a larger timestamp, then I will put it in my Priority Queue, and wait for my request to get executed first.
	// Eventually, my own request will be removed from the priority queue, and response will be sent to everyone else.
	if remoteTime > *client.RequestTimeStamp {
		system.Println("I have request of my own, and it's larger. I am sending ACK to", fromId)
		systemresponse.Println("Adding remote request into PQ with timestamp", remoteTime)
		heap.Push(&client.PQ, &PQEntry{ID: fromId, Timestamp: remoteTime})
		ch <- false
	}

	// Concurrent events
	if remoteTime == *client.RequestTimeStamp {
		// If incoming ID smaller, then it gets higher priority
		system.Println("Concurrent events encountered")
		if fromId < client.Id {
			system.Println("Their ID is smaller. I am sending response to", fromId)
			ch <- true
		} else {
			system.Println("My ID is smaller. I am sending ACK to", fromId)
			systemresponse.Println("Adding remote request into PQ with timestamp", remoteTime)
			heap.Push(&client.PQ, &PQEntry{ID: fromId, Timestamp: remoteTime})
			ch <- false
		}
	}
}

func (client *Client) SendResponse(value int) {
	system.Println(client.Id, "is done with the CS. Sending response to everyone else...")
	*client.RequestTimeStamp = INF
	responseList := 0
	response := Message{Type: RESPONSE, From: client.Id, UpdatedValue: client.CS}
	for id, IP := range client.Clientlist {
		if id == client.Id {
			continue
		} // if own id then continue
		system.Println("Sending response to", IP)
		responseChannel := make(chan Message)
		go client.CallRPC(response, IP, responseChannel)
		reply := <-responseChannel
		if reply.Type == ACK+RESPONSE {
			responseList += 1
		}
	}
	for {
		if responseList == len(client.Clientlist)-1 {
			systemtimer.Println("**********\nTime elapsed:", time.Since(Timer), "\n**********")
			break
		}
	}
}
