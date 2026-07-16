package usageevent

import (
	"context"
	"sort"

	"github.com/seakee/cpa-statistics/apps/manager-server/internal/model"
)

// ModelPriceMigrationCandidates exposes only unambiguous legacy alias prices.
// The caller must still explicitly save the candidate under the billing model.
func (r *repository) ModelPriceMigrationCandidates(ctx context.Context, prices map[string]model.ModelPrice) ([]model.ModelPriceSyncCandidateSet, error) {
	rows, err := r.db.QueryContext(ctx, `select
		model,
		coalesce(nullif(resolved_model, ''), model) as billing_model,
		count(*)
		from usage_events
		where nullif(resolved_model, '') is not null and model <> resolved_model
		group by model, billing_model
		order by model, billing_model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type target struct {
		model string
		calls int64
	}
	mappings := map[string][]target{}
	for rows.Next() {
		var alias string
		var value target
		if err := rows.Scan(&alias, &value.model, &value.calls); err != nil {
			return nil, err
		}
		mappings[alias] = append(mappings[alias], value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sets := make([]model.ModelPriceSyncCandidateSet, 0)
	for alias, targets := range mappings {
		if len(targets) != 1 {
			continue
		}
		legacyPrice, ok := prices[alias]
		if !ok {
			continue
		}
		billingModel := targets[0].model
		if _, exists := prices[billingModel]; exists {
			continue
		}
		candidatePrice := legacyPrice
		candidatePrice.Source = "migration"
		candidatePrice.SourceModelID = alias
		sets = append(sets, model.ModelPriceSyncCandidateSet{
			Model: billingModel,
			Candidates: []model.ModelPriceSyncCandidate{{
				SourceModelID: alias,
				Score:         1,
				Reason:        "legacy_alias_price",
				Price:         candidatePrice,
			}},
		})
	}
	sort.Slice(sets, func(i, j int) bool { return sets[i].Model < sets[j].Model })
	return sets, nil
}
