package models

import "time"

type Address struct {
	Province string
	District *string // nil = absent
	Ward     *string // nil = absent
}

type ResolvedAddress struct {
	Province string // resolved province (new if post-merger)
	District string // original district (pre-merger only)
	Ward     string // resolved ward
	Region   string // from province→region map
	Depot    string // old province name (for distance rule)
}

type Lane string

const (
	LaneLocal        Lane = "LOCAL"
	LaneSameProvince Lane = "SAME_PROVINCE"
	LaneSameRegion   Lane = "SAME_REGION"
	LaneCrossRegion  Lane = "CROSS_REGION"
)

type Vehicle string

const (
	VehicleMotorbike Vehicle = "motorbike"
	VehicleTruck     Vehicle = "truck"
)

func VehicleForLane(lane Lane) Vehicle {
	switch lane {
	case LaneLocal, LaneSameProvince:
		return VehicleMotorbike
	default:
		return VehicleTruck
	}
}

type RejectionReason string

const (
	ReasonNoPriceBoard     RejectionReason = "NO_PRICE_BOARD"
	ReasonUnknownProvince  RejectionReason = "UNKNOWN_PROVINCE"
	ReasonMissingDistrict  RejectionReason = "MISSING_DISTRICT"
	ReasonBadWeight        RejectionReason = "BAD_WEIGHT"
	ReasonBadDimensions    RejectionReason = "BAD_DIMENSIONS"
	ReasonUnknownWard      RejectionReason = "UNKNOWN_WARD"
	ReasonUnknownNewAddr   RejectionReason = "UNKNOWN_NEW_ADDRESS"
	ReasonNoRoute          RejectionReason = "NO_ROUTE"
	ReasonTooHeavy         RejectionReason = "TOO_HEAVY"
)

// LaneConfig holds fee settings per lane
type LaneConfig struct {
	FirstBlock    int `json:"first_block"`
	FirstBlockG   int `json:"first_block_g"`
	PerExtraBlock int `json:"per_extra_block"`
	ExtraBlockG   int `json:"extra_block_g"`
	TransitDays   int `json:"transit_days"`
}

// FuelSurchargeConfig holds surcharge calculation parameters
type FuelSurchargeConfig struct {
	PetrolReference        int     `json:"petrol_reference"`
	DieselReference        int     `json:"diesel_reference"`
	SurchargePercentPerStep float64 `json:"surcharge_percent_per_step"`
	CapPercent             float64 `json:"cap_percent"`
	StepSize               int     `json:"step_size"`
}

// FeesConfig holds all fee-related configuration
type FeesConfig struct {
	Lanes         map[Lane]LaneConfig  `json:"lanes"`
	FuelSurcharge FuelSurchargeConfig  `json:"fuel_surcharge"`
}

// FuelPrices holds petrol and diesel prices for a date
type FuelPrices struct {
	Petrol int `json:"petrol"`
	Diesel int `json:"diesel"`
}

// HubCapacity holds capacity per hub for a date (grams)
type HubCapacity struct {
	Hubs map[string]int // normalized hub name → daily grams
}

// RawOrder represents the raw JSON input before validation
type RawOrder struct {
	OrderID   string          `json:"order_id"`
	CreatedAt string          `json:"created_at"`
	From      RawAddress      `json:"from"`
	To        RawAddress      `json:"to"`
	WeightG   interface{}     `json:"weight_g"`
	LengthCM  interface{}     `json:"length_cm"`
	WidthCM   interface{}     `json:"width_cm"`
	HeightCM  interface{}     `json:"height_cm"`
}

type RawAddress struct {
	Province string      `json:"province"`
	District interface{} `json:"district"`
	Ward     interface{} `json:"ward"`
}

// ParsedDate returns created_at as time.Time
func (o *RawOrder) ParsedDate() time.Time {
	t, _ := time.Parse("2006-01-02", o.CreatedAt)
	return t
}

// GetDistrict returns the district string pointer
func (a *RawAddress) GetDistrict() *string {
	if a.District == nil {
		return nil
	}
	s, ok := a.District.(string)
	if !ok {
		return nil
	}
	return &s
}

// GetWard returns the ward string pointer
func (a *RawAddress) GetWard() *string {
	if a.Ward == nil {
		return nil
	}
	s, ok := a.Ward.(string)
	if !ok {
		return nil
	}
	return &s
}

// QuoteOutput is the JSON output for one order
type QuoteOutput struct {
	OrderID      string  `json:"order_id"`
	Status       string  `json:"status"`
	Lane         *string `json:"lane"`
	ChargeableG  *int    `json:"chargeable_g"`
	Base         *int    `json:"base"`
	Surcharge    *int    `json:"surcharge"`
	Total        *int    `json:"total"`
	DistanceKM   *int    `json:"distance_km"`
	PickupDate   *string `json:"pickup_date"`
	PromisedDate *string `json:"promised_date"`
	Duplicate    bool    `json:"duplicate"`
	Reason       *string `json:"reason"`
}

// NewRejectedQuote creates a rejected quote output
func NewRejectedQuote(orderID string, reason RejectionReason) QuoteOutput {
	r := string(reason)
	return QuoteOutput{
		OrderID:  orderID,
		Status:   "rejected",
		Reason:   &r,
		Duplicate: false,
	}
}

// NewQuotedOutput creates a successful quote output
func NewQuotedOutput(orderID string, lane Lane, chargeableG, base, surcharge, total int,
	distanceKM *int, pickupDate, promisedDate time.Time) QuoteOutput {
	l := string(lane)
	pd := pickupDate.Format("2006-01-02")
	prd := promisedDate.Format("2006-01-02")
	return QuoteOutput{
		OrderID:      orderID,
		Status:       "quoted",
		Lane:         &l,
		ChargeableG:  &chargeableG,
		Base:         &base,
		Surcharge:    &surcharge,
		Total:        &total,
		DistanceKM:   distanceKM,
		PickupDate:   &pd,
		PromisedDate: &prd,
		Duplicate:    false,
	}
}
