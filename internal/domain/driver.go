package domain

import "context"

type Driver struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	LicensePlate string `json:"license_plate"`
}

type DriverRepository interface {
	GetAll(ctx context.Context) ([]Driver, error)
}
