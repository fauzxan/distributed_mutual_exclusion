package client

import (
	"net/rpc"
	"sync"
)

const BEFORE = "before"
const AFTER = "after"
const EQUAL = "equal"

/*
***************************
UTILITY FUNCTIONS
***************************
*/
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

// Node utility function to call RPC given a request message, and a destination IP address
func (client *Client) CallRPC(msg Message, IP string, ch chan<- Message) {
	// system.Println(client.Id, "is sending message of type", msg.Type, "to", IP)
	clnt, err := rpc.Dial("tcp", IP)
	msgCopy := msg
	reply := Message{}
	if err != nil {
		system.Println("Error Dialing RPC:", err)
		system.Println("Received reply", reply)
		ch <- reply
		return
	}
	err = clnt.Call("Client.HandleIncomingMessage", msgCopy, &reply)
	if err != nil {
		system.Println("Faced an error trying to call RPC:", err)
		system.Println("Received reply", reply)
		ch <- reply
		return
	}
	// system.Println("Received reply", reply, "from", IP)
	ch <- reply
}

func (client *Client) UpdateLocalClock() {
	client.LocalClock++
}

func (client *Client) IsSmaller(mine []int, other []int) bool {
	for index := range mine {
		if other[index] < mine[index] {
			return false
		} // is there any element in mine that is greater than other's element? then return false as mine is not smaller.
	}
	return true
}

// func allOnes(arr []int) bool {
// 	sum := 0
// 	for _, element := range arr {
// 		sum += element
// 	}
// 	return sum == len(arr)
// }

func (client *Client) ShowPriorityQueue() []int {
	var pq []int
	for _, entry := range client.PQ {
		pq = append(pq, entry.ID)
	}
	system.Println(client.PQ)
	return pq
}

func (client *Client) Showclients() map[int]string {
	return client.Clientlist
}

func (client *Client) ShowCSValue() int {
	return client.CS
}

func printListOfResponses(client *Client) {
	client.listOfResponses.Range(func(key, value interface{}) bool {
		systemmodify.Println("Key:", key, "Value:", value)
		return true
	})
}

func countEntries(m *sync.Map) int {
	count := 0
	m.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

func clearMap(m *sync.Map) {
	m.Range(func(key, value interface{}) bool {
		m.Delete(key)
		return true
	})
}