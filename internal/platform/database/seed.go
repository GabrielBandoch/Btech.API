package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/btech/fleetcontrol-api/internal/platform/security"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedDevelopmentDatabase checks if the database has any organizations.
// If it's empty, it inserts a demo organization and creates 4 demo users (owner, admin, operator, viewer).
func SeedDevelopmentDatabase(
	ctx context.Context,
	pool *pgxpool.Pool,
	userRepo domain.UserRepository,
	orgRepo domain.OrganizationRepository,
	bcryptCost int,
	logger *slog.Logger,
) error {
	logger.Info("Checking if development database needs seeding...")

	// Check if any organization exists
	var count int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM organizations").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count organizations: %w", err)
	}

	if count > 0 {
		logger.Info("Database already seeded. Skipping seeding.")
		return nil
	}

	logger.Info("Seeding development database with demo tenant and accounts...")

	now := time.Now()
	orgID := "btech-demo-org-id"
	org := &domain.Organization{
		ID:        orgID,
		Name:      "BTech Demo Org",
		Slug:      "btech-demo-org",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := orgRepo.Create(ctx, org); err != nil {
		return fmt.Errorf("failed to seed organization: %w", err)
	}

	// Seed Pro subscription for the demo organization
	subID := "btech-demo-sub-id"
	startsAt := now
	endsAt := now.AddDate(10, 0, 0) // Valid for 10 years
	_, err = pool.Exec(ctx, `
		INSERT INTO subscriptions (id, organization_id, plan_id, status, starts_at, ends_at, trial_ends_at, canceled_at, created_at, updated_at)
		VALUES ($1, $2, 'plan-pro-id', 'active', $3, $4, NULL, NULL, $5, $5)
		ON CONFLICT (organization_id) DO NOTHING`,
		subID, orgID, startsAt, endsAt, now)
	if err != nil {
		return fmt.Errorf("failed to seed subscription: %w", err)
	}

	demoUsers := []struct {
		Email string
		Name  string
		Role  string
	}{
		{Email: "owner@btech.com", Name: "Demo Owner", Role: "owner"},
		{Email: "admin@btech.com", Name: "Demo Admin", Role: "admin"},
		{Email: "operator@btech.com", Name: "Demo Operator", Role: "operator"},
		{Email: "viewer@btech.com", Name: "Demo Viewer", Role: "viewer"},
	}

	passwordHash, err := security.HashPassword("Password123!", bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash seed password: %w", err)
	}

	for _, du := range demoUsers {
		userID := du.Role + "-demo-id"
		user := &domain.User{
			ID:             userID,
			Name:           du.Name,
			Email:          du.Email,
			PasswordHash:   passwordHash,
			Role:           du.Role,
			OrganizationID: orgID,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		if err := userRepo.Create(ctx, user); err != nil {
			return fmt.Errorf("failed to seed user %s: %w", du.Email, err)
		}

		orgUser := &domain.OrganizationUser{
			ID:             du.Role + "-mapping-id",
			OrganizationID: orgID,
			UserID:         userID,
			Role:           du.Role,
			CreatedAt:      now,
		}

		if err := orgRepo.CreateOrganizationUser(ctx, orgUser); err != nil {
			return fmt.Errorf("failed to seed organization user mapping for %s: %w", du.Email, err)
		}

		logger.Info("Successfully seeded demo user account", slog.String("email", du.Email), slog.String("role", du.Role))
	}

	// 5. Seed Drivers
	drivers := []struct {
		ID               string
		Name             string
		Avatar           string
		Status           string
		Score            int
		TripsCount       int
		IncidentsCount   int
		NextScale        string
		Role             string
		LicenseExpiry    string
		ToxicologyExpiry string
		TrainingExpiry   string
	}{
		{
			ID:               "DRV-002",
			Name:             "Carlos Alberto",
			Avatar:           "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiNWKZN5WSOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
			Status:           "active",
			Score:            96,
			TripsCount:       28,
			IncidentsCount:   0,
			NextScale:        "Livre agora",
			Role:             "Operadora Urbana",
			LicenseExpiry:    "2028-05-14",
			ToxicologyExpiry: "2026-09-10",
			TrainingExpiry:   "2027-02-18",
		},
		{
			ID:               "DRV-003",
			Name:             "Marcos Souza",
			Avatar:           "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiGEOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
			Status:           "inactive",
			Score:            68,
			TripsCount:       34,
			IncidentsCount:   6,
			NextScale:        "Escalado em 8h",
			Role:             "Motorista Interestadual",
			LicenseExpiry:    "2026-05-28",
			ToxicologyExpiry: "2026-06-10",
			TrainingExpiry:   "2026-07-01",
		},
		{
			ID:               "DRV-004",
			Name:             "João Santos",
			Avatar:           "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiNWKZN5WSOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
			Status:           "blocked",
			Score:            72,
			TripsCount:       19,
			IncidentsCount:   2,
			NextScale:        "Férias",
			Role:             "Motorista Regional",
			LicenseExpiry:    "2029-01-20",
			ToxicologyExpiry: "2026-12-15",
			TrainingExpiry:   "2026-10-30",
		},
	}

	for _, d := range drivers {
		_, err = pool.Exec(ctx, `
			INSERT INTO drivers (
				id, organization_id, name, avatar, status, score, trips_count, incidents_count,
				next_scale, role, license_expiry, toxicology_expiry, training_expiry, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14)
			ON CONFLICT (id) DO NOTHING`,
			d.ID, orgID, d.Name, d.Avatar, d.Status, d.Score, d.TripsCount, d.IncidentsCount,
			d.NextScale, d.Role, d.LicenseExpiry, d.ToxicologyExpiry, d.TrainingExpiry, now,
		)
		if err != nil {
			return fmt.Errorf("failed to seed driver %s: %w", d.ID, err)
		}
	}

	// 6. Seed Vehicles
	vehicles := []struct {
		ID      string
		Placa   string
		Brand   string
		Model   string
		Year    int
		Type    string
		Mileage int
		Status  string
	}{
		{ID: "VH-001", Placa: "BRA-2E19", Brand: "Mercedes-Benz", Model: "Actros", Year: 2022, Type: "Truck", Mileage: 125000, Status: "disponivel"},
		{ID: "VH-002", Placa: "KGB-8840", Brand: "Volvo", Model: "FH 540", Year: 2023, Type: "Truck", Mileage: 98000, Status: "disponivel"},
		{ID: "VH-003", Placa: "MLX-9018", Brand: "Scania", Model: "R 450", Year: 2021, Type: "Truck", Mileage: 182000, Status: "manutencao"},
	}

	for _, v := range vehicles {
		_, err = pool.Exec(ctx, `
			INSERT INTO vehicles (
				id, organization_id, placa, brand, model, year, type, mileage, status, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
			ON CONFLICT (id) DO NOTHING`,
			v.ID, orgID, v.Placa, v.Brand, v.Model, v.Year, v.Type, v.Mileage, v.Status, now,
		)
		if err != nil {
			return fmt.Errorf("failed to seed vehicle %s: %w", v.ID, err)
		}
	}

	// 7. Seed Trips
	trips := []struct {
		ID                  string
		Origin              string
		Destination         string
		Status              string
		DriverName          string
		DriverAvatar        string
		VehiclePlaca        string
		VehicleModel        string
		CargoType           string
		CargoValue          float64
		CargoWeight         int
		TemperatureRequired string
		EstimatedTime       string
		Speed               int
		FuelLevel           int
		LastSignalTime      string
		CurrentLocation     string
	}{
		{
			ID:             "TR-990",
			Origin:          "CD São Paulo - SP",
			Destination:     "CD Rio de Janeiro - RJ",
			Status:          "em_transito",
			DriverName:      "Carlos Alberto",
			DriverAvatar:    "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiNWKZN5WSOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
			VehiclePlaca:    "BRA-2E19",
			VehicleModel:    "Mercedes-Benz Actros",
			CargoType:       "Eletrônicos Premium",
			CargoValue:      450000.00,
			CargoWeight:     8500,
			EstimatedTime:   "12:30",
			Speed:           82,
			FuelLevel:       75,
			LastSignalTime:  "Faz 1 min",
			CurrentLocation: "Resende - RJ",
		},
		{
			ID:                  "VT-422",
			Origin:              "CD Curitiba - PR",
			Destination:         "CD Porto Alegre - RS",
			Status:              "em_transito",
			DriverName:          "Marcos Souza",
			DriverAvatar:        "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiGEOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
			VehiclePlaca:        "KGB-8840",
			VehicleModel:        "Volvo FH 540",
			CargoType:           "Vacinas Climatizadas",
			CargoValue:          1200000.00,
			CargoWeight:         4200,
			TemperatureRequired: "-18°C a -22°C",
			EstimatedTime:       "14:45",
			Speed:               78,
			FuelLevel:           64,
			LastSignalTime:      "Faz 3 min",
			CurrentLocation:     "Joinville - SC",
		},
		{
			ID:             "TR-8820",
			Origin:          "CD Belo Horizonte - MG",
			Destination:     "CD Salvador - BA",
			Status:          "atrasada",
			DriverName:      "João Santos",
			DriverAvatar:    "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiNWKZN5WSOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
			VehiclePlaca:    "MLX-9018",
			VehicleModel:    "Scania R 450",
			CargoType:       "Carga Seca Geral",
			CargoValue:      150000.00,
			CargoWeight:     12000,
			EstimatedTime:   "Atrasado (+45m)",
			Speed:           0,
			FuelLevel:       22,
			LastSignalTime:  "Faz 12 min",
			CurrentLocation: "Teófilo Otoni - MG",
		},
	}

	for _, t := range trips {
		_, err = pool.Exec(ctx, `
			INSERT INTO trips (
				id, organization_id, origin, destination, status, driver_name, driver_avatar,
				vehicle_placa, vehicle_model, cargo_type, cargo_value, cargo_weight,
				temperature_required, estimated_time, speed, fuel_level, last_signal_time,
				current_location, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $19)
			ON CONFLICT (id) DO NOTHING`,
			t.ID, orgID, t.Origin, t.Destination, t.Status, t.DriverName, t.DriverAvatar,
			t.VehiclePlaca, t.VehicleModel, t.CargoType, t.CargoValue, t.CargoWeight,
			t.TemperatureRequired, t.EstimatedTime, t.Speed, t.FuelLevel, t.LastSignalTime,
			t.CurrentLocation, now,
		)
		if err != nil {
			return fmt.Errorf("failed to seed trip %s: %w", t.ID, err)
		}
	}

	// 8. Seed Trip Checkpoints
	checkpoints := []struct {
		ID        string
		TripID    string
		Sequence  int
		Name      string
		PlannedAt *string
		ArrivedAt *string
	}{
		{ID: "CK-TR-990-1", TripID: "TR-990", Sequence: 1, Name: "CD São Paulo - SP", PlannedAt: nil, ArrivedAt: ptrString("2026-05-25T08:00:00Z")},
		{ID: "CK-TR-990-2", TripID: "TR-990", Sequence: 2, Name: "Pedágio Jacareí", PlannedAt: nil, ArrivedAt: ptrString("2026-05-25T09:30:00Z")},
		{ID: "CK-TR-990-3", TripID: "TR-990", Sequence: 3, Name: "Checkpoint Resende", PlannedAt: ptrString("12:15"), ArrivedAt: nil},
		{ID: "CK-TR-990-4", TripID: "TR-990", Sequence: 4, Name: "CD Rio de Janeiro - RJ", PlannedAt: ptrString("15:00"), ArrivedAt: nil},

		{ID: "CK-VT-422-1", TripID: "VT-422", Sequence: 1, Name: "CD Curitiba - PR", PlannedAt: nil, ArrivedAt: ptrString("2026-05-25T07:15:00Z")},
		{ID: "CK-VT-422-2", TripID: "VT-422", Sequence: 2, Name: "Pedágio Garuva", PlannedAt: nil, ArrivedAt: ptrString("2026-05-25T08:45:00Z")},
		{ID: "CK-VT-422-3", TripID: "VT-422", Sequence: 3, Name: "CD Porto Alegre - RS", PlannedAt: ptrString("16:30"), ArrivedAt: nil},

		{ID: "CK-TR-8820-1", TripID: "TR-8820", Sequence: 1, Name: "CD Belo Horizonte - MG", PlannedAt: nil, ArrivedAt: ptrString("2026-05-24T22:00:00Z")},
		{ID: "CK-TR-8820-2", TripID: "TR-8820", Sequence: 2, Name: "Teófilo Otoni - MG", PlannedAt: nil, ArrivedAt: ptrString("2026-05-25T04:30:00Z")},
		{ID: "CK-TR-8820-3", TripID: "TR-8820", Sequence: 3, Name: "CD Salvador - BA", PlannedAt: ptrString("18:00"), ArrivedAt: nil},
	}

	for _, cp := range checkpoints {
		_, err = pool.Exec(ctx, `
			INSERT INTO trip_checkpoints (
				id, trip_id, sequence, name, latitude, longitude, planned_at, arrived_at, created_at
			) VALUES ($1, $2, $3, $4, NULL, NULL, $5, $6, $7)
			ON CONFLICT (id) DO NOTHING`,
			cp.ID, cp.TripID, cp.Sequence, cp.Name, cp.PlannedAt, cp.ArrivedAt, now,
		)
		if err != nil {
			return fmt.Errorf("failed to seed checkpoint %s: %w", cp.ID, err)
		}
	}

	// 9. Seed Incidents
	incidents := []struct {
		ID           string
		TripID       string
		VehiclePlaca string
		DriverName   string
		Type         string
		Severity     string
		Description  string
		Timestamp    string
		Location     string
		Status       string
	}{
		{
			ID:           "INC-001",
			TripID:       "TR-8820",
			VehiclePlaca: "MLX-9018",
			DriverName:   "João Santos",
			Type:         "Atraso Logístico",
			Severity:     "high",
			Description:  "Veículo parado em congestionamento na rodovia devido a obras na pista.",
			Timestamp:    "Faz 12 min",
			Location:     "Teófilo Otoni - MG",
			Status:       "revisao",
		},
		{
			ID:           "INC-002",
			TripID:       "VT-422",
			VehiclePlaca: "KGB-8840",
			DriverName:   "Marcos Souza",
			Type:         "Desvio de Temperatura",
			Severity:     "critical",
			Description:  "Alerta térmico: Temperatura do baú subiu para -12°C. Requer ação preventiva.",
			Timestamp:    "Faz 3 min",
			Location:     "Joinville - SC",
			Status:       "investigando",
		},
	}

	for _, i := range incidents {
		_, err = pool.Exec(ctx, `
			INSERT INTO incidents (
				id, organization_id, trip_id, vehicle_placa, driver_name, type, severity,
				description, timestamp, location, status, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12)
			ON CONFLICT (id) DO NOTHING`,
			i.ID, orgID, i.TripID, i.VehiclePlaca, i.DriverName, i.Type, i.Severity,
			i.Description, i.Timestamp, i.Location, i.Status, now,
		)
		if err != nil {
			return fmt.Errorf("failed to seed incident %s: %w", i.ID, err)
		}
	}

	// 10. Seed Maintenance Suppliers
	suppliers := []struct {
		ID      string
		Name    string
		Phone   string
		Email   string
		Address string
	}{
		{ID: "MS-001", Name: "Oficina Mecânica TruckCenter", Phone: "(11) 99999-8888", Email: "contato@truckcenter.com", Address: "Av. Marginal Pinheiros, 1200 - São Paulo, SP"},
		{ID: "MS-002", Name: "AutoPeças e Serviços Diesel", Phone: "(11) 98888-7777", Email: "servicos@dieselserv.com", Address: "Rua das Oficinas, 450 - Guarulhos, SP"},
	}

	for _, s := range suppliers {
		_, err = pool.Exec(ctx, `
			INSERT INTO maintenance_suppliers (
				id, organization_id, name, phone, email, address, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
			ON CONFLICT (id) DO NOTHING`,
			s.ID, orgID, s.Name, s.Phone, s.Email, s.Address, now,
		)
		if err != nil {
			return fmt.Errorf("failed to seed maintenance supplier %s: %w", s.ID, err)
		}
	}

	// 11. Seed Maintenance Plans
	plans := []struct {
		ID             string
		VehicleID      string
		Name           string
		IntervalKM     int
		IntervalMonths int
		LastKM         int
		LastDate       time.Time
		NextKM         int
		NextDate       time.Time
	}{
		{
			ID: "MP-001", VehicleID: "VH-001", Name: "Troca de Óleo e Filtros",
			IntervalKM: 10000, IntervalMonths: 6,
			LastKM: 120000, LastDate: now.AddDate(0, -3, 0),
			NextKM: 130000, NextDate: now.AddDate(0, 3, 0),
		},
		{
			ID: "MP-002", VehicleID: "VH-002", Name: "Revisão Geral Preventiva",
			IntervalKM: 40000, IntervalMonths: 12,
			LastKM: 80000, LastDate: now.AddDate(0, -11, 15),
			NextKM: 120000, NextDate: now.AddDate(0, 0, 15), // Close to next date (within 30 days)
		},
		{
			ID: "MP-003", VehicleID: "VH-003", Name: "Manutenção de Freios",
			IntervalKM: 20000, IntervalMonths: 6,
			LastKM: 180000, LastDate: now.AddDate(0, -5, 0),
			NextKM: 200000, NextDate: now.AddDate(0, 1, 0),
		},
	}

	for _, p := range plans {
		_, err = pool.Exec(ctx, `
			INSERT INTO maintenance_plans (
				id, organization_id, vehicle_id, name, interval_km, interval_months,
				last_maintenance_km, last_maintenance_date, next_due_km, next_due_date, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
			ON CONFLICT (id) DO NOTHING`,
			p.ID, orgID, p.VehicleID, p.Name, p.IntervalKM, p.IntervalMonths,
			p.LastKM, p.LastDate, p.NextKM, p.NextDate, now,
		)
		if err != nil {
			return fmt.Errorf("failed to seed maintenance plan %s: %w", p.ID, err)
		}
	}

	// 12. Seed Maintenances
	maintenances := []struct {
		ID                string
		VehicleID         string
		PlanID            *string
		SupplierID        *string
		Type              string
		Priority          string
		Status            string
		Date              time.Time
		OdometerAtService int
		DowntimeHours     float64
		Cost              float64
		Description       string
		Attachments       []string
	}{
		{
			ID: "MNT-001", VehicleID: "VH-001", PlanID: ptrString("MP-001"), SupplierID: ptrString("MS-001"),
			Type: "preventive", Priority: "medium", Status: "completed",
			Date: now.AddDate(0, -3, 0), OdometerAtService: 120000, DowntimeHours: 4.5, Cost: 1250.00,
			Description: "Troca de óleo do motor, filtro de óleo e filtro de combustível efetuados com sucesso.",
			Attachments: []string{"https://example.com/invoice-001.pdf", "https://example.com/photo-001.jpg"},
		},
		{
			ID: "MNT-002", VehicleID: "VH-003", PlanID: nil, SupplierID: ptrString("MS-002"),
			Type: "corrective", Priority: "critical", Status: "in_progress",
			Date: now, OdometerAtService: 182000, DowntimeHours: 24.0, Cost: 4800.00,
			Description: "Quebra inesperada da suspensão dianteira durante viagem. Veículo guinchado até a oficina.",
			Attachments: []string{"https://example.com/service-order-002.pdf"},
		},
		{
			ID: "MNT-003", VehicleID: "VH-002", PlanID: ptrString("MP-002"), SupplierID: ptrString("MS-001"),
			Type: "preventive", Priority: "high", Status: "scheduled",
			Date: now.AddDate(0, 0, 15), OdometerAtService: 120000, DowntimeHours: 8.0, Cost: 3500.00,
			Description: "Agendamento de revisão geral de 120.000km conforme plano preventivo do veículo.",
			Attachments: []string{},
		},
	}

	for _, m := range maintenances {
		var planIDVal, supplierIDVal interface{}
		if m.PlanID != nil {
			planIDVal = *m.PlanID
		}
		if m.SupplierID != nil {
			supplierIDVal = *m.SupplierID
		}

		_, err = pool.Exec(ctx, `
			INSERT INTO maintenances (
				id, organization_id, vehicle_id, maintenance_plan_id, supplier_id,
				type, priority, status, date, odometer_at_service, downtime_hours, cost, description, attachments, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $15)
			ON CONFLICT (id) DO NOTHING`,
			m.ID, orgID, m.VehicleID, planIDVal, supplierIDVal,
			m.Type, m.Priority, m.Status, m.Date, m.OdometerAtService, m.DowntimeHours, m.Cost, m.Description, m.Attachments, now,
		)
		if err != nil {
			return fmt.Errorf("failed to seed maintenance %s: %w", m.ID, err)
		}
	}

	// 13. Seed Maintenance Alerts (VH-002 is close to date limit for MP-002)
	_, err = pool.Exec(ctx, `
		INSERT INTO maintenance_alerts (
			id, organization_id, vehicle_id, maintenance_plan_id, type, title, message, status, created_at, updated_at
		) VALUES ($1, $2, 'VH-002', 'MP-002', 'date_due', 'Revisão Geral Preventiva Próxima do Vencimento', 
		          'O veículo VH-002 está próximo da data limite prevista para a revisão Geral Preventiva (vence em menos de 30 dias).',
		          'active', $3, $3)
		ON CONFLICT (id) DO NOTHING`,
		"MNT-ALT-001", orgID, now,
	)
	if err != nil {
		return fmt.Errorf("failed to seed maintenance alert: %w", err)
	}

	logger.Info("Development database seeding completed successfully.")
	return nil
}

func ptrString(s string) *string {
	return &s
}
