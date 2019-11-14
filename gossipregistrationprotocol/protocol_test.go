package gossipregistrationprotocol

/*
The test-file should at the very least run the protocol for a varying number
of nodes. It is even better practice to test the different methods of the
protocol, as in Test Driven Development.
*/

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

var tSuite = suites.MustFind("Ed25519")

func TestMain(m *testing.M) {
	log.MainTest(m)
}

// Tests a 2, 5 and 13-node system. It is good practice to test different
// sizes of trees to make sure your protocol is stable.
func TestGossip(t *testing.T) {
	addSigner := func(Announce) error {
		return nil
	}
	_, err := onet.GlobalProtocolRegister(Name, NewGossipProtocol(addSigner))
	if err != nil {
		panic(err)
	}

	nodes := []int{2, 5, 13}
	for _, nbrNodes := range nodes {
		local := onet.NewLocalTest(tSuite)
		hosts, _, tree := local.GenTree(nbrNodes, true)
		ann := Announce{
			Signer: hosts[0].ServerIdentity.ID,
			Proof:  &SignatureResponse{},
			Epoch:  2,
		}

		pi, err := local.CreateProtocol(Name, tree)
		require.Nil(t, err)

		protocol := pi.(*GossipRegistationProtocol)
		protocol.Msg = ann
		require.NoError(t, protocol.Start())

		timeout := network.WaitRetry * time.Duration(network.MaxRetryConnect*nbrNodes*2) * time.Millisecond
		select {
		case sum := <-protocol.ConfirmationsChan:
			require.Equal(t, nbrNodes, sum, "The number of confirmations is not the number of nodes")

		case <-time.After(timeout):
			t.Fatal("Didn't finish in time")
		}
		local.CloseAll()
	}
}
