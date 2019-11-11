package nylechain

import (
	"github.com/dedis/student_19_nyleCtrlPlane/membershipchainservice"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/onet/v3"
)

// Client is a structure to communicate with the template
// service
type Client struct {
	*onet.Client
}

// NewClient instantiates a new template.Client
func NewClient() *Client {
	return &Client{Client: onet.NewClient(cothority.Suite, membershipchainservice.ServiceName)}
}
