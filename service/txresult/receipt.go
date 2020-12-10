package txresult

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"reflect"
	"regexp"
	"strings"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
)

const (
	EventLogICXTransfer = "ICXTransfer(Address,Address,int)"
)

type eventLogJSON struct {
	Addr    common.Address `json:"scoreAddress"`
	Indexed []interface{}  `json:"indexed"`
	Data    []interface{}  `json:"data"`
}

type eventLogData struct {
	Addr    common.Address
	Indexed [][]byte
	Data    [][]byte
}

type eventLog struct {
	eventLogData
}

func (log *eventLog) Bytes() []byte {
	return codec.BC.MustMarshalToBytes(log)
}

func (log *eventLog) Reset(s db.Database, k []byte) error {
	_, err := codec.BC.UnmarshalFromBytes(k, log)
	return err
}

func (log *eventLog) Flush() error {
	// do nothing
	return nil
}

func (log *eventLog) Equal(obj trie.Object) bool {
	if l2, ok := obj.(*eventLog); ok {
		return reflect.DeepEqual(&log.eventLogData, &l2.eventLogData)
	}
	return false
}

func (log *eventLog) Resolve(builder merkle.Builder) error {
	// nothing to do
	return nil
}

func (log *eventLog) ClearCache() {
	return
}

func (log *eventLog) Address() module.Address {
	return &log.eventLogData.Addr
}

func (log *eventLog) Indexed() [][]byte {
	return log.eventLogData.Indexed
}

func (log *eventLog) Data() [][]byte {
	return log.eventLogData.Data
}

func (log *eventLog) ToJSON(module.JSONVersion) (*eventLogJSON, error) {
	_, pts := DecomposeEventSignature(string(log.eventLogData.Indexed[0]))
	if len(pts)+1 != len(log.eventLogData.Indexed)+len(log.eventLogData.Data) {
		return nil, errors.InvalidStateError.New("NumberOfParametersAreNotSameAsData")
	}

	eljson := new(eventLogJSON)
	eljson.Addr = log.eventLogData.Addr
	eljson.Indexed = make([]interface{}, len(log.eventLogData.Indexed))
	eljson.Data = make([]interface{}, len(log.eventLogData.Data))

	aidx := 0
	eljson.Indexed[0] = string(log.eventLogData.Indexed[0])
	for i, v := range log.eventLogData.Indexed[1:] {
		if s, err := DecodeForJSONByType(pts[aidx], v); err != nil {
			return nil, err
		} else {
			eljson.Indexed[i+1] = s
			aidx++
		}
	}
	for i, v := range log.eventLogData.Data {
		if s, err := DecodeForJSONByType(pts[aidx], v); err != nil {
			return nil, err
		} else {
			eljson.Data[i] = s
			aidx++
		}
	}
	return eljson, nil
}

type Version int

const (
	Version1 Version = iota
	Version2
	Version3
	ReservedVersion
	LastVersion = ReservedVersion - 1
)
const (
	listItemsForVersion1 = 8
	listItemsForVersion2 = 9
	listItemsForVersion3 = 10
)

type receiptData struct {
	Status             module.Status
	To                 common.Address
	CumulativeStepUsed common.HexInt
	StepUsed           common.HexInt
	StepPrice          common.HexInt
	LogsBloom          LogsBloom
	EventLogs          []*eventLog
	SCOREAddress       *common.Address
	FeeDetail          feeDetail
}

func (r *receiptData) Equal(r2 *receiptData) bool {
	return r.Status == r2.Status &&
		r.To.Equal(&r2.To) &&
		r.CumulativeStepUsed.Cmp(&r2.CumulativeStepUsed.Int) == 0 &&
		r.StepUsed.Cmp(&r2.StepUsed.Int) == 0 &&
		r.StepPrice.Cmp(&r2.StepPrice.Int) == 0 &&
		r.LogsBloom.Equal(&r2.LogsBloom) &&
		reflect.DeepEqual(r.EventLogs, r2.EventLogs) &&
		r.SCOREAddress.Equal(r2.SCOREAddress) &&
		reflect.DeepEqual(r.FeeDetail, r2.FeeDetail)
}

