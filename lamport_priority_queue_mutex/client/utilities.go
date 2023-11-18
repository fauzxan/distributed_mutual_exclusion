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
	system.Println(client.Id, "is sending message of type", msg.Type, "to", IP)
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
	system.Println("Received reply", reply, "from", IP)
	ch <- reply
}

func (client *Client) UpdateLocalClock(){
	client.LocalClock ++
	// client.Vectorclock[client.Id] = client.LocalClock
}

/*
	Input:
		V1 is the incoming vector.
		V2 is the vector on the local process. 

	This function receives two vectors- one from the message that just came in, and one from the local process.
	It will compare all the elements in one vector to all the elements in the other vector, and:
	1. See if there are any causality violations
		If any element in V2 is greater than any element in V2, then there is a causality violation. In this case return []. The receiver will receive this and see that there
		is a causality violation, and flag it to the terminal. It will still continue running, but there is a potential causality violation. 
	2. Update each element in V1 and V2 as max(V1.1, V2.1), max(V1.2, V2.2) and so on

	Finally, this function should return the updated V2, which will be places into the receiver process
*/
func (client *Client) UpdateVectorClock(V1 []int, V2 []int, from int) []int{

	for i := range V1{
		if (V2[i] > V1[i] && i == from){
			// there is a potential causality violation
			system.Println("There is a potential causality violation!!")
		}
		V2[i] = max(V2[i], V1[i])
	}
	return V2
}

func (client *Client) IsSmaller(mine []int, other []int) bool{
	for index := range mine{
		if (other[index] < mine[index]){ return false } // is there any element in mine that is greater than other's element? then return false as mine is not smaller. 
	}
	return true
}

// func (client *Client) CheckIfInPQ() bool{
// 	for _, entry := range client.PQ {
// 		if entry.pid == client.Id {return true}
// 	}
// 	return false
// }

// Go through pq, and see if your own entry is before or after the parameter
// func (client *Client) CheckPos(other int) string{
// 	ownIndex := 0
// 	otherIndex := 0
// 	for index, node := range client.PQ{
// 		 if node.Id == other {otherIndex = index}
// 		 if node == client {ownIndex = index}
// 	}
// 	if ownIndex < otherIndex {return BEFORE} else
// 	if ownIndex > otherIndex {return AFTER}
// 	return ""
// }

// func (client *Client) ShowClock() []int{
// 	return client.Vectorclock
// }

func (client *Client) ShowPriorityQueue() []int{
	var pq []int
	for _, entry := range client.PQ{pq = append(pq, entry.ID)}
	system.Println(client.PQ)
	return pq
}

func (client *Client) Showclients() map[int]string{
	return client.Clientlist
}

func (client *Client) ShowCSValue() int{
	return client.CS
}