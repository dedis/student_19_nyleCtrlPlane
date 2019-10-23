package nylechain

import (
	"testing"

	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3/log"
)

var testSuite = pairing.NewSuiteBn256()

func TestMain(m *testing.M) {
	log.MainTest(m)
}