type receipt struct {
	version   Version
	db        db.Database
	data      receiptData
	eventLogs trie.ImmutableForObject
	reason    error
}

func (r *receipt) SCOREAddress() module.Address {
	return r.data.SCOREAddress
}

func (r *receipt) To() module.Address {
	return &r.data.To
}

func (r *receipt) Bytes() []byte {
	bs, err := codec.BC.MarshalToBytes(r)
	if err != nil {
		log.Errorf("Fail to marshal object err=%+v\ndata=%+v\n", err, r.data)
	}
	return bs
}

func (r *receipt) Reset(s db.Database, k []byte) error {
	r.db = s
	_, err := codec.BC.UnmarshalFromBytes(k, r)
	return err
}

func (r *receipt) Flush() error {
	if r.version >= Version2 {
		if ss, ok := r.eventLogs.(trie.SnapshotForObject); ok {
			return ss.Flush()
		}
	}
	return nil
}

func (r *receipt) Equal(o trie.Object) bool {
	if rct, ok := o.(*receipt); ok {
		if rct == r {
			return true
		}
	}
	return false
}

func (r *receipt) ClearCache() {
	if r.version >= Version2 {
		r.eventLogs.ClearCache()
	}
}

func (r *receipt) RLPEncodeSelf(e codec.Encoder) error {
	if r.version == Version1 {
		return e.EncodeListOf(
			r.data.Status,
			&r.data.To,
			&r.data.CumulativeStepUsed,
			&r.data.StepUsed,
			&r.data.StepPrice,
			&r.data.LogsBloom,
			r.data.EventLogs,
			r.data.SCOREAddress,
		)
	} else if r.version == Version2 {
		hash := r.eventLogs.Hash()
		return e.EncodeListOf(
			r.data.Status,
			&r.data.To,
			&r.data.CumulativeStepUsed,
			&r.data.StepUsed,
			&r.data.StepPrice,
			&r.data.LogsBloom,
			r.data.EventLogs,
			r.data.SCOREAddress,
			hash)
	} else {
		hash := r.eventLogs.Hash()
		return e.EncodeListOf(
			r.data.Status,
			&r.data.To,
			&r.data.CumulativeStepUsed,
			&r.data.StepUsed,
			&r.data.StepPrice,
			&r.data.LogsBloom,
			r.data.EventLogs,
			r.data.SCOREAddress,
			hash,
			r.data.FeeDetail,
		)
	}
}

func (r *receipt) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}
	var hash []byte
	if cnt, err := d2.DecodeMulti(
		&r.data.Status,
		&r.data.To,
		&r.data.CumulativeStepUsed,
		&r.data.StepUsed,
		&r.data.StepPrice,
		&r.data.LogsBloom,
		&r.data.EventLogs,
		&r.data.SCOREAddress,
		&hash,
		&r.data.FeeDetail,
	); err == nil || err == io.EOF {
		if cnt == listItemsForVersion1 {
			r.version = Version1
			r.eventLogs = nil
		} else if cnt == listItemsForVersion2 {
			r.version = Version2
			r.eventLogs = trie_manager.NewImmutableForObject(r.db, hash,
				reflect.TypeOf((*eventLog)(nil)))
		} else if cnt == listItemsForVersion3 {
			r.version = Version3
			r.eventLogs = trie_manager.NewImmutableForObject(r.db, hash,
				reflect.TypeOf((*eventLog)(nil)))
		} else {
			return codec.ErrInvalidFormat
		}
	} else {
		return err
	}
	return nil
}

func (r *receipt) Resolve(bd merkle.Builder) error {
	if r.version >= Version2 {
		r.eventLogs.Resolve(bd)
	}
	return nil
}

func (r *receipt) LogsBloom() module.LogsBloom {
	return &r.data.LogsBloom
}

func (r *receipt) EventLogIterator() module.EventLogIterator {
	if r.version >= Version2 {
		return &eventLogIteratorV2{r.eventLogs.Iterator()}
	}
	return &eventLogIterator{r.data.EventLogs, 0}
}

