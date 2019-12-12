package nylechain

import (
	mbrSer "github.com/dedis/student_19_nyleCtrlPlane/membershipchainservice"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
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
	log.Lvl1("called", dst.String())

	serviceReq := &mbrSer.SetGenesisSignersRequest{
		Servers: servers,
	}

	log.LLvl1("Sending genesis message to", dst)
	reply := &mbrSer.SetGenesisSignersReply{}
	err := c.SendProtobuf(dst, serviceReq, reply)
	log.Lvl1(err, "Done with init request")
	if err != nil {
		return nil, err
	}
	return reply, nil

}

// ExecEpochRequest sends a message to a service to set genesis Request
func (c *Client) ExecEpochRequest(dst *network.ServerIdentity, e mbrSer.Epoch) (*mbrSer.ExecEpochReply, error) {
	log.Lvl1("called", dst.String())

	serviceReq := &mbrSer.ExecEpochRequest{
		Epoch: e,
	}

	log.LLvl1("Sending exec message to", dst)
	reply := &mbrSer.ExecEpochReply{}
	err := c.SendProtobuf(dst, serviceReq, reply)
	log.Lvl1(err, "Done with init request")
	if err != nil {
		return nil, err
	}
	return reply, nil

}
