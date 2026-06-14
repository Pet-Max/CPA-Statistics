package model

import "github.com/seakee/cpa-statistics/apps/manager-server/internal/usage"

type UsageEvent = usage.Event

type InsertResult struct {
	Inserted int `json:"inserted"`
	Skipped  int `json:"skipped"`
}
