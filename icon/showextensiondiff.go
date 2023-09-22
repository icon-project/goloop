package icon

import (
	"bytes"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/service"
)

type ExtensionValues struct {
	State  common.HexBytes `json:"state"`
	Front  common.HexBytes `json:"front"`
	Back   common.HexBytes `json:"back"`
	Back2  common.HexBytes `json:"back2"`
	Reward common.HexBytes `json:"reward"`
}

func ShowExtensionDiff(c service.DiffContext, e, r []byte) error {
	var ev, rv ExtensionValues
	_, err := codec.BC.UnmarshalFromBytes(e, &ev)
	if err != nil {
		return err
	}
	_, err = codec.BC.UnmarshalFromBytes(r, &rv)
	if err != nil {
		return err
	}
	if !bytes.Equal(ev.State.Bytes(), rv.State.Bytes()) {
		tdb := icobject.AttachObjectFactory(c.Database(), icstate.NewObjectImpl)
		err := c.ShowObjectMPTDiff("ext.state", tdb, icobject.ObjectType,
			ev.State.Bytes(), rv.State.Bytes(), nil)
		if err != nil {
			return err
		}
	}
	if !bytes.Equal(ev.Front.Bytes(), rv.Front.Bytes()) {
		tdb := icobject.AttachObjectFactory(c.Database(), icstage.NewObjectImpl)
		err := c.ShowObjectMPTDiff("ext.front", tdb, icobject.ObjectType,
			ev.Front.Bytes(), rv.Front.Bytes(), nil)
		if err != nil {
			return err
		}
	}
	if !bytes.Equal(ev.Back.Bytes(), rv.Back.Bytes()) {
		tdb := icobject.AttachObjectFactory(c.Database(), icstage.NewObjectImpl)
		err := c.ShowObjectMPTDiff("ext.back", tdb, icobject.ObjectType,
			ev.Back.Bytes(), rv.Back.Bytes(), nil)
		if err != nil {
			return err
		}
	}
	if !bytes.Equal(ev.Back2.Bytes(), rv.Back2.Bytes()) {
		tdb := icobject.AttachObjectFactory(c.Database(), icstage.NewObjectImpl)
		err := c.ShowObjectMPTDiff("ext.back2", tdb, icobject.ObjectType,
			ev.Back2.Bytes(), rv.Back2.Bytes(), nil)
		if err != nil {
			return err
		}
	}
	if !bytes.Equal(ev.Reward.Bytes(), rv.Reward.Bytes()) {
		tdb := icobject.AttachObjectFactory(c.Database(), icreward.NewObjectImpl)
		err := c.ShowObjectMPTDiff("ext.reward", tdb, icobject.ObjectType,
			ev.Reward.Bytes(), rv.Reward.Bytes(), nil)
		if err != nil {
			return err
		}
	}
	return nil
}