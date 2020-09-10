package miner

import (
	"encoding/binary"
	"github.com/spacemeshos/go-spacemesh/common/types"
)

// EpochBeaconProvider holds all the dependencies for generating an epoch beacon. There are currently none.
type EpochBeaconProvider struct{}

// GetBeacon returns a beacon given an epoch ID. The current implementation returns the epoch ID in byte format.
func (p *EpochBeaconProvider) GetBeacon(epochNumber types.EpochID) []byte {
	// Note: the EpochID is only 32 bits, so this will only fill in 4 of these bytes.
	ret := make([]byte, 32)
	binary.LittleEndian.PutUint64(ret, uint64(epochNumber))
	return ret
}
