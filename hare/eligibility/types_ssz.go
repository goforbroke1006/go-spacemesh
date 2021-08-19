// Code generated by fastssz. DO NOT EDIT.
package eligibility

import (
	ssz "github.com/ferranbt/fastssz"
	"github.com/spacemeshos/go-spacemesh/common/types"
)

// MarshalSSZ ssz marshals the vrfMessage object
func (v *vrfMessage) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(v)
}

// MarshalSSZTo ssz marshals the vrfMessage object to a target array
func (v *vrfMessage) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	dst = buf

	// Field (0) 'Beacon'
	dst = ssz.MarshalUint32(dst, v.Beacon)

	// Field (1) 'Round'
	dst = ssz.MarshalUint32(dst, v.Round)

	// Field (2) 'Layer'
	if dst, err = v.Layer.MarshalSSZTo(dst); err != nil {
		return
	}

	return
}

// UnmarshalSSZ ssz unmarshals the vrfMessage object
func (v *vrfMessage) UnmarshalSSZ(buf []byte) error {
	var err error
	size := uint64(len(buf))
	if size != 12 {
		return ssz.ErrSize
	}

	// Field (0) 'Beacon'
	v.Beacon = ssz.UnmarshallUint32(buf[0:4])

	// Field (1) 'Round'
	v.Round = ssz.UnmarshallUint32(buf[4:8])

	// Field (2) 'Layer'
	if err = v.Layer.UnmarshalSSZ(buf[8:12]); err != nil {
		return err
	}

	return err
}

// SizeSSZ returns the ssz encoded size in bytes for the vrfMessage object
func (v *vrfMessage) SizeSSZ() (size int) {
	size = 12
	return
}
