package iiss

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type PRepState interface {
	SetBytes(b []byte) error
	Bytes() []byte
	SetPRep(name string, email string, website string, country string, city string, details string, endpoint string,
		node module.Address) error
	GetPRep() map[string]interface{}
}

type PRepStateImpl struct {
	name        string
	country     string
	city        string
	email       string
	website     string
	details     string
	p2pEndpoint string
	//registerBlockHeight uint
	//registerTxIndex     uint
	//iRep                common.HexInt
	//iRepBlockHeight     uint
	node *common.Address
	//bondList            []common.Address
}

func NewPRepState() PRepState {
	return &PRepStateImpl{}
}

func (p *PRepStateImpl) SetPRep(name string, email string, website string, country string,
	city string, details string, endpoint string, node module.Address) error {
	p.name = name
	p.email = email
	p.website = website
	p.country = country
	p.city = city
	p.details = details
	p.p2pEndpoint = endpoint
	p.node = node.(*common.Address)
	return nil
}

func (p *PRepStateImpl) GetPRep() map[string]interface{} {
	data := make(map[string]interface{})
	data["name"] = p.name
	data["email"] = p.email
	data["website"] = p.website
	data["country"] = p.country
	data["city"] = p.city
	data["details"] = p.details
	data["p2pEndpoint"] = p.p2pEndpoint
	data["node"] = p.node
	return data
}

func (p *PRepStateImpl) Bytes() []byte {
	if bs, err := codec.BC.MarshalToBytes(p); err != nil {
		panic(err)
	} else {
		return bs
	}
}

func (p *PRepStateImpl) SetBytes(bs []byte) error {
	_, err := codec.BC.UnmarshalFromBytes(bs, p)
	return err
}

func (p *PRepStateImpl) RLPEncodeSelf(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	if err := e2.EncodeMulti(p.name, p.country, p.city, p.email, p.website, p.details, p.p2pEndpoint, p.node); err != nil {
		return err
	}
	//p.registerBlockHeight,
	//p.registerTxIndex,
	//p.iRep,
	//p.bondList,
	return nil
}

func (p *PRepStateImpl) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}

	if _, err := d2.DecodeMulti(&p.name, &p.country, &p.city, &p.email, &p.website, &p.details, &p.p2pEndpoint, &p.node); err != nil {
		return errors.Wrap(err, "Fail to decode PRepSnapshot")
	}
	//&p.registerBlockHeight,
	//&p.registerTxIndex,
	//&p.iRep,
	//&p.bondList
	return nil
}

type PRepStatus struct {
	version      int
	state        int
	grade        int
	penalty      int
	delegated    common.HexInt
	bonded       common.HexInt
	vTotal       common.HexInt
	vFail        common.HexInt
	vFailCount   common.HexInt
	vPenaltyMask int
	lastState    int
	lastHeight   int
}
