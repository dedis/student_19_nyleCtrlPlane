package main

import (
	// Service needs to be imported here to be instantiated.
	_ "github.com/dedis/student_19_nyleCtrlPlane/membershipchainservice"

	"go.dedis.ch/onet/v3/simul"
)

func main() {
	simul.Start()
}
