package icstate

import (
	"fmt"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/service/scoreresult"
)

const (
	JFlagInJail = 1 << iota
	JFlagUnjailing
	JFlagDoubleVote
)

type JailInfo struct {
	flags               int
	unjailRequestHeight int64
	minDoubleVoteHeight int64
}

func (ji *JailInfo) Flags() int {
	return ji.flags
}

func (ji *JailInfo) SetFlags(flags int) {
	ji.flags = flags
}

func (ji *JailInfo) IsInJail() bool {
	return icutils.MatchAll(ji.flags, JFlagInJail)
}

func (ji *JailInfo) IsUnjailing() bool {
	return icutils.MatchAll(ji.flags, JFlagUnjailing)
}

func (ji *JailInfo) IsInDoubleVotePenalty() bool {
	return icutils.MatchAll(ji.flags, JFlagDoubleVote)
}

func (ji *JailInfo) UnjailRequestHeight() int64 {
	return ji.unjailRequestHeight
}

func (ji *JailInfo) SetUnajilRequestHeight(blockHeight int64) {
	ji.unjailRequestHeight = blockHeight
}

func (ji *JailInfo) MinDoubleVoteHeight() int64 {
	return ji.minDoubleVoteHeight
}

func (ji *JailInfo) SetMinDoubleVoteHeight(blockHeight int64) {
	ji.minDoubleVoteHeight = blockHeight
}

func (ji *JailInfo) IsEmpty() bool {
	return ji.flags == 0 && ji.unjailRequestHeight == 0 && ji.minDoubleVoteHeight == 0
}

func (ji *JailInfo) RequestUnjail(blockHeight int64) error {
	if blockHeight < 0 {
		return scoreresult.InvalidParameterError.Errorf("InvalidBlockHeight(%d)", blockHeight)
	}
	if ji.flags&(JFlagInJail|JFlagUnjailing) == JFlagInJail {
		ji.flags |= JFlagUnjailing
		ji.unjailRequestHeight = blockHeight
	}
	return nil
}

func (ji *JailInfo) ToJSON(jso map[string]interface{}) map[string]interface{} {
	if jso == nil {
		jso = make(map[string]interface{})
	}
	jso["jailFlags"] = ji.flags
	jso["unjailRequestHeight"] = ji.unjailRequestHeight
	jso["minDoubleVoteHeight"] = ji.minDoubleVoteHeight
	return jso
}

func (ji *JailInfo) RLPDecodeSelf(d codec.Decoder) error {
	err := d.DecodeListOf(&ji.flags, &ji.unjailRequestHeight, &ji.minDoubleVoteHeight)
	return err
}

func (ji *JailInfo) RLPEncodeSelf(e codec.Encoder) error {
	return e.EncodeListOf(ji.flags, ji.unjailRequestHeight, ji.minDoubleVoteHeight)
}

func (ji JailInfo) String() string {
	return fmt.Sprintf("JailInfo{%d %d %d}", ji.flags, ji.unjailRequestHeight, ji.minDoubleVoteHeight)
}

func (ji JailInfo) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		var format string
		if f.Flag('+') {
			format = "JailInfo{flags:%d urbh:%d mdvbh:%d}"
		} else {
			format = "JailInfo{%d %d %d}"
		}
		_, _ = fmt.Fprintf(f, format, ji.flags, ji.unjailRequestHeight, ji.minDoubleVoteHeight)
	case 's':
		_, _ = fmt.Fprint(f, ji.String())
	}
}
