package simpleblscosi

import (
	"testing"
	"time"

	"github.com/dedis/student_19_nyleCtrlPlane/transaction"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/util/random"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/kyber/v3/sign/bls"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/protobuf"
)

const protoName = "testProtocol"

var testSuite = pairing.NewSuiteBn256()

func testProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	vf := func(a []byte, id onet.TreeID) error { return nil }

	// Add random data to pass the test
	atomCoin := []int32{0}
	coinToAtomMap := make(map[string]int)
	treeID := onet.TreeID{1}
	mapDist := map[string]map[string]float64{}

	return NewProtocol(n, vf, treeID, atomCoin, coinToAtomMap, mapDist, testSuite)
}

func init() {
	if _, err := onet.GlobalProtocolRegister(protoName, testProtocol); err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestCosi(t *testing.T) {
	for _, nbrHosts := range []int{4, 7} {
		log.LLvl2("Running cosi with", nbrHosts, "hosts")
		local := onet.NewLocalTest(testSuite)
		_, el, tree := local.GenBigTree(nbrHosts, nbrHosts, 3, true)
		aggPublic := testSuite.Point().Null()
		for _, e := range el.List {
			aggPublic = aggPublic.Add(aggPublic, e.Public)
		}

		// create the message we want to sign for this round
		//msg := []byte("Hello World Cosi")

		// Try to make the test pass
		PvK0, PbK0 := bls.NewKeyPair(testSuite, random.New())
		_, PbK1 := bls.NewKeyPair(testSuite, random.New())

		PubK0, _ := PbK0.MarshalBinary()
		PubK1, _ := PbK1.MarshalBinary()
		iD0 := []byte("Genesis0")
		coinID := []byte("0")

		// First transaction
		inner := transaction.InnerTx{
			CoinID:     coinID,
			PreviousTx: iD0,
			SenderPK:   PubK0,
			ReceiverPK: PubK1,
		}
		innerEncoded, _ := protobuf.Encode(&inner)
		signature, _ := bls.Sign(testSuite, PvK0, innerEncoded)
		tx := transaction.Tx{
			Inner:     inner,
			Signature: signature,
		}
		txEncoded, _ := protobuf.Encode(&tx)

		// Register the function generating the protocol instance
		var root *SimpleBLSCoSi

		// Start the protocol
		p, err := local.CreateProtocol(protoName, tree)
		if err != nil {
			t.Fatal("Couldn't create new node:", err)
		}
		root = p.(*SimpleBLSCoSi)
		root.Message = txEncoded
		go func() {
			err := root.Start()
			require.NoError(t, err)
		}()
		select {
		case sig := <-root.FinalSignature:
			log.LLvlf3("Error OUT? ")
			require.NoError(t, bls.Verify(testSuite, aggPublic, txEncoded, sig))
		case <-time.After(time.Second * 10):
			t.Fatal("Could not get signature verification done in time")
		}

		local.CloseAll()
	}
}
