package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrPlanNotFound = errors.New("plan not found")
)

type Plan struct {
	ID           string    `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	MonthlyPrice float64   `json:"monthlyPrice"`
	YearlyPrice  float64   `json:"yearlyPrice"`
	IsActive     bool      `json:"isActive"`
	CreatedAt    time.Time `json:"createdAt"`
}

type PlanEntitlement struct {
	PlanID string `json:"planId"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

type PlanRepository interface {
	GetByID(ctx context.Context, id string) (*Plan, error)
	GetByCode(ctx context.Context, code string) (*Plan, error)
	ListActive(ctx context.Context) ([]*Plan, error)
	GetEntitlements(ctx context.Context, planID string) ([]*PlanEntitlement, error)
	Create(ctx context.Context, plan *Plan) error
	CreateEntitlement(ctx context.Context, ent *PlanEntitlement) error
}
