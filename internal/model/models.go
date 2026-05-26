package model

type Checkpoint struct {
	Name        string `json:"name"`
	PlannedTime string `json:"plannedTime,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
	Type        string `json:"type"` // origin, checkpoint, destination, current
}

type Trip struct {
	ID                  string       `json:"id"`
	Origin              string       `json:"origin"`
	Destination         string       `json:"destination"`
	Status              string       `json:"status"` // em_transito, iniciada, atrasada, finalizada
	DriverName          string       `json:"driverName"`
	DriverAvatar        string       `json:"driverAvatar"`
	VehiclePlaca        string       `json:"vehiclePlaca"`
	VehicleModel        string       `json:"vehicleModel"`
	CargoType           string       `json:"cargoType"`
	CargoValue          float64      `json:"cargoValue"`
	CargoWeight         int          `json:"cargoWeight"`
	TemperatureRequired string       `json:"temperatureRequired,omitempty"`
	EstimatedTime       string       `json:"estimatedTime"`
	Speed               int          `json:"speed"`
	FuelLevel           int          `json:"fuelLevel"`
	LastSignalTime      string       `json:"lastSignalTime"`
	CurrentLocation     string       `json:"currentLocation"`
	Checkpoints         []Checkpoint `json:"checkpoints"`
}

type Driver struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Avatar           string `json:"avatar"`
	Status           string `json:"status"` // online, em_rota, descanso, offline
	Score            int    `json:"score"`
	TripsCount       int    `json:"tripsCount"`
	IncidentsCount   int    `json:"incidentsCount"`
	NextScale        string `json:"nextScale,omitempty"`
	Role             string `json:"role"`
	LicenseExpiry    string `json:"licenseExpiry"`
	ToxicologyExpiry string `json:"toxicologyExpiry"`
	TrainingExpiry   string `json:"trainingExpiry"`
}

type Incident struct {
	ID           string `json:"id"`
	TripID       string `json:"tripId,omitempty"`
	VehiclePlaca string `json:"vehiclePlaca"`
	DriverName   string `json:"driverName"`
	Type         string `json:"type"`     // falha_mecanica, desvio_rota, acidente, atraso, area_insegura
	Severity     string `json:"severity"` // critica, media, baixa
	Description  string `json:"description"`
	Timestamp    string `json:"timestamp"`
	Location     string `json:"location"`
	Status       string `json:"status"` // aberta, investigando, resolvida
}
