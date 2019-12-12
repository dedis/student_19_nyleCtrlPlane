package nylechain

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	mbrSer "github.com/dedis/student_19_nyleCtrlPlane/membershipchainservice"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

var testSuite = pairing.NewSuiteBn256()

func TestMain(m *testing.M) {
	log.MainTest(m)
}
func TestFewEpochs(t *testing.T) {
	c := NewClient()
	local := onet.NewTCPTest(testSuite)

	nbrNodes := 10
	hosts, _, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()

	services := local.GetServices(hosts, mbrSer.MembershipID)
	for i, s := range services {
		s.(*mbrSer.Service).Name = "node_" + strconv.Itoa(i)
		s.(*mbrSer.Service).PrefixForReadingFile = "."
	}

	servers := make(map[*network.ServerIdentity]string)

	for i := 0; i < 2; i++ {
		servers[hosts[i].ServerIdentity] = services[i].(*mbrSer.Service).Name
	}
	log.LLvl1(servers)

	var wg sync.WaitGroup
	for _, h := range hosts {
		wg.Add(1)
		_, err := c.SetGenesisSignersRequest(h.ServerIdentity, servers)
		assert.Nil(t, err)
		wg.Done()
	}
	wg.Wait()

	for e := mbrSer.Epoch(1); e < mbrSer.Epoch(4); e++ {
		for _, h := range hosts {
			wg.Add(1)
			go func(si *network.ServerIdentity, ee mbrSer.Epoch) {
				_, err := c.ExecEpochRequest(si, ee)
				assert.Nil(t, err)
				wg.Done()
			}(h.ServerIdentity, e)
		}
		wg.Wait()
	}
}
