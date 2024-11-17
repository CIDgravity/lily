// Code generated by: `make actors-gen`. DO NOT EDIT.
package power

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	builtin9 "github.com/filecoin-project/go-state-types/builtin"
	power9 "github.com/filecoin-project/go-state-types/builtin/v9/power"
	adt9 "github.com/filecoin-project/go-state-types/builtin/v9/util/adt"
	"github.com/filecoin-project/go-state-types/manifest"
	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/chain/actors/builtin"

	"github.com/filecoin-project/lotus/chain/actors"
)

var _ State = (*state9)(nil)

func load9(store adt.Store, root cid.Cid) (State, error) {
	out := state9{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type state9 struct {
	power9.State
	store adt.Store
}

func (s *state9) TotalLocked() (abi.TokenAmount, error) {
	return s.TotalPledgeCollateral, nil
}

func (s *state9) TotalPower() (Claim, error) {
	return Claim{
		RawBytePower:    s.TotalRawBytePower,
		QualityAdjPower: s.TotalQualityAdjPower,
	}, nil
}

// Committed power to the network. Includes miners below the minimum threshold.
func (s *state9) TotalCommitted() (Claim, error) {
	return Claim{
		RawBytePower:    s.TotalBytesCommitted,
		QualityAdjPower: s.TotalQABytesCommitted,
	}, nil
}

func (s *state9) MinerPower(addr address.Address) (Claim, bool, error) {
	claims, err := s.ClaimsMap()
	if err != nil {
		return Claim{}, false, err
	}
	var claim power9.Claim
	ok, err := claims.Get(abi.AddrKey(addr), &claim)
	if err != nil {
		return Claim{}, false, err
	}
	return Claim{
		RawBytePower:    claim.RawBytePower,
		QualityAdjPower: claim.QualityAdjPower,
	}, ok, nil
}

func (s *state9) MinerNominalPowerMeetsConsensusMinimum(a address.Address) (bool, error) {
	return s.State.MinerNominalPowerMeetsConsensusMinimum(s.store, a)
}

func (s *state9) TotalPowerSmoothed() (builtin.FilterEstimate, error) {
	return builtin.FilterEstimate(s.State.ThisEpochQAPowerSmoothed), nil
}

func (s *state9) MinerCounts() (uint64, uint64, error) {
	return uint64(s.State.MinerAboveMinPowerCount), uint64(s.State.MinerCount), nil
}

func (s *state9) RampStartEpoch() int64 {
	return 0
}
func (s *state9) RampDurationEpochs() uint64 {
	return 0
}

func (s *state9) ListAllMiners() ([]address.Address, error) {
	claims, err := s.ClaimsMap()
	if err != nil {
		return nil, err
	}

	var miners []address.Address
	err = claims.ForEach(nil, func(k string) error {
		a, err := address.NewFromBytes([]byte(k))
		if err != nil {
			return err
		}
		miners = append(miners, a)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return miners, nil
}

func (s *state9) ForEachClaim(cb func(miner address.Address, claim Claim) error) error {
	claims, err := s.ClaimsMap()
	if err != nil {
		return err
	}

	var claim power9.Claim
	return claims.ForEach(&claim, func(k string) error {
		a, err := address.NewFromBytes([]byte(k))
		if err != nil {
			return err
		}
		return cb(a, Claim{
			RawBytePower:    claim.RawBytePower,
			QualityAdjPower: claim.QualityAdjPower,
		})
	})
}

func (s *state9) ClaimsChanged(other State) (bool, error) {
	other9, ok := other.(*state9)
	if !ok {
		// treat an upgrade as a change, always
		return true, nil
	}
	return !s.State.Claims.Equals(other9.State.Claims), nil
}

func (s *state9) ClaimsMap() (adt.Map, error) {
	return adt9.AsMap(s.store, s.Claims, builtin9.DefaultHamtBitwidth)
}

func (s *state9) ClaimsMapBitWidth() int {

	return builtin9.DefaultHamtBitwidth

}

func (s *state9) ClaimsMapHashFunction() func(input []byte) []byte {

	return func(input []byte) []byte {
		res := sha256.Sum256(input)
		return res[:]
	}

}

func (s *state9) decodeClaim(val *cbg.Deferred) (Claim, error) {
	var ci power9.Claim
	if err := ci.UnmarshalCBOR(bytes.NewReader(val.Raw)); err != nil {
		return Claim{}, err
	}
	return fromV9Claim(ci), nil
}

func fromV9Claim(v9 power9.Claim) Claim {
	return Claim{
		RawBytePower:    v9.RawBytePower,
		QualityAdjPower: v9.QualityAdjPower,
	}
}

func (s *state9) ActorKey() string {
	return manifest.PowerKey
}

func (s *state9) ActorVersion() actorstypes.Version {
	return actorstypes.Version9
}

func (s *state9) Code() cid.Cid {
	code, ok := actors.GetActorCodeID(s.ActorVersion(), s.ActorKey())
	if !ok {
		panic(fmt.Errorf("didn't find actor %v code id for actor version %d", s.ActorKey(), s.ActorVersion()))
	}

	return code
}