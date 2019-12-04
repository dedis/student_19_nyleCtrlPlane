package gossipregistrationprotocol

/*
The test-file should at the very least run the protocol for a varying number
of nodes. It is even better practice to test the different methods of the
protocol, as in Test Driven Development.
*/

import (
	"sync"
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
	addSigner := func(Announce) error {
		return nil
	}
	_, err := onet.GlobalProtocolRegister(Name, NewGossipProtocol(addSigner))
	if err != nil {
		panic(err)
	}
	log.MainTest(m)
}

// Tests a 2, 5 and 13-node system. It is good practice to test different
// sizes of trees to make sure your protocol is stable.
func TestGossip(t *testing.T) {

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
func TestGossipWithGoRoutine(t *testing.T) {

	nodes := []int{2, 5, 13}
	for _, nbrNodes := range nodes {
		local := onet.NewLocalTest(tSuite)
		hosts, _, tree := local.GenTree(nbrNodes, true)
		ann := Announce{
			Signer: hosts[0].ServerIdentity.ID,
			Proof:  &SignatureResponse{},
			Epoch:  2,
		}

		var wg sync.WaitGroup
		wg.Add(1)
		for j := 0; j < 1; j++ {
			go func(l *onet.LocalTest) {
				pi, err := l.CreateProtocol(Name, tree)
				require.Nil(t, err)

				protocol := pi.(*GossipRegistationProtocol)
				protocol.Msg = ann
				require.NoError(t, protocol.Start())

				timeout := network.WaitRetry * time.Duration(network.MaxRetryConnect*nbrNodes*2) * time.Millisecond
				select {
				case sum := <-protocol.ConfirmationsChan:
					require.Equal(t, nbrNodes, sum, "The number of confirmations is not the number of nodes")
					wg.Done()

				case <-time.After(timeout):
					t.Fatal("Didn't finish in time")
				}
			}(local)
		}

		wg.Wait()
		local.CloseAll()
	}
}

func TestWithFlatTree(t *testing.T) {
	for i := 5; i < 50; i += 5 {
		local := onet.NewLocalTest(tSuite)
		hosts, roster, _ := local.GenTree(i, true)
		ann := Announce{
			Signer: hosts[0].ServerIdentity.ID,
			Proof:  &SignatureResponse{},
			Epoch:  2,
		}

		tree := roster.GenerateNaryTree(i)
		pi, err := local.CreateProtocol(Name, tree)
		require.Nil(t, err)

		protocol := pi.(*GossipRegistationProtocol)
		protocol.Msg = ann
		require.NoError(t, protocol.Start())

		timeout := network.WaitRetry * time.Duration(network.MaxRetryConnect*i*2) * time.Millisecond
		select {
		case sum := <-protocol.ConfirmationsChan:
			require.Equal(t, i, sum, "The number of confirmations is not the number of nodes")

		case <-time.After(timeout):
			t.Fatal("Didn't finish in time")
		}
		local.CloseAll()
	}

}
func TestWithParallelCommincation(t *testing.T) {
	nbrNodes := 5
	local := onet.NewLocalTest(tSuite)
	hosts, roster, _ := local.GenTree(nbrNodes, true)
	ann := Announce{
		Signer: hosts[0].ServerIdentity.ID,
		Proof:  &SignatureResponse{},
		Epoch:  2,
	}

	var wg sync.WaitGroup
	for j := 0; j < 1; j++ {
		wg.Add(1)
		go func(i int, l *onet.LocalTest) {
			//tree := roster.GenerateNaryTreeWithRoot(nbrNodes-1, hosts[i].ServerIdentity)
			tree := roster.NewRosterWithRoot(hosts[i].ServerIdentity).GenerateBinaryTree()
			//pi, err := l.CreateProtocol(Name+strconv.Itoa(i), tree)
			pi, err := l.CreateProtocol(Name, tree)
			require.Nil(t, err)

			protocol := pi.(*GossipRegistationProtocol)
			protocol.Msg = ann
			err = protocol.Start()
			log.LLvl1("ERROR : ", err)
			require.Nil(t, err)

			timeout := 10 * time.Second
			log.LLvl1("Before select")
			select {
			case sum := <-protocol.ConfirmationsChan:
				log.LLvl1("IN ANSWER")
				wg.Done()
				require.Equal(t, 5, sum, "The number of confirmations is not the number of nodes")

			case <-time.After(timeout):
				log.LLvl1("Time out")
				t.Fatal("Didn't finish in time")
			}

		}(j, local)
	}
	log.LLvl1("After for")
	wg.Wait()
	local.CloseAll()
}
