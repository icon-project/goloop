package txresult

import (
	"container/list"
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

var ReceiptType = reflect.TypeOf((*receipt)(nil))

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

func (log *eventLog) MarshalJSON() ([]byte, error) {
	jso := log.ToJSON(module.JSONVersionLast)
	return json.Marshal(jso)
}

func (log *eventLog) ToJSON(v module.JSONVersion) *eventLogJSON {
	if jso, err := log.toValidJSON(v); err == nil {
		return jso
	}
	return log.toFallbackJSON()
}

func (log *eventLog) toFallbackJSON() *eventLogJSON {
	indexed := make([]interface{}, len(log.eventLogData.Indexed))
	data := make([]interface{}, len(log.eventLogData.Data))
	for i, d := range log.eventLogData.Indexed {
		indexed[i] = d
	}
	for i, d := range log.eventLogData.Data {
		data[i] = d
	}
	return &eventLogJSON{
		Addr:    log.eventLogData.Addr,
		Indexed: indexed,
		Data:    data,
	}
}

func (log *eventLog) toValidJSON(module.JSONVersion) (*eventLogJSON, error) {
	sig := string(log.eventLogData.Indexed[0])
	_, pts := DecomposeEventSignature(sig)
	count := len(log.eventLogData.Indexed) + len(log.eventLogData.Data)
	if len(pts)+1 != count {
		if ns, ok := signaturePatches[sig]; ok {
			_, pts = DecomposeEventSignature(ns)
		}
		if len(pts)+1 != count {
			return nil, errors.InvalidStateError.New("NumberOfParametersAreNotSameAsData")
		}
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

const (
	ExtensionFeeDetail = 1 << iota
	ExtensionDisableLogsBloom
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
	DisableLogsBloom   bool
}

func (r *receiptData) Equal(r2 *receiptData) bool {
	return r.Status == r2.Status &&
		r.To.Equal(&r2.To) &&
		r.CumulativeStepUsed.Cmp(&r2.CumulativeStepUsed.Int) == 0 &&
		r.StepUsed.Cmp(&r2.StepUsed.Int) == 0 &&
		r.StepPrice.Cmp(&r2.StepPrice.Int) == 0 &&
		r.LogsBloom.Equal(&r2.LogsBloom) &&
		r.SCOREAddress.Equal(r2.SCOREAddress) &&
		r.DisableLogsBloom == r2.DisableLogsBloom &&
		reflect.DeepEqual(r.FeeDetail, r2.FeeDetail)
}

func (r *receiptData) Extension() int {
	var extension int
	if r.FeeDetail.Has() {
		extension |= ExtensionFeeDetail
	}
	if r.DisableLogsBloom {
		extension |= ExtensionDisableLogsBloom
	}
	return extension
}

type receipt struct {
	version   Version
	db        db.Database
	data      receiptData
	eventLogs trie.ImmutableForObject
	logsBloom []byte
	reason    error
	// steps for fee
	feeSteps *big.Int
	btpMsgs  *list.List
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
		extension := r.data.Extension()
		e2, err := e.EncodeList()
		if err != nil {
			return err
		}
		if err = e2.EncodeMulti(
			r.data.Status,
			&r.data.To,
			&r.data.CumulativeStepUsed,
			&r.data.StepUsed,
			&r.data.StepPrice,
			r.logsBloom,
			r.data.EventLogs,
			r.data.SCOREAddress,
			hash,
			extension,
		); err != nil {
			return err
		}
		if (extension & ExtensionFeeDetail) != 0 {
			if err = e2.Encode(&r.data.FeeDetail); err != nil {
				return err
			}
		}
		return nil
	}
}

func (r *receipt) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}
	var logsBloom []byte
	var hash []byte
	var extension int
	if cnt, err := d2.DecodeMulti(
		&r.data.Status,
		&r.data.To,
		&r.data.CumulativeStepUsed,
		&r.data.StepUsed,
		&r.data.StepPrice,
		&logsBloom,
		&r.data.EventLogs,
		&r.data.SCOREAddress,
		&hash,
		&extension,
	); err == nil || err == io.EOF {
		if cnt == listItemsForVersion1 {
			r.version = Version1
		} else if cnt == listItemsForVersion2 {
			r.version = Version2
		} else if cnt == listItemsForVersion3 {
			r.version = Version3
			if (extension & ExtensionFeeDetail) != 0 {
				if err := d2.Decode(&r.data.FeeDetail); err != nil {
					return err
				}
			}
		} else {
			return codec.ErrInvalidFormat
		}
		if r.version >= Version2 {
			r.eventLogs = trie_manager.NewImmutableForObject(r.db, hash,
				reflect.TypeOf((*eventLog)(nil)))
		} else {
			r.eventLogs = nil
		}
		if r.version >= Version3 {
			r.logsBloom = logsBloom
			r.data.LogsBloom.SetCompressedBytes(logsBloom)
			if (extension & ExtensionDisableLogsBloom) != 0 {
				r.data.DisableLogsBloom = true
			}
		} else {
			r.data.LogsBloom.SetBytes(logsBloom)
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

func (r *receipt) BTPMessages() *list.List {
	return r.btpMsgs
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

// AddPayment add payment information
// addr is payer. steps is total steps paid by the payer.
// feeSteps is amount of steps for fee.
// ( steps - feeSteps ) is virtual steps paid by the payer.
// feeSteps can be nil if there is no steps for fee
func (r *receipt) AddPayment(addr module.Address, steps *big.Int, feeSteps *big.Int) {
	ok := r.data.FeeDetail.AddPayment(addr, steps)
	if ok && r.version < Version3 {
		r.version = Version3
	}
	// cumulate steps for fee
	if feeSteps != nil {
		if r.feeSteps == nil {
			r.feeSteps = feeSteps
		} else {
			r.feeSteps = new(big.Int).Add(r.feeSteps, feeSteps)
		}
	}
}

func (r *receipt) DisableLogsBloom() {
	r.data.DisableLogsBloom = true
	r.data.LogsBloom.SetBytes(nil)
	if r.version < Version3 {
		r.version = Version3
	}
}

func (r *receipt) LogsBloomDisabled() bool {
	return r.data.DisableLogsBloom
}

type eventLogIteratorV2 struct {
	trie.IteratorForObject
}

func (i *eventLogIteratorV2) Get() (module.EventLog, error) {
	obj, _, err := i.IteratorForObject.Get()
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, nil
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
	AddBTPMessages(messages list.List)
	// AddPayment adds payment information.
	// addr is payer. steps is total steps paid by the payer.
	// feeSteps is amount of steps for fee.
	// ( steps - feeSteps ) is virtual steps paid by the payer.
	// feeSteps can be nil if there is no steps for fee
	AddPayment(addr module.Address, steps *big.Int, feeSteps *big.Int)
	// FeeByEOA returns a fee paid by EOA (not including deposit).
	FeeByEOA() *big.Int
	// Fee returns total fee (excluding virtual steps).
	Fee() *big.Int
	DisableLogsBloom()
	SetCumulativeStepUsed(cumulativeUsed *big.Int)
	SetResult(status module.Status, used, price *big.Int, addr module.Address)
	SetReason(e error)
	Reason() error
	Flush() error
}

type receiptJSON struct {
	To                 common.Address   `json:"to"`
	CumulativeStepUsed common.HexInt    `json:"cumulativeStepUsed"`
	StepUsed           common.HexInt    `json:"stepUsed"`
	StepPrice          common.HexInt    `json:"stepPrice"`
	SCOREAddress       *common.Address  `json:"scoreAddress,omitempty"`
	Failure            *failureReason   `json:"failure,omitempty"`
	EventLogs          []*eventLogJSON  `json:"eventLogs"`
	LogsBloom          *LogsBloom       `json:"logsBloom"`
	Status             common.HexUint16 `json:"status"`
	FeeDetail          feeDetail        `json:"stepUsedDetails,omitempty"`
}

func (r *receipt) ToJSON(version module.JSONVersion) (interface{}, error) {
	jso := map[string]interface{}{
		"to":                 &r.data.To,
		"cumulativeStepUsed": &r.data.CumulativeStepUsed,
		"stepUsed":           &r.data.StepUsed,
		"stepPrice":          &r.data.StepPrice,
	}

	if !r.data.DisableLogsBloom {
		jso["logsBloom"] = &r.data.LogsBloom
	}

	logs := make([]*eventLogJSON, 0, len(r.data.EventLogs))
	for itr := r.EventLogIterator(); itr.Has(); itr.Next() {
		item, err := itr.Get()
		if err != nil {
			return nil, err
		}
		logs = append(logs, item.(*eventLog).ToJSON(version))
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
	if rjson.LogsBloom != nil {
		data.LogsBloom.SetBytes(rjson.LogsBloom.Bytes())
	} else {
		data.DisableLogsBloom = true
	}
	data.FeeDetail = rjson.FeeDetail
	if r.data.Extension() != 0 && r.version < Version3 {
		r.version = Version3
	}
	if r.version >= Version2 {
		r.buildMerkleListOfLogs()
	}
	if r.version >= Version3 {
		r.logsBloom = r.data.LogsBloom.CompressedBytes()
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

func (r *receipt) AddBTPMessages(messages list.List) {
	if r.btpMsgs == nil {
		r.btpMsgs = list.New()
	}
	r.btpMsgs.PushBackList(&messages)
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
		r.logsBloom = r.data.LogsBloom.CompressedBytes()
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

func (r *receipt) stepsPaidByEOA() *big.Int {
	if r.data.FeeDetail.Has() {
		return r.data.FeeDetail.GetStepsPaidByEOA()
	}
	return r.data.StepUsed.Value()
}

func (r *receipt) Fee() *big.Int {
	var feeSteps *big.Int
	if r.data.FeeDetail.Has() {
		if r.feeSteps != nil {
			feeSteps = r.feeSteps
		} else {
			return new(big.Int)
		}
	} else {
		feeSteps = r.data.StepUsed.Value()
	}
	return new(big.Int).Mul(feeSteps, r.data.StepPrice.Value())
}

func (r *receipt) FeeByEOA() *big.Int {
	return new(big.Int).Mul(r.stepsPaidByEOA(), r.data.StepPrice.Value())
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
		return errors.InvalidStateError.New("DifferentData")
	}
	for itr1, itr2, idx := r.EventLogIterator(), r2.EventLogIterator(), 0; itr1.Has() || itr2.Has(); idx, _, _ = idx+1, itr1.Next(), itr2.Next() {
		ev1, err := itr1.Get()
		if err != nil {
			return errors.InvalidStateError.Wrap(err, "FailOnReadingEvents")
		}
		ev2, err := itr2.Get()
		if err != nil {
			return errors.InvalidStateError.Wrap(err, "FailOnReadingEvents")
		}
		elog1 := ev1.(*eventLog)
		elog2 := ev2.(*eventLog)
		if !elog1.Equal(elog2) {
			return errors.InvalidStateError.Errorf("DifferentEvent(idx=%d)", idx)
		}
	}
	return nil
}

func (r *receipt) FeePaymentIterator() module.FeePaymentIterator {
	return r.data.FeeDetail.Iterator()
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

var signaturePatches = map[string]string{
	// fill patches if required
}

func eventLogFromJSON(e *eventLogJSON) (*eventLog, error) {
	el := new(eventLog)
	el.eventLogData.Addr = e.Addr
	el.eventLogData.Indexed = make([][]byte, len(e.Indexed))
	el.eventLogData.Data = make([][]byte, len(e.Data))
	sig := e.Indexed[0].(string)
	_, pts := DecomposeEventSignature(sig)
	count := len(e.Indexed) + len(e.Data)
	if len(pts)+1 != count {
		if ns, ok := signaturePatches[sig]; ok {
			_, pts = DecomposeEventSignature(ns)
		}
		if len(pts)+1 != count {
			return nil, errors.InvalidStateError.New("InvalidSignatureCount")
		}
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