func (r *receipt) GetProofOfEvent(i int) ([][]byte, error) {
	if r.version < Version2 {
		return nil, errors.ErrInvalidState
	}
	k := codec.BC.MustMarshalToBytes(uint(i))
	proof := r.eventLogs.GetProof(k)
	if proof == nil {
		return nil, errors.NotFoundError.Errorf("EventNotFound(idx=%d)", i)
	}
	return proof, nil
}

func (r *receipt) AddPayment(addr module.Address, steps *big.Int) {
	ok := r.data.FeeDetail.AddPayment(addr, steps)
	if ok && r.version < Version3 {
		r.version = Version3
	}
}

type eventLogIteratorV2 struct {
	trie.IteratorForObject
}

func (i *eventLogIteratorV2) Get() (module.EventLog, error) {
	obj, _, err := i.IteratorForObject.Get()
	if err != nil {
		return nil, err
	}
	return obj.(module.EventLog), nil
}

type eventLogIterator struct {
	slice []*eventLog
	index int
}

func (it *eventLogIterator) Has() bool {
	return it.index < len(it.slice)
}

func (it *eventLogIterator) Next() error {
	if it.index >= len(it.slice) {
		return errors.InvalidStateError.New("no next item")
	}
	it.index++
	return nil
}

func (it *eventLogIterator) Get() (module.EventLog, error) {
	if !it.Has() {
		return nil, errors.InvalidStateError.New("no item")
	}
	return it.slice[it.index], nil
}

type failureReason struct {
	CodeValue    common.HexUint16 `json:"code"`
	MessageValue string           `json:"message"`
}

func (f *failureReason) Code() uint16 {
	return f.CodeValue.Value
}

func (f *failureReason) Message() string {
	return f.MessageValue
}

func failureReasonByCode(status module.Status) *failureReason {
	return &failureReason{
		CodeValue:    common.HexUint16{Value: uint16(status)},
		MessageValue: status.String(),
	}
}

type Receipt interface {
	module.Receipt
	AddLog(addr module.Address, indexed, data [][]byte)
	AddPayment(addr module.Address, steps *big.Int)
	SetCumulativeStepUsed(cumulativeUsed *big.Int)
	SetResult(status module.Status, used, price *big.Int, addr module.Address)
	SetReason(e error)
	Reason() error
}

type receiptJSON struct {
	To                 common.Address   `json:"to"`
	CumulativeStepUsed common.HexInt    `json:"cumulativeStepUsed"`
	StepUsed           common.HexInt    `json:"stepUsed"`
	StepPrice          common.HexInt    `json:"stepPrice"`
	SCOREAddress       *common.Address  `json:"scoreAddress,omitempty"`
	Failure            *failureReason   `json:"failure,omitempty"`
	EventLogs          []*eventLogJSON  `json:"eventLogs"`
	LogsBloom          LogsBloom        `json:"logsBloom"`
	Status             common.HexUint16 `json:"status"`
	FeeDetail          feeDetail        `json:"stepUsedDetails"`
}

func (r *receipt) ToJSON(version module.JSONVersion) (interface{}, error) {
	jso := map[string]interface{}{
		"to":                 &r.data.To,
		"cumulativeStepUsed": &r.data.CumulativeStepUsed,
		"stepUsed":           &r.data.StepUsed,
		"stepPrice":          &r.data.StepPrice,
		"logsBloom":          &r.data.LogsBloom,
	}

	logs := make([]*eventLogJSON, 0, len(r.data.EventLogs))
	for itr := r.EventLogIterator(); itr.Has(); itr.Next() {
		item, err := itr.Get()
		if err != nil {
			return nil, err
		}
		if jso, err := item.(*eventLog).ToJSON(version); err != nil {
			return nil, err
		} else {
			logs = append(logs, jso)
		}
	}
	jso["eventLogs"] = logs

	if r.data.FeeDetail.Has() {
		details, err := r.data.FeeDetail.ToJSON(version)
		if err != nil {
			return nil, err
		}
		jso["stepUsedDetails"] = details
	}

	if r.data.Status == module.StatusSuccess {
		jso["status"] = "0x1"
		if r.data.SCOREAddress != nil {
			jso["scoreAddress"] = r.data.SCOREAddress
		}
	} else {
		jso["status"] = "0x0"
		jso["failure"] = failureReasonByCode(r.data.Status)
	}
	return jso, nil
}

