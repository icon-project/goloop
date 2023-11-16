package transaction

import (
	"bytes"
	"encoding/json"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
)

type transactionJSON struct {
	transactionV3Data
	Fee      common.HexInt   `json:"fee"`               // V2 only
	TxHash   common.HexBytes `json:"txHash,omitempty"`  // V3 only
	TxHashV2 common.HexBytes `json:"tx_hash,omitempty"` // V2 only

	raw []byte
}

const (
	Version2 = 2
	Version3 = 3
)

var (
	transactionSaltBytes = []byte("icx_sendTransaction.")
	transactionFields    = map[int]struct {
		inclusion map[string]bool
		exclusion map[string]bool
	}{
		Version2: {
			exclusion: map[string]bool{
				"method":    true,
				"signature": true,
				"tx_hash":   true,
			},
		},
		Version3: {
			exclusion: map[string]bool{
				"signature": true,
				"txHash":    true,
			},
		},
	}
)

func calcHashOfTransactionJSON(bs []byte, version int) ([]byte, error) {
	var data map[string]interface{}
	var err error
	if err = json.Unmarshal(bs, &data); err != nil {
		return nil, err
	}
	return calcHashOfTransactionJSMap(data, version)
}

func calcHashOfTransactionJSMap(data map[string]any, version int) ([]byte, error) {
	fields, ok := transactionFields[version]
	if !ok {
		return nil, errors.IllegalArgumentError.Errorf("InvalidVersion(version=%d)", version)
	}
	bs, err := SerializeMap(data, fields.inclusion, fields.exclusion)
	if err != nil {
		return nil, InvalidFormat.Wrapf(err, "Serialize FAILs(%s)", string(bs))
	}
	bs = append(transactionSaltBytes, bs...)

	return crypto.SHA3Sum256(bs), nil
}

func (tx *transactionJSON) calcHash(version int) ([]byte, error) {
	return calcHashOfTransactionJSON(tx.raw, version)
}

func parseTransactionJSON(js []byte) (*transactionJSON, error) {
	jso := new(transactionJSON)
	jso.Version.Value = 2
	if err := json.Unmarshal(js, jso); err != nil {
		return nil, InvalidFormat.Wrapf(err, "Invalid json for transactionV3(%s)", string(js))
	}
	jso.raw = js
	return jso, nil
}

func jsonCompact(js []byte) ([]byte, error) {
	b := bytes.NewBuffer(nil)
	if err := json.Compact(b, js); err != nil {
		return nil, err
	} else {
		return b.Bytes(), nil
	}
}
