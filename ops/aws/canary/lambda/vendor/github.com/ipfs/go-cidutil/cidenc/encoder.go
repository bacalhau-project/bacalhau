package cidenc

import (
	cid "github.com/ipfs/go-cid"
	mbase "github.com/multiformats/go-multibase"
)

// Encoder is a basic Encoder that will encode CIDs using a specified
// base and optionally upgrade a CIDv0 to CIDv1
type Encoder struct {
	Base    mbase.Encoder // The multibase to use
	Upgrade bool          // If true upgrade CIDv0 to CIDv1 when encoding
}

// Default return a new default encoder
func Default() Encoder {
	return Encoder{Base: mbase.MustNewEncoder(mbase.Base32)}
}

// Encode encodes the cid using the parameters of the Encoder
func (enc Encoder) Encode(c cid.Cid) string {
	if enc.Upgrade && c.Version() == 0 {
		c = cid.NewCidV1(c.Type(), c.Hash())
	}
	return c.Encode(enc.Base)
}

// Recode reencodes the cid string to match the parameters of the
// encoder
func (enc Encoder) Recode(v string) (string, error) {
	skip, err := enc.noopRecode(v)
	if skip || err != nil {
		return v, err
	}

	c, err := cid.Decode(v)
	if err != nil {
		return v, err
	}

	return enc.Encode(c), nil
}

func (enc Encoder) noopRecode(v string) (bool, error) {
	if len(v) < 2 {
		return false, cid.ErrCidTooShort
	}
	ver := cidVer(v)
	skip := ver == 0 && !enc.Upgrade || ver == 1 && v[0] == byte(enc.Base.Encoding())
	return skip, nil
}

func cidVer(v string) int {
	if len(v) == 46 && v[:2] == "Qm" {
		return 0
	} else {
		return 1
	}
}
