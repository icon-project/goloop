package common

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_RawHexBytes_UnmarshalJSON(t *testing.T) {
	var hb RawHexBytes

	err := json.Unmarshal([]byte("null"), &hb)
	if err != nil {
		t.Errorf("Fail to unmarshal json null err=%+v", err)
	}
	assert.Nil(t, hb)
}

func Test_RawHexBytes_MarshalJSON(t *testing.T) {
	var hb RawHexBytes
	bs, err := json.Marshal(hb)
	if err != nil {
		t.Errorf("Fail to marshal json null err=%+v", err)
	}
	assert.Equal(t, []byte("null"), bs)
}

func Test_HexBytes_UnmarshalJSON(t *testing.T) {
	var hb HexBytes

	err := json.Unmarshal([]byte("null"), &hb)
	if err != nil {
		t.Errorf("Fail to unmarshal json null err=%+v", err)
	}
	assert.Nil(t, hb)
}

func Test_HexBytes_MarshalJSON(t *testing.T) {
	var hb HexBytes
	bs, err := json.Marshal(hb)
	if err != nil {
		t.Errorf("Fail to marshal json null err=%+v", err)
	}
	assert.Equal(t, []byte("null"), bs)
}
