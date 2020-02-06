package membershipchainservice

import (
	"bytes"
	"errors"
	"math/rand"
	"sort"

	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

// GetEpoch returns the current epoch
func (s *Service) GetEpoch() Epoch {
	return s.e
}

// GetSigners gives the registrations that are stored on this node
func (s *Service) GetSigners(e Epoch) *SignersReply {
	s.storage.Lock()
	defer s.storage.Unlock()
	if e < 0 || e >= Epoch(len(s.storage.Signers)) {
		return &SignersReply{Set: nil}
	}
	return &SignersReply{Set: s.storage.Signers[e]}

}

func getKeys(m SignersSet) []network.ServerIdentityID {
	var keys []network.ServerIdentityID
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		aB := [16]byte(keys[a])
		bB := [16]byte(keys[b])
		return bytes.Compare(aB[:], bB[:]) < 0
	})
	return keys
}
func (s *Service) getGlobalRoster() *onet.Roster {
	sis := []*network.ServerIdentity{}
	s.ServersMtx.Lock()
	for _, v := range s.Servers {
		sis = append(sis, v)
	}
	s.ServersMtx.Unlock()

	return onet.NewRoster(sis)
}

func (s *Service) getRosterForEpoch(e Epoch) (*onet.Roster, error) {
	s.storage.Lock()
	mbrs, err := s.getServerIdentityFromSignersSet(s.storage.Signers[s.e])
	if err != nil {
		defer s.storage.Unlock()
		return nil, err
	}
	if _, ok := s.storage.Signers[s.e][s.ServerIdentity().ID]; !ok {
		defer s.storage.Unlock()
		return nil, errors.New("One node cannot start a new Epoch if it didn't registrate")
	}
	s.storage.Unlock()

	return onet.NewRoster(mbrs), nil

}

func (s *Service) getServerIdentityFromSignersSet(m SignersSet) ([]*network.ServerIdentity, error) {
	mbrsIDs := getKeys(m)
	var mbrs []*network.ServerIdentity
	ro := s.getGlobalRoster()
	for _, mID := range mbrsIDs {
		_, si := ro.Search(mID)
		if si == nil {
			return nil, errors.New("Server Identity not found in Roster")
		}
		mbrs = append(mbrs, si)
	}
	return mbrs, nil
}

// GetNamesFromSignerSet return name from signers set
func (s *Service) GetNamesFromSignerSet(m SignersSet) ([]string, error) {
	mbrsIDs := getKeys(m)
	var mbrs []string
	for _, mID := range mbrsIDs {
		name, ok := s.ServerIdentityToName[mID]
		if !ok {
			return nil, errors.New("Server Identity not found in ServerIdentityToName")
		}
		mbrs = append(mbrs, name)
	}
	return mbrs, nil
}

func (s *Service) getServers() map[string]*network.ServerIdentity {
	s.ServersMtx.Lock()
	dst := make(map[string]*network.ServerIdentity, len(s.Servers))

	for k, v := range s.Servers {
		dst[k] = v
	}
	s.ServersMtx.Unlock()
	return dst
}

func (s *Service) getepochOfEntryMap() map[string]Epoch {
	temp := make(map[network.ServerIdentityID]Epoch)
	s.storage.Lock()
	for i := len(s.storage.Signers) - 1; i >= 0; i-- {
		for id := range s.storage.Signers[i] {
			temp[id] = Epoch(i)
		}
	}
	s.storage.Unlock()

	ret := make(map[string]Epoch)
	s.ServersMtx.Lock()
	for id, e := range temp {
		ret[s.ServerIdentityToName[id]] = e
	}
	s.ServersMtx.Unlock()
	return ret
}

// GetRandomName return a random Name from the list of Servers
func (s *Service) GetRandomName() string {
	var names []string
	s.ServersMtx.Lock()
	for name := range s.Servers {
		if name != s.Name {
			names = append(names, name)
		}
	}
	log.LLvl1(s.Name, "has a this list ", names, " of random names.")

	s.ServersMtx.Unlock()
	index := rand.Intn(len(names))
	return names[index]
}
