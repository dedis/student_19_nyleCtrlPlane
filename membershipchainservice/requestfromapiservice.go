package membershipchainservice

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"go.dedis.ch/onet/v3/log"
)

const START_EPOCH = false

// SetGenesisSignersRequest handles requests for the function
func (s *Service) SetGenesisSignersRequest(req *SetGenesisSignersRequest) (*SetGenesisSignersReply, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	writeToFile("Function,DeltaT,startTime", dir+"/"+s.PrefixForReadingFile+"/Data/Timing/"+s.Name+".txt")
	startTime := time.Now()

	s.e = 0
	s.storage.Lock()
	s.storage.Signers = make([]SignersSet, 0)
	s.storage.Unlock()
	s.SetGenesisSigners(req.Servers)
	writeToFile(fmt.Sprintf("GenesisSignersRequest - 0 - end, %v, %v", int64(time.Now().Sub(startTime)/time.Millisecond), startTime), dir+"/"+s.PrefixForReadingFile+"/Data/Timing/"+s.Name+".txt")
	return &SetGenesisSignersReply{}, nil
}

//ExecEpochRequest handles requests for the function
func (s *Service) ExecEpochRequest(req *ExecEpochRequest) (*ExecEpochReply, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	if s.e != req.Epoch-1 {
		err = s.UpdateHistoryWith(s.GetRandomName())
		if err != nil {
			return nil, err
		}
	}
	writeToFile(fmt.Sprintf("ExecEpochRequest - 2 - AfterUpdate, %v, %v", int64(time.Now().Sub(startTime)/time.Millisecond), startTime), dir+"/"+s.PrefixForReadingFile+"/Data/Timing/"+s.Name+".txt")
	log.LLvl1("WRITING TO OS : ", dir+"/"+s.PrefixForReadingFile+"/Data/Throughput/"+s.Name+".txt,  ------------------------------------------")
	log.LLvl1("................................................................................................", dir+"/"+s.PrefixForReadingFile+"/Data/Timing/"+s.Name+".txt")
	err = s.CreateProofForEpoch(req.Epoch)
	if err != nil {
		log.LLvl3("ERROR IN CREATE PROOF : ", err)
		writeToFile(s.Name+",False,"+strconv.Itoa(int(s.e)), dir+"/"+s.PrefixForReadingFile+"/Data/Throughput/"+s.Name+".txt")
	} else {
		writeToFile(s.Name+",True,"+strconv.Itoa(int(s.e)), dir+"/"+s.PrefixForReadingFile+"/Data/Throughput/"+s.Name+".txt")
	}

	if START_EPOCH {
		// TODO change with a random leader
		if s.Name == "node_0" {
			err = s.GetConsencusOnNewSigners()
			if err != nil {
				return nil, err
			}
		}
		err = s.StartNewEpoch()
		if err != nil {
			return nil, err
		}
		log.LLvl1("PASSS: ", s.Name, "is ending epoch", s.e)
	}
	return &ExecEpochReply{}, nil
}

// ExecWriteSigners handles writing request
func (s *Service) ExecWriteSigners(req *ExecWriteSigners) (*ExecWriteSignersReply, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	signRep := s.GetSigners(1)
	names, err := s.GetNamesFromSignerSet(signRep.Set)
	if err != nil {
		panic(err)
	}
	writeToFile("Joined,Epoch", dir+"/"+s.PrefixForReadingFile+"/Data/SignerSet"+s.Name+".txt")
	for _, name := range names {
		writeToFile(name+","+strconv.Itoa(int(s.e)), dir+"/"+s.PrefixForReadingFile+"/Data/SignerSet"+s.Name+".txt")
	}

	return &ExecWriteSignersReply{}, nil
}

// ExecSetDuration sets the registration duration
func (s *Service) ExecSetDuration(req *SetDurationRequest) (*SetDurationReply, error) {
	log.LLvl1("Duration", req.Duration, "is set for node ", s.Name)
	REGISTRATION_DUR = req.Duration
	return &SetDurationReply{}, nil
}

//ExecUpdateForNewNode does what its name says
func (s *Service) ExecUpdateForNewNode(req *UpdateForNewNodeRequest) (*UpdateForNewNodeReply, error) {
	ro, err := s.getRosterForEpoch(req.Epoch)

	if s.GetEpoch() != req.Epoch-1 || err != nil || ro == nil {
		log.LLvl1(s.Name, "is trying to update")
		name := s.GetRandomName()
		err = s.UpdateHistoryWith(name)
		if err != nil {
			return nil, err
		}
	}
	if s.Cycle.GetCurrentPhase() != REGISTRATION {
		log.LLvl1(s.Name, "is waiting ", s.Cycle.GetTimeTillNextCycle(), "s to register")
		time.Sleep(s.Cycle.GetTimeTillNextCycle() + 100*time.Millisecond)
	}
	return &UpdateForNewNodeReply{}, nil
}

//ExecUpdate does what its name says
func (s *Service) ExecUpdate(req *UpdateRequest) (*UpdateReply, error) {
	name := s.GetRandomName()
	err := s.UpdateHistoryWith(name)
	if err != nil {
		return nil, err
	}
	log.LLvl1(s.Name, "is trying to update")
	return &UpdateReply{}, nil
}

//ExecCreateProofForEpoch does what its name says
func (s *Service) ExecCreateProofForEpoch(req *CreateProofForEpochRequest) (*CreateProofForEpochReply, error) {
	err := s.CreateProofForEpoch(req.Epoch)
	if err != nil {
		return nil, err
	}
	return &CreateProofForEpochReply{}, nil
}

//ExecStartNewEpoch does what its name says
func (s *Service) ExecStartNewEpoch(req *StartNewEpochRequest) (*StartNewEpochReply, error) {
	err := s.StartNewEpoch()
	if err != nil {
		return nil, err
	}
	return &StartNewEpochReply{}, nil
}

//ExecGetConsencusOnNewSigners does what its name says
func (s *Service) ExecGetConsencusOnNewSigners(req *GetConsencusOnNewSignersRequest) (*GetConsencusOnNewSignersReply, error) {
	err := s.GetConsencusOnNewSigners()
	if err != nil {
		return nil, err
	}
	return &GetConsencusOnNewSignersReply{}, nil
}

//ExecPause does what its name says
func (s *Service) ExecPause(req *SendPauseRequest) (*SendPauseReply, error) {
	time.Sleep(10 * time.Hour)
	return &SendPauseReply{}, nil
}
