package actorstate

import (
	"context"

	"github.com/filecoin-project/lotus/chain/actors/builtin/reward"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"go.opentelemetry.io/otel/api/global"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	rewardmodel "github.com/filecoin-project/sentinel-visor/model/actors/reward"
)

// was services/processor/tasks/reward/reward.go

// RewardExtractor extracts reward actor state
type RewardExtractor struct{}

func init() {
	Register(builtin.RewardActorCodeID, RewardExtractor{})
}

func (RewardExtractor) Extract(ctx context.Context, a ActorInfo, node ActorStateAPI) (model.Persistable, error) {
	ctx, span := global.Tracer("").Start(ctx, "RewardExtractor")
	defer span.End()

	stop := metrics.Timer(ctx, metrics.ProcessingDuration)
	defer stop()

	rewardActor, err := node.StateGetActor(ctx, builtin.RewardActorAddr, a.TipSet)
	if err != nil {
		return nil, err
	}

	rstate, err := reward.Load(node.Store(), rewardActor)
	if err != nil {
		return nil, err
	}

	csbaseline, err := rstate.CumsumBaseline()
	if err != nil {
		return nil, err
	}

	csrealized, err := rstate.CumsumRealized()
	if err != nil {
		return nil, err
	}
	effectiveBaselinePower, err := rstate.EffectiveBaselinePower()
	if err != nil {
		return nil, err
	}

	thisBaslinePower, err := rstate.ThisEpochBaselinePower()
	if err != nil {
		return nil, err
	}

	thisRewardSmoothed, err := rstate.ThisEpochRewardSmoothed()
	if err != nil {
		return nil, err
	}

	totalMinedReward, err := rstate.TotalStoragePowerReward()
	if err != nil {
		return nil, err
	}

	thisReward, err := rstate.ThisEpochReward()
	if err != nil {
		return nil, err
	}

	networkTime, err := rstate.EffectiveNetworkTime()
	if err != nil {
		return nil, err
	}

	return &rewardmodel.ChainReward{
		StateRoot:                         a.ParentStateRoot.String(),
		CumSumBaseline:                    csbaseline.String(),
		CumSumRealized:                    csrealized.String(),
		EffectiveBaselinePower:            effectiveBaselinePower.String(),
		NewBaselinePower:                  thisBaslinePower.String(),
		NewRewardSmoothedPositionEstimate: thisRewardSmoothed.PositionEstimate.String(),
		NewRewardSmoothedVelocityEstimate: thisRewardSmoothed.VelocityEstimate.String(),
		TotalMinedReward:                  totalMinedReward.String(),
		NewReward:                         thisReward.String(),
		EffectiveNetworkTime:              int64(networkTime),
	}, nil
}