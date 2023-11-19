package client

import "net/rpc"

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

//  Node utility function to call RPC given a request message, and a destination IP address
func (client *Client) CallRPC(msg Message, IP string, ch chan <- Message) {
	// system.Println(client.Id, "is sending message of type", msg.Type, "to", IP)
	clnt, err := rpc.Dial("tcp", IP)
	msgCopy := msg
	reply := Message{}
	if err != nil {
		system.Println("Error Dialing RPC:", err)
		ch <- reply
		return 
	}
	err = clnt.Call("Client.HandleIncomingMessage", msgCopy, &reply)
	if err != nil {
		system.Println("Faced an error trying to call RPC:", err)
		ch <- reply
		return 
	}
	// system.Println("Received reply", reply, "from", IP)
	ch <- reply
}

func (client *Client) UpdateLocalClock(){
	client.LocalClock ++
}

func isQuorumReached(responselist []int, clientlist map[int]string) bool {
    sum := 0
    for _, response := range responselist {
        sum += response
    }
	if len(clientlist)%2 == 0{
		return sum >= len(clientlist)/2
	}
    return sum > len(clientlist)/2
}

func (client *Client) ShowPriorityQueue() []int{
	var pq []int
	for _, entry := range client.PQ.entries{pq = append(pq, entry.ID)}
	return pq
}

func (client *Client) Showclients() map[int]string{
	return client.Clientlist
}

func (client *Client) ShowCSValue() int{
	return client.CS
}