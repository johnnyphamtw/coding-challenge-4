package pricing

import (
	"encoding/json"
	"math"
	"strings"

	"mini-delivery/internal/data"
	"mini-delivery/internal/models"
	"mini-delivery/internal/normalize"
)

// ValidateOrder checks input in order:
// 1. NO_PRICE_BOARD
// 2. UNKNOWN_PROVINCE
// 3. MISSING_DISTRICT (pre-merger only)
// 4. BAD_WEIGHT
// 5. BAD_DIMENSIONS
func ValidateOrder(raw *models.RawOrder, ds *data.DataStore) *models.RejectionReason {
	orderDate := raw.ParsedDate()

	// 1. NO_PRICE_BOARD
	if ds.FuelPrices.Lookup(orderDate) == nil {
		r := models.ReasonNoPriceBoard
		return &r
	}

	// 2. UNKNOWN_PROVINCE
	for _, addr := range []*models.RawAddress{&raw.From, &raw.To} {
		normProv := normalize.Name(addr.Province)
		if !ds.Provinces[normProv] {
			r := models.ReasonUnknownProvince
			return &r
		}
	}

	// 3. MISSING_DISTRICT (only before merger)
	if orderDate.Before(ds.MergerDate) {
		for _, addr := range []*models.RawAddress{&raw.From, &raw.To} {
			dist := addr.GetDistrict()
			if dist == nil || strings.TrimSpace(*dist) == "" {
				r := models.ReasonMissingDistrict
				return &r
			}
		}
	}

	// 4. BAD_WEIGHT
	if !isPositiveWholeNumber(raw.WeightG) {
		r := models.ReasonBadWeight
		return &r
	}

	// 5. BAD_DIMENSIONS
	hasAny := raw.LengthCM != nil || raw.WidthCM != nil || raw.HeightCM != nil
	allValid := isPositiveWholeNumber(raw.LengthCM) &&
		isPositiveWholeNumber(raw.WidthCM) &&
		isPositiveWholeNumber(raw.HeightCM)
	if hasAny && !allValid {
		r := models.ReasonBadDimensions
		return &r
	}

	return nil
}

func isPositiveWholeNumber(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case float64:
		return val > 0 && val == math.Floor(val)
	case int:
		return val > 0
	case int64:
		return val > 0
	case json.Number:
		f, err := val.Float64()
		return err == nil && f > 0 && f == math.Floor(f)
	default:
		return false
	}
}

// ExtractWeight extracts weight in grams from raw value
func ExtractWeight(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	default:
		return 0
	}
}

// ExtractDimension extracts dimension in cm from raw value
func ExtractDimension(v interface{}) *int {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case float64:
		i := int(val)
		return &i
	case int:
		return &val
	case int64:
		i := int(val)
		return &i
	default:
		return nil
	}
}

// DetermineLane computes the lane from two resolved addresses
func DetermineLane(from, to *models.ResolvedAddress, postMerger bool) models.Lane {
	fromProv := normalize.Name(from.Province)
	toProv := normalize.Name(to.Province)

	if fromProv != toProv {
		if from.Region == to.Region && from.Region != "" {
			return models.LaneSameRegion
		}
		return models.LaneCrossRegion
	}

	if postMerger {
		if normalize.Name(from.Ward) == normalize.Name(to.Ward) && from.Ward != "" {
			return models.LaneLocal
		}
		return models.LaneSameProvince
	}

	if normalize.Name(from.District) == normalize.Name(to.District) && from.District != "" {
		return models.LaneLocal
	}
	return models.LaneSameProvince
}

// CalculateChargeableWeight computes max(weight_g, volume_g)
// volume_g = ceil(L * W * H / 5)
func CalculateChargeableWeight(weightG int, lengthCM, widthCM, heightCM *int) int {
	if lengthCM == nil || widthCM == nil || heightCM == nil {
		return weightG
	}
	vol := *lengthCM * *widthCM * *heightCM
	volG := (vol + 4) / 5
	if volG > weightG {
		return volG
	}
	return weightG
}

// CalculateBaseFee computes the base fee for a lane and chargeable weight
func CalculateBaseFee(config models.LaneConfig, chargeableG int) int {
	if chargeableG <= config.FirstBlockG {
		return config.FirstBlock
	}
	if config.ExtraBlockG <= 0 {
		return config.FirstBlock
	}
	remaining := chargeableG - config.FirstBlockG
	extraBlocks := (remaining + config.ExtraBlockG - 1) / config.ExtraBlockG
	return config.FirstBlock + extraBlocks*config.PerExtraBlock
}

// CalculateSurcharge computes fuel surcharge
func CalculateSurcharge(baseFee, fuelPrice, refPrice int, surchargePercentPerStep, capPercent float64, stepSize int) int {
	if fuelPrice <= refPrice || stepSize <= 0 {
		return 0
	}
	steps := (fuelPrice - refPrice) / stepSize
	pct := float64(steps) * surchargePercentPerStep
	if pct > capPercent {
		pct = capPercent
	}
	return int(math.Floor(float64(baseFee) * pct / 100.0))
}

// CalculateTotal rounds (base + surcharge) to nearest 1000, half up
func CalculateTotal(baseFee, surcharge int) int {
	raw := baseFee + surcharge
	remainder := raw % 1000
	if remainder >= 500 {
		return raw - remainder + 1000
	}
	return raw - remainder
}