func (r *receipt) MarshalJSON() ([]byte, error) {
	obj, err := r.ToJSON(module.JSONVersionLast)
	if err != nil {
		return nil, err
	}
	return json.Marshal(obj)
}

func (r *receipt) UnmarshalJSON(bs []byte) error {
	var rjson receiptJSON
	if err := json.Unmarshal(bs, &rjson); err != nil {
		return err
	}
	data := &r.data
	if rjson.Status.Value == 1 {
		data.Status = module.StatusSuccess
		data.SCOREAddress = rjson.SCOREAddress
	} else {
		data.Status = module.Status(rjson.Failure.CodeValue.Value)
	}
	data.To = rjson.To
	data.CumulativeStepUsed.Set(&rjson.CumulativeStepUsed.Int)
	data.StepUsed.Set(&rjson.StepUsed.Int)
	data.StepPrice.Set(&rjson.StepPrice.Int)
	if len(rjson.EventLogs) > 0 {
		data.EventLogs = make([]*eventLog, len(rjson.EventLogs))
		for i, e := range rjson.EventLogs {
			if el, err := eventLogFromJSON(e); err != nil {
				return err
			} else {
				data.EventLogs[i] = el
			}
		}
	}
	data.LogsBloom.SetBytes(rjson.LogsBloom.Bytes())
	data.FeeDetail = rjson.FeeDetail
	if r.data.FeeDetail.Has() {
		r.version = Version3
	}
	if r.version >= Version2 {
		r.buildMerkleListOfLogs()
	}
	return nil
}

func (r *receipt) AddLog(addr module.Address, indexed, data [][]byte) {
	log := new(eventLog)
	log.eventLogData.Addr.Set(addr)
	log.eventLogData.Indexed = indexed
	log.eventLogData.Data = data

	r.data.EventLogs = append(r.data.EventLogs, log)
	r.data.LogsBloom.AddLog(&log.eventLogData.Addr, log.eventLogData.Indexed)
}

func (r *receipt) SetCumulativeStepUsed(cumulativeUsed *big.Int) {
	r.data.CumulativeStepUsed.Set(cumulativeUsed)
}

func (r *receipt) buildMerkleListOfLogs() {
	mt := trie_manager.NewMutableForObject(r.db, nil, reflect.TypeOf((*eventLog)(nil)))
	for idx, item := range r.data.EventLogs {
		k, _ := codec.BC.MarshalToBytes(uint(idx))
		_, err := mt.Set(k, item)
		if err != nil {
			log.Panicf("Fail to add event log to the list err=%+v", err)
		}
	}
	r.eventLogs = mt.GetSnapshot()
	r.data.EventLogs = nil
}

func (r *receipt) SetResult(status module.Status, used, price *big.Int, addr module.Address) {
	r.data.Status = status
	if status == module.StatusSuccess && addr != nil {
		r.data.SCOREAddress = common.AddressToPtr(addr)
	}
	r.data.StepUsed.Set(used)
	r.data.StepPrice.Set(price)
	if r.version >= Version2 {
		r.buildMerkleListOfLogs()
	}
	if r.version >= Version3 {
		r.data.FeeDetail.Normalize()
	}
}

func (r *receipt) SetReason(e error) {
	r.reason = e
}

func (r *receipt) Reason() error {
	return r.reason
}

func (r *receipt) CumulativeStepUsed() *big.Int {
	p := new(big.Int)
	p.Set(&r.data.CumulativeStepUsed.Int)
	return p
}

func (r *receipt) StepPrice() *big.Int {
	p := new(big.Int)
	p.Set(&r.data.StepPrice.Int)
	return p
}

func (r *receipt) StepUsed() *big.Int {
	p := new(big.Int)
	p.Set(&r.data.StepUsed.Int)
	return p
}

func (r *receipt) Status() module.Status {
	return r.data.Status
}

