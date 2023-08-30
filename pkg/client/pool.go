package client

import "sync"

var ManagerClientGlobal *ManagerClient = NewManagerClient()

type ManagerClient struct {
	clients *sync.Map
}

func NewManagerClient() *ManagerClient {
	return &ManagerClient{
		clients: new(sync.Map),
	}
}

func (c *ManagerClient) GetClient(endPoint string) *Client {
	val, existed := c.clients.Load(endPoint)
	if existed {
		return val.(*Client)
	}
	client, _ := NewClient(endPoint)
	val, _ = c.clients.LoadOrStore(endPoint, client)
	return val.(*Client)
}
