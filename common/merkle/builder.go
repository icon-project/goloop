package merkle

import (
	"container/list"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/pkg/errors"
)

type DataRequester interface {
	OnData(value []byte, builder Builder) error
}

type RequestIterator interface {
	Next() bool
	Key() []byte
}

type Builder interface {
	OnData(value []byte) error
	UnresolvedCount() int
	Requests() RequestIterator
	RequestData(id db.BucketID, key []byte, requester DataRequester)
	Database() db.LayerDB
	Flush(write bool) error
}

type request struct {
	key        []byte
	bucketIDs  []db.BucketID
	requesters []DataRequester
}

type merkleBuilder struct {
	layer      db.LayerDB
	requests   *list.List
	requestMap map[string]*list.Element
}

type requestIterator struct {
	element *list.Element
	request *request
}

func (i *requestIterator) Next() bool {
	if i.element == nil {
		return false
	} else {
		i.request = i.element.Value.(*request)
		i.element = i.element.Next()
		return true
	}
}

func (i *requestIterator) Key() []byte {
	if i.request != nil {
		return i.request.key
	}
	return nil
}

func (b *merkleBuilder) Requests() RequestIterator {
	return &requestIterator{
		element: b.requests.Front(),
	}
}

func (b *merkleBuilder) OnData(value []byte) error {
	key := crypto.SHA3Sum256(value)
	reqID := string(key)
	if e, ok := b.requestMap[reqID]; ok {
		req := e.Value.(*request)
		for i, requester := range req.requesters {
			bkID := req.bucketIDs[i]
			bk, err := b.layer.GetBucket(bkID)
			if err != nil {
				return err
			}
			if err := bk.Set(key, value); err != nil {
				return err
			}
			if err := requester.OnData(value, b); err != nil {
				return err
			}
		}
		b.requests.Remove(e)
		return nil
	} else {
		return errors.New("IllegalArguments")
	}
}

func (b *merkleBuilder) RequestData(id db.BucketID, key []byte, requester DataRequester) {
	reqID := string(key)
	if e, ok := b.requestMap[reqID]; ok {
		req := e.Value.(*request)
		req.bucketIDs = append(req.bucketIDs, id)
		req.requesters = append(req.requesters, requester)
	} else {
		req := &request{
			key:        key,
			bucketIDs:  []db.BucketID{id},
			requesters: []DataRequester{requester},
		}
		e := b.requests.PushBack(req)
		b.requestMap[reqID] = e
	}
}

func (b *merkleBuilder) UnresolvedCount() int {
	return b.requests.Len()
}

func (b *merkleBuilder) Flush(write bool) error {
	return b.layer.Flush(write)
}

func (b *merkleBuilder) Database() db.LayerDB {
	return b.layer
}

func NewBuilder(dbase db.Database) Builder {
	ldb := db.NewLayerDB(dbase)
	builder := &merkleBuilder{
		layer:      ldb,
		requests:   list.New(),
		requestMap: make(map[string]*list.Element),
	}
	return builder
}