func (r *receipt) Check(r2 module.Receipt) error {
	rct2, ok := r2.(*receipt)
	if !ok {
		return errors.IllegalArgumentError.New("IncompatibleReceipt")
	}
	if !r.data.Equal(&rct2.data) {
		return errors.InvalidStateError.New("DataIsn'tEqual")
	}
	if r.version != rct2.version {
		return errors.InvalidStateError.New("VersionMismatch")
	}
	if r.version >= Version2 {
		if !r.eventLogs.Equal(rct2.eventLogs, true) {
			return errors.InvalidStateError.New("DifferentEventLogs")
		}
	} else {
		if r.eventLogs != nil || rct2.eventLogs != nil {
			return errors.InvalidStateError.New("InvalidEventLogHash")
		}
	}
	return nil
}

func versionForRevision(revision module.Revision) Version {
	if revision.UseMPTOnEvents() {
		return Version2
	} else {
		return Version1
	}
}

func NewReceiptFromJSON(database db.Database, revision module.Revision, bs []byte) (Receipt, error) {
	r := new(receipt)
	r.version = versionForRevision(revision)
	r.db = database
	if err := json.Unmarshal(bs, r); err != nil {
		return nil, err
	}
	return r, nil
}

func NewReceipt(database db.Database, revision module.Revision, to module.Address) Receipt {
	r := new(receipt)
	r.db = database
	r.version = versionForRevision(revision)
	r.data.To.Set(to)
	return r
}

func DecomposeEventSignature(s string) (string, []string) {
	reg := regexp.MustCompile(`^(\w+)\(((?:\w+)(?:,(?:\w+))*)\)$`)
	if reg == nil {
		return "", nil
	}
	matches := reg.FindStringSubmatch(s)
	if len(matches) < 2 {
		return "", nil
	}
	return matches[1], strings.Split(matches[2], ",")
}

func DecodeForJSONByType(t string, v []byte) (interface{}, error) {
	dt := scoreapi.DataTypeOf(t)
	if dt == scoreapi.Unknown {
		return nil, errors.Errorf("UnknownType(%s)For(<% x>)", t, v)
	}
	return dt.ConvertBytesToJSO(v)
}

func EventDataStringToBytesByType(t string, v string) ([]byte, error) {
	switch t {
	case "Address":
		var addr common.Address
		if err := addr.SetString(v); err != nil {
			return nil, err
		}
		return addr.Bytes(), nil
	case "int":
		var ivalue common.HexInt
		ivalue.SetString(v, 0)
		return ivalue.Bytes(), nil
	case "str":
		return []byte(v), nil
	case "bytes":
		if len(v) < 2 || v[0:2] != "0x" {
			return nil, errors.Errorf("IllegalFormatForBytes(%s)", v)
		}
		return hex.DecodeString(v[2:])
	case "bool":
		if v == "0x1" {
			return []byte{1}, nil
		}
		return []byte{0}, nil
	default:
		return nil, errors.Errorf("UnknownType(%s)For(%s)", t, v)
	}
}

func EventDataToBytesByType(t string, v interface{}) ([]byte, error) {
	if v == nil {
		return nil, nil
	} else {
		if s, ok := v.(string); ok {
			return EventDataStringToBytesByType(t, s)
		} else {
			return nil, errors.IllegalArgumentError.Errorf("InvalidJSON(%+v)", v)
		}
	}
}

func eventLogFromJSON(e *eventLogJSON) (*eventLog, error) {
	el := new(eventLog)
	el.eventLogData.Addr = e.Addr
	el.eventLogData.Indexed = make([][]byte, len(e.Indexed))
	el.eventLogData.Data = make([][]byte, len(e.Data))
	sig := e.Indexed[0].(string)
	_, pts := DecomposeEventSignature(sig)

	if len(pts)+1 != len(e.Indexed)+len(e.Data) {
		return nil, errors.InvalidStateError.New("InvalidSignatureCount")
	}

	el.eventLogData.Indexed[0] = []byte(sig)

	aidx := 0
	for i, is := range e.Indexed[1:] {
		if bs, err := EventDataToBytesByType(pts[aidx], is); err != nil {
			return nil, err
		} else {
			el.eventLogData.Indexed[i+1] = bs
			aidx++
		}
	}

	for i, is := range e.Data {
		if bs, err := EventDataToBytesByType(pts[aidx], is); err != nil {
			return nil, err
		} else {
			el.eventLogData.Data[i] = bs
			aidx++
		}
	}

	return el, nil
}
