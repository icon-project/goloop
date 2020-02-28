package trace

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func ToJSON(v interface{}) string {
	bs, err := json.MarshalIndent(v, "", "")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	buf := bytes.NewBuffer(nil)
	err = json.Compact(buf, bs)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return buf.String()
}
