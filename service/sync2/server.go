package sync2

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type server struct {
	log         log.Logger
	merkleTrie  db.Bucket
	bytesByHash db.Bucket
}

func (s *server) hasNode(msg []byte, id module.PeerID) *result {
	hr := new(hasNode)
	if _, err := c.UnmarshalFromBytes(msg, &hr); err != nil {
		s.log.Tracef("Failed to unmarshal data (%#x)\n", msg)
		return nil
	}

	status := NoError
	for _, hash := range [][]byte{hr.StateHash, hr.PatchHash, hr.NormalHash} {
		if len(hash) == 0 {
			continue
		}
		if v, err := s.merkleTrie.Get(hash); err != nil || v == nil {
			s.log.Tracef("hasNode NoData err(%v), v(%v) hash(%#x)\n", err, v, hash)
			status = ErrNoData
			break
		}
	}

	if hr.ValidatorHash != nil {
		if v, err := s.bytesByHash.Get(hr.ValidatorHash); err != nil || v == nil {
			s.log.Tracef("hasNode NoData err(%v), v(%v) hash(%#x)\n", err, v, hr.ValidatorHash)
			status = ErrNoData
		}
	}

	res := &result{hr.ReqID, status}
	s.log.Tracef("responseResult(%s) to peer(%s)\n", res, id)

	return res
}

func (s *server) _resolveNode(hashes [][]byte) (errCode, [][]byte) {
	s.log.Tracef("_resolveNode len(%d)\n", len(hashes))
	values := make([][]byte, 0, len(hashes))
	for _, hash := range hashes {
		var err error
		var v []byte
		for _, bucket := range []db.Bucket{s.merkleTrie, s.bytesByHash} {
			if v, err = bucket.Get(hash); err == nil && v != nil {
				values = append(values, v)
				break
			}
		}
	}
	s.log.Debugf("_resolveNode values len(%d)\n", len(values))
	if len(values) == 0 {
		return ErrNoData, nil
	}
	return NoError, values
}

func (s *server) requestNode(msg []byte, id module.PeerID) *nodeData {
	req := new(requestNodeData)
	if _, err := c.UnmarshalFromBytes(msg, &req); err != nil {
		s.log.Info("Failed to unmarshal error(%+v), (%#x)\n", err, msg)
		return nil
	}

	s.log.Debugf("requestNode() request data reqID(%d), len(%d)\n", req.ReqID, len(req.Hashes))
	status, values := s._resolveNode(req.Hashes)
	s.log.Tracef("responseNode node(%d), status(%d), peer(%s)\n", len(values), status, id)
	res := &nodeData{req.ReqID, status, req.Type, values}

	return res
}

func (s *server) _resolveData(bnbs []BucketIDAndBytes) (errCode, []BucketIDAndBytes) {
	s.log.Tracef("_resolveData() len(%d)\n", len(bnbs))
	resData := make([]BucketIDAndBytes, 0, len(bnbs))

	for _, bnb := range bnbs {
		var err error
		var v []byte
		var bucket db.Bucket

		switch bnb.BkID {
		case db.MerkleTrie:
			bucket = s.merkleTrie
		case db.BytesByHash:
			bucket = s.bytesByHash
		default:
			bucket = nil
			continue
		}

		if v, err = bucket.Get(bnb.Bytes); err == nil && v != nil {
			bb := BucketIDAndBytes{BkID: bnb.BkID, Bytes: v}
			resData = append(resData, bb)
		}

	}

	s.log.Debugf("_resolveData() response data len(%d)\n", len(resData))
	if len(resData) == 0 {
		return ErrNoData, nil
	}
	return NoError, resData
}

func (s *server) requestV2(msg []byte, id module.PeerID) *responseData {
	req := new(requestData)
	if _, err := c.UnmarshalFromBytes(msg, &req); err != nil {
		s.log.Info("Failed to unmarshal error(%+v), (%#x)\n", err, msg)
		return nil
	}

	s.log.Debugf("requestV2() request data reqID(%d), len(%d)\n", req.ReqID, len(req.Data))
	status, data := s._resolveData(req.Data)
	s.log.Tracef("responseData data(%d), status(%d), peer(%s)\n", len(data), status, id)
	res := &responseData{req.ReqID, status, data}

	return res
}

func newServer(database db.Database, log log.Logger) *server {
	merkleTrie, err := database.GetBucket(db.MerkleTrie)
	if err != nil {
		log.Panicf("Failed to get bucket for MerkleTrie err(%s)\n", err)
	}

	bytesByHash, err := database.GetBucket(db.BytesByHash)
	if err != nil {
		log.Panicf("Failed to get bucket for BytesByHash err(%s)\n", err)
	}

	server := &server{
		log:         log,
		merkleTrie:  merkleTrie,
		bytesByHash: bytesByHash,
	}

	return server
}
