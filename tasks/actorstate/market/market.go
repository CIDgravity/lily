package market

import (
	"context"
	"unicode/utf8"

	"github.com/filecoin-project/lotus/chain/types"
	"go.opentelemetry.io/otel"
	"golang.org/x/text/runes"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/lily/chain/actors/adt"
	"github.com/filecoin-project/lily/tasks/actorstate"

	market "github.com/filecoin-project/lily/chain/actors/builtin/market"

	"github.com/filecoin-project/lily/model"
)

// was services/processor/tasks/market/market.go

// StorageMarketExtractor extracts market actor state
type StorageMarketExtractor struct{}

type MarketStateExtractionContext struct {
	PrevState market.State
	PrevTs    *types.TipSet

	CurrActor *types.Actor
	CurrState market.State
	CurrTs    *types.TipSet

	Store adt.Store
}

func NewMarketStateExtractionContext(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (*MarketStateExtractionContext, error) {
	curState, err := market.Load(node.Store(), &a.Actor)
	if err != nil {
		return nil, xerrors.Errorf("loading current market state: %w", err)
	}

	prevTipset := a.Current
	prevState := curState
	if a.Current.Height() != 0 {
		prevTipset = a.Executed

		prevActor, err := node.Actor(ctx, a.Address, a.Executed.Key())
		if err != nil {
			return nil, xerrors.Errorf("loading previous market actor state at tipset %s epoch %d: %w", a.Executed.Key(), a.Current.Height(), err)
		}

		prevState, err = market.Load(node.Store(), prevActor)
		if err != nil {
			return nil, xerrors.Errorf("loading previous market actor state: %w", err)
		}

	}
	return &MarketStateExtractionContext{
		PrevState: prevState,
		PrevTs:    prevTipset,
		CurrActor: &a.Actor,
		CurrState: curState,
		CurrTs:    a.Current,
		Store:     node.Store(),
	}, nil
}

func (m *MarketStateExtractionContext) IsGenesis() bool {
	return m.CurrTs.Height() == 0
}

func (m StorageMarketExtractor) Extract(ctx context.Context, a actorstate.ActorInfo, node actorstate.ActorStateAPI) (model.Persistable, error) {
	ctx, span := otel.Tracer("").Start(ctx, "StorageMarketExtractor.Extract")
	defer span.End()
	if span.IsRecording() {
		span.SetAttributes(a.Attributes()...)
	}

	dealStateModel, err := DealStateExtractor{}.Extract(ctx, a, node)
	if err != nil {
		return nil, xerrors.Errorf("extracting market deal state changes: %w", err)
	}

	dealProposalModel, err := DealProposalExtractor{}.Extract(ctx, a, node)
	if err != nil {
		return nil, xerrors.Errorf("extracting market proposal changes: %w", err)
	}

	return &model.PersistableList{
		dealProposalModel,
		dealStateModel,
	}, nil
}

// SanitizeLabel ensures that s is a valid utf8 string by replacing any ill formed bytes with a replacement character.
func SanitizeLabel(s string) string {
	if utf8.ValidString(s) {
		return s
	}

	tr := runes.ReplaceIllFormed()
	return tr.String(s)
}