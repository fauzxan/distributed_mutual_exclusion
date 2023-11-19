package client

import (
	// "encoding/json"
	// "os"
	"container/heap"
	"math"
	// "fmt"
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
	Lock           		sync.Mutex
	Id             		int
	Clientlist     		map[int]string
	CS 			   		int
	// Vectorclock	   	[]int
	LocalClock	   		int 
	PQ 			   		PriorityQueue
	RequestTimeStamp 	int
	listOfResponses 	[]int
}

const INF = math.MaxInt16

// Message types
const (
    REQUEST  = "request"
    RESPONSE = "response"
    RELEASE  = "release"
    JOIN     = "join"
    ACK      = "ack"
	RESCIND	 = "rescind"
)


// Timer variable for timing elections:
var Timer = time.Now()
var PQ PriorityQueue


// Handle incoming messages from RPC channel
func (client *Client) HandleIncomingMessage(msg Message, reply *Message) error { 
	// Update clock for receive event
	client.UpdateLocalClock()
	
	switch msg.Type{
		case REQUEST:
			// call the function to approve requests
			heap.Push(&client.PQ, &PQEntry{ID: msg.From, Timestamp: msg.RemoteTimestamp})
			go client.ResponseHandler(msg.Vectorclock, msg.From, msg.RemoteTimestamp)
			systemresponse.Println("Added to the local queue, response will be handled accordingly")
			reply.Type = ACK + REQUEST
		case RELEASE:
			systemmodify.Println("Updating local value to", msg.UpdatedValue, "as set by", msg.From)
			client.CS = msg.UpdatedValue
			res := client.PQ.Remove(msg.From) // May not be at the head of the queue, so we implement Remove in priorityqueue.go
			if res {systemmodify.Println("Removed", msg.From, "from PQ")} else {systemmodify.Println("Did not remove", msg.From, "from PQ")}
			reply.Type = ACK + RELEASE
		case RESCIND:
			if client.RequestTimeStamp == INF {break} // If I don't have any requests, I don't need to modify the listOfResponses.
			client.listOfResponses[msg.From] = 0
			systemmodify.Println("Received a rescind. Response list:", client.listOfResponses)
			reply.Type = ACK + RESCIND
		case RESPONSE:
			if client.RequestTimeStamp == INF {break} // If I don't have any requests, I don't need to modify the listOfResponses.
			client.listOfResponses[msg.From] = 1
			systemmodify.Println("Received a response. Response list:", client.listOfResponses)
			reply.Type = ACK + RESPONSE
		case JOIN:
			client.Clientlist[msg.From] = msg.IP
			client.listOfResponses = make([]int, len(client.listOfResponses)+1)
			// client.Vectorclock = append(client.Vectorclock, 0)
			reply.Type = ACK + JOIN
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
		if reply.Type == ACK + JOIN {
			system.Println("Client", id, "has been notified")
		}
	}
	client.PQ = PriorityQueue{mu: sync.Mutex{}, entries: make([]*PQEntry, 0)} 
	client.RequestTimeStamp = INF
	client.listOfResponses = make([]int, len(client.Clientlist))
}


// Should be a go routine, because the priority queue should be able to get modified when this is occuring. 
func (client *Client) ModifyCriticalSection(){
	for {
		timeNow := time.Now()
		if timeNow.Second() % 10 == 0 {break}
	} // Even if I press one at slightly different times, they will all send concurrently at the same time.

	// Update the time at which you are making a request
	Timer = time.Now()
	client.UpdateLocalClock()
	client.RequestTimeStamp = client.LocalClock
	systemmodify.Println("Request made at timestamp", client.RequestTimeStamp)
	// client.listOfResponses[client.Id] = 1

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

	// client.listOfResponses will get updated asynchronously within HandleIncomingMessages. You will need to block until majority votes have been received. 
    quorumReached := false
	for !quorumReached{
		quorumReached = isQuorumReached(client.listOfResponses, client.Clientlist)
	}
	
	
	// randomize some input and carry out the CS modification
	systemmodify.Println("Quorum reached. Modifying CS.")
	var input int
	time.Sleep(time.Second * 1)
	systemmodify.Println("Enter value")
	// fmt.Scanln(&input) // <- this might be too slow when trying to compare the time, so just randomize it!
	input = rand.Intn(1000)
	systemmodify.Println("Entered", input)
	systemmodify.Println("Setting CS...")
	client.CS = input


	// Once done, you will need to send a release lock message to everyone in the clientlist.
	go client.ReleaseSender(input)
}


func (client *Client) ResponseHandler(from []int, fromId int, remoteTime int) {
    isPriorityQueueEmpty := false
    for !isPriorityQueueEmpty {
        // if client.RequestTimeStamp < remoteTime {
        //     continue // Don't need to send a response yet if your timestamp is less than remoteTime
        // }
        count := 0

        // Get a copy of the priority queue entries
        entriesCopy := client.PQ.GetEntries()
		if len(entriesCopy) == 0 {
            isPriorityQueueEmpty = true
			break
        }
		

        // send rescind to everyone but the top entry in your PQ
        for _, PQEntry := range entriesCopy {
            if count == 0 {
                // don't send rescind to the first element in the queue
                count++
                continue
            }
            msg := Message{Type: RESCIND, From: client.Id}
            go client.CallRPC(msg, client.Clientlist[PQEntry.ID], make(chan<- Message))
        }

        // send a response to the first entry in your PQ
        msg := Message{Type: RESPONSE, From: client.Id}
        go client.CallRPC(msg, client.Clientlist[entriesCopy[0].ID], make(chan<- Message))
        time.Sleep(1500 * time.Millisecond)
    }
}


func (client *Client) ReleaseSender(value int) {
	systemmodify.Println(client.Id, "is releasing...")
	client.RequestTimeStamp = INF
	responseList := 0
	release := Message{Type: RELEASE, From: client.Id, UpdatedValue: client.CS}
	for id, IP := range client.Clientlist{
		if id == client.Id {continue} // if own id then continue
		responseChannel := make(chan Message)
		go client.CallRPC(release, IP, responseChannel)
		reply := <- responseChannel
		if (reply.Type == (ACK + RELEASE)) { responseList +=1 }
	}
	client.listOfResponses = make([]int, len(client.Clientlist))
	if (responseList == len(client.Clientlist)-1){
		systemtimer.Println("**********\nTime elapsed:", time.Since(Timer), "\n**********")
	}
}