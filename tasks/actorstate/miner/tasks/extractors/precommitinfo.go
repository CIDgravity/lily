package extractors

import (
	"context"

	"github.com/filecoin-project/sentinel-visor/metrics"
	"github.com/filecoin-project/sentinel-visor/model"
	"github.com/filecoin-project/sentinel-visor/tasks/actorstate/miner/tasks"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/label"
	"golang.org/x/xerrors"
)

func init() {
	tasks.Register(&MinerPreCommitInfo{}, ExtractMinerPreCommitInfo)
}

func ExtractMinerPreCommitInfo(ctx context.Context, ec *tasks.MinerStateExtractionContext) (model.Persistable, error) {
	if !ec.HasPreviousState() {
		return nil, nil
	}

	preCommitChanges, err := tasks.GetPreCommitDiff(ctx, ec)
	if err != nil {
		return nil, err
	}

	preCommitModel := MinerPreCommitInfoList{}
	for _, added := range preCommitChanges.Added {
		pcm := &MinerPreCommitInfo{
			Height:                 int64(ec.CurrTs.Height()),
			MinerID:                ec.Address.String(),
			SectorID:               uint64(added.Info.SectorNumber),
			StateRoot:              ec.CurrTs.ParentState().String(),
			SealedCID:              added.Info.SealedCID.String(),
			SealRandEpoch:          int64(added.Info.SealRandEpoch),
			ExpirationEpoch:        int64(added.Info.Expiration),
			PreCommitDeposit:       added.PreCommitDeposit.String(),
			PreCommitEpoch:         int64(added.PreCommitEpoch),
			DealWeight:             added.DealWeight.String(),
			VerifiedDealWeight:     added.VerifiedDealWeight.String(),
			IsReplaceCapacity:      added.Info.ReplaceCapacity,
			ReplaceSectorDeadline:  added.Info.ReplaceSectorDeadline,
			ReplaceSectorPartition: added.Info.ReplaceSectorPartition,
			ReplaceSectorNumber:    uint64(added.Info.ReplaceSectorNumber),
		}
		preCommitModel = append(preCommitModel, pcm)
	}
	return preCommitModel, nil
}

type MinerPreCommitInfo struct {
	Height    int64  `pg:",pk,notnull,use_zero"`
	MinerID   string `pg:",pk,notnull"`
	SectorID  uint64 `pg:",pk,use_zero"`
	StateRoot string `pg:",pk,notnull"`

	SealedCID       string `pg:",notnull"`
	SealRandEpoch   int64  `pg:",use_zero"`
	ExpirationEpoch int64  `pg:",use_zero"`

	PreCommitDeposit   string `pg:"type:numeric,notnull"`
	PreCommitEpoch     int64  `pg:",use_zero"`
	DealWeight         string `pg:"type:numeric,notnull"`
	VerifiedDealWeight string `pg:"type:numeric,notnull"`

	IsReplaceCapacity      bool
	ReplaceSectorDeadline  uint64 `pg:",use_zero"`
	ReplaceSectorPartition uint64 `pg:",use_zero"`
	ReplaceSectorNumber    uint64 `pg:",use_zero"`
}

type MinerPreCommitInfoV0 struct {
	//lint:ignore U1000 tableName is a convention used by go-pg
	tableName struct{} `pg:"miner_pre_commit_infos"`
	Height    int64    `pg:",pk,notnull,use_zero"`
	MinerID   string   `pg:",pk,notnull"`
	SectorID  uint64   `pg:",pk,use_zero"`
	StateRoot string   `pg:",pk,notnull"`

	SealedCID       string `pg:",notnull"`
	SealRandEpoch   int64  `pg:",use_zero"`
	ExpirationEpoch int64  `pg:",use_zero"`

	PreCommitDeposit   string `pg:",notnull"`
	PreCommitEpoch     int64  `pg:",use_zero"`
	DealWeight         string `pg:",notnull"`
	VerifiedDealWeight string `pg:",notnull"`

	IsReplaceCapacity      bool
	ReplaceSectorDeadline  uint64 `pg:",use_zero"`
	ReplaceSectorPartition uint64 `pg:",use_zero"`
	ReplaceSectorNumber    uint64 `pg:",use_zero"`
}

func (mpi *MinerPreCommitInfo) AsVersion(version model.Version) (interface{}, bool) {
	switch version.Major {
	case 0:
		if mpi == nil {
			return (*MinerPreCommitInfoV0)(nil), true
		}

		return &MinerPreCommitInfoV0{
			Height:                 mpi.Height,
			MinerID:                mpi.MinerID,
			SectorID:               mpi.SectorID,
			StateRoot:              mpi.StateRoot,
			SealedCID:              mpi.SealedCID,
			SealRandEpoch:          mpi.SealRandEpoch,
			ExpirationEpoch:        mpi.ExpirationEpoch,
			PreCommitDeposit:       mpi.PreCommitDeposit,
			PreCommitEpoch:         mpi.PreCommitEpoch,
			DealWeight:             mpi.DealWeight,
			VerifiedDealWeight:     mpi.VerifiedDealWeight,
			IsReplaceCapacity:      mpi.IsReplaceCapacity,
			ReplaceSectorDeadline:  mpi.ReplaceSectorDeadline,
			ReplaceSectorPartition: mpi.ReplaceSectorPartition,
			ReplaceSectorNumber:    mpi.ReplaceSectorNumber,
		}, true
	case 1:
		return mpi, true
	default:
		return nil, false
	}
}

func (mpi *MinerPreCommitInfo) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_pre_commit_infos"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	m, ok := mpi.AsVersion(version)
	if !ok {
		return xerrors.Errorf("MinerPreCommitInfo not supported for schema version %s", version)
	}

	return s.PersistModel(ctx, m)
}

type MinerPreCommitInfoList []*MinerPreCommitInfo

func (ml MinerPreCommitInfoList) Persist(ctx context.Context, s model.StorageBatch, version model.Version) error {
	ctx, span := global.Tracer("").Start(ctx, "MinerPreCommitInfoList.Persist", trace.WithAttributes(label.Int("count", len(ml))))
	defer span.End()

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.Table, "miner_pre_commit_infos"))
	stop := metrics.Timer(ctx, metrics.PersistDuration)
	defer stop()

	if len(ml) == 0 {
		return nil
	}

	if version.Major != 1 {
		// Support older versions, but in a non-optimal way
		for _, m := range ml {
			if err := m.Persist(ctx, s, version); err != nil {
				return err
			}
		}
		return nil
	}

	return s.PersistModel(ctx, ml)
}