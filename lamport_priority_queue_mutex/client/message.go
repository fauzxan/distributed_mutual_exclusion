package client


type Message struct{
	Type string // REQUEST | RELEASE | RESPONSE | JOIN
	From int // Nodeid of the sender
	UpdatedValue int // Nodeid of the request who you are approving for
	Vectorclock []int // My own vectorclock
	Responselist []int // Nodeid  
	IP string // IP address of new joiner
	Clientlist map[int]string
	RemoteTimestamp int
}