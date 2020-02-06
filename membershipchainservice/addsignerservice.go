package membershipchainservice

import (
	"errors"
	"fmt"

	gpr "github.com/dedis/student_19_nyleCtrlPlane/gossipregistrationprotocol"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

func (s *Service) addSignerFromMessage(ann gpr.Announce) error {
	s.InteractionMtx.Lock()
	s.CountInteractions[s.GetEpoch()][ann.Name]++
	s.InteractionMtx.Unlock()

	s.ServersMtx.Lock()
	s.Servers[ann.Name] = ann.Server
	s.ServerIdentityToName[ann.Signer] = ann.Name
	s.ServersMtx.Unlock()
	return s.addSigner(ann.Signer, ann.Proof, ann.Epoch)
}

// addSigner will add one signer to the storage if the proof is convincing
func (s *Service) addSigner(signer network.ServerIdentityID, proof *gpr.SignatureResponse, e int) error {
	if proof != nil {
		if e < 0 {
			return errors.New("Epoch cannot be negative")
		}
		s.storage.Lock()

		if e > len(s.storage.Signers) {
			log.LLvl1(" Error in add signer ? ")
			return errors.New("Epoch is too in the future")
		}

		if e == len(s.storage.Signers) {
			s.storage.Signers = append(s.storage.Signers, make(SignersSet))
		}

		if s.e > Epoch(e) {
			return errors.New(" Error in add signer - Cannot sign for previous epochs ")
		}
		if s.Cycle.GetTimeTillNextEpoch() < TIME_FOR_CONSENCUS || s.Cycle.GetEpoch() >= Epoch(e) {
			return errors.New(" Error in add signer - Cannot sign for previous epochs ")
		}

		s.storage.Signers[Epoch(e)][signer] = *proof
		s.storage.Unlock()
		return nil
	}
	return fmt.Errorf("Addsigner cannot be completed for %v as %v did not send a signature", s.Name, signer)

}
