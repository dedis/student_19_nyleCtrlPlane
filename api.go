package nylechain

import (
	"time"

	mbrSer "github.com/dedis/student_19_nyleCtrlPlane/membershipchainservice"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
)

// Client is a structure to communicate with the template
// service
type Client struct {
	*onet.Client
}

// NewClient instantiates a new template.Client
func NewClient() *Client {
	return &Client{Client: onet.NewClient(cothority.Suite, mbrSer.ServiceName)}
}

// SetGenesisSignersRequest sends a message to a service to set genesis Request
func (c *Client) SetGenesisSignersRequest(dst *network.ServerIdentity, servers map[*network.ServerIdentity]string) (*mbrSer.SetGenesisSignersReply, error) {
	serviceReq := &mbrSer.SetGenesisSignersRequest{
		Servers: servers,
	}
	reply := &mbrSer.SetGenesisSignersReply{}
	err := c.SendProtobuf(dst, serviceReq, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func (c *Client) SetRegistrationDuration(dst *network.ServerIdentity, dur time.Duration) (*mbrSer.SetDurationReply, error) {
	serviceReq := &mbrSer.SetDurationRequest{
		Duration: dur,
	}
	reply := &mbrSer.SetDurationReply{}
	err := c.SendProtobuf(dst, serviceReq, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil

}

// ExecEpochRequest sends a message to a service to set genesis Request
func (c *Client) ExecEpochRequest(dst *network.ServerIdentity, e mbrSer.Epoch) (*mbrSer.ExecEpochReply, error) {
	serviceReq := &mbrSer.ExecEpochRequest{
		Epoch: e,
	}
	reply := &mbrSer.ExecEpochReply{}
	err := c.SendProtobuf(dst, serviceReq, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

// ExecWriteSigners sends a message to a service to write its signers to a file
func (c *Client) ExecWriteSigners(dst *network.ServerIdentity, e mbrSer.Epoch) (*mbrSer.ExecWriteSignersReply, error) {
	serviceReq := &mbrSer.ExecWriteSigners{
		Epoch: e,
	}
	reply := &mbrSer.ExecWriteSignersReply{}
	err := c.SendProtobuf(dst, serviceReq, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

// UpdateForNewNode send a request to update for a new node
func (c *Client) UpdateForNewNode(dst *network.ServerIdentity, e mbrSer.Epoch) (*mbrSer.UpdateForNewNodeReply, error) {
	serviceReq := &mbrSer.UpdateForNewNodeRequest{
		Epoch: e,
	}
	reply := &mbrSer.UpdateForNewNodeReply{}
	err := c.SendProtobuf(dst, serviceReq, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

// UpdateNode send a request to update for a new node
func (c *Client) UpdateNode(dst *network.ServerIdentity) (*mbrSer.UpdateReply, error) {
	serviceReq := &mbrSer.UpdateRequest{}
	reply := &mbrSer.UpdateReply{}
	err := c.SendProtobuf(dst, serviceReq, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

// CreateProofForEpochRequest send a request to Create Proof
func (c *Client) CreateProofForEpochRequest(dst *network.ServerIdentity, e mbrSer.Epoch) (*mbrSer.CreateProofForEpochReply, error) {
	serviceReq := &mbrSer.CreateProofForEpochRequest{
		Epoch: e,
	}
	reply := &mbrSer.CreateProofForEpochReply{}
	err := c.SendProtobuf(dst, serviceReq, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

// StartNewEpochRequest send a request to Start New Epoch
func (c *Client) StartNewEpochRequest(dst *network.ServerIdentity) (*mbrSer.StartNewEpochReply, error) {
	serviceReq := &mbrSer.StartNewEpochRequest{}
	reply := &mbrSer.StartNewEpochReply{}
	err := c.SendProtobuf(dst, serviceReq, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

// GetConsencusOnNewSignersRequest send a request to get consencus
func (c *Client) GetConsencusOnNewSignersRequest(dst *network.ServerIdentity) (*mbrSer.GetConsencusOnNewSignersReply, error) {
	serviceReq := &mbrSer.GetConsencusOnNewSignersRequest{}
	reply := &mbrSer.GetConsencusOnNewSignersReply{}
	err := c.SendProtobuf(dst, serviceReq, reply)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

// SendPause send a request to make sleep a service
func (c *Client) SendPause(dst *network.ServerIdentity) {
	serviceReq := &mbrSer.SendPauseRequest{}
	reply := &mbrSer.SendPauseReply{}
	c.SendProtobuf(dst, serviceReq, reply)
}
