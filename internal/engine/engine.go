package engine

import (
	"encoding/json"

	"mini-delivery/internal/data"
	"mini-delivery/internal/dispatch"
	"mini-delivery/internal/models"
	"mini-delivery/internal/normalize"
	"mini-delivery/internal/pricing"
)

type Engine struct {
	ds       *data.DataStore
	hubState *dispatch.HubState
	cache    map[string][]byte // order_id → JSON output bytes
}

func New(ds *data.DataStore) *Engine {
	return &Engine{
		ds:       ds,
		hubState: dispatch.NewHubState(ds.Capacity),
		cache:    make(map[string][]byte),
	}
}

func (e *Engine) ProcessOrder(line []byte) []byte {
	// 1. Parse raw order
	var raw models.RawOrder
	if err := json.Unmarshal(line, &raw); err != nil {
		// Should not happen per spec ("Every input line is valid JSON")
		return e.marshalReject(raw.OrderID, models.ReasonBadWeight)
	}

	// 2. Duplicate check
	if cached, ok := e.cache[raw.OrderID]; ok {
		return setDuplicate(cached)
	}

	// 3. Process the order
	result := e.processNewOrder(&raw)

	// 4. Cache and return
	e.cache[raw.OrderID] = result
	return result
}

func (e *Engine) processNewOrder(raw *models.RawOrder) []byte {
	orderDate := raw.ParsedDate()

	// Step 1: Input validation (AC-04)
	if reason := pricing.ValidateOrder(raw, e.ds); reason != nil {
		return e.marshalReject(raw.OrderID, *reason)
	}

	// Step 2: Temporal context
	postMerger := !orderDate.Before(e.ds.MergerDate)
	distanceActive := !orderDate.Before(e.ds.RoadsDate)

	// Step 3: Resolve addresses
	fromResolved, toResolved, reason := e.resolveAddresses(raw, postMerger)
	if reason != nil {
		return e.marshalReject(raw.OrderID, *reason)
	}

	// Step 4: Determine lane (AC-01)
	lane := pricing.DetermineLane(fromResolved, toResolved, postMerger)

	// Step 5: Vehicle + Distance
	var vehicle models.Vehicle
	var distanceKM *int
	var extraDays int

	if distanceActive {
		// AC-10: Use distance rule
		dist := e.ds.Roads.ShortestPath(
			normalize.Name(fromResolved.Depot),
			normalize.Name(toResolved.Depot),
		)
		if dist == -1 {
			return e.marshalReject(raw.OrderID, models.ReasonNoRoute)
		}
		distanceKM = &dist
		vehicle = data.DetermineVehicle(dist, e.ds.DistanceConfig.MotorbikeMaxKM)
		extraDays = data.ExtraDays(dist, e.ds.DistanceConfig.ExtraDayKM)
	} else {
		vehicle = models.VehicleForLane(lane)
	}

	// Step 6: Calculate chargeable weight
	weightG := pricing.ExtractWeight(raw.WeightG)
	lengthCM := pricing.ExtractDimension(raw.LengthCM)
	widthCM := pricing.ExtractDimension(raw.WidthCM)
	heightCM := pricing.ExtractDimension(raw.HeightCM)
	chargeableG := pricing.CalculateChargeableWeight(weightG, lengthCM, widthCM, heightCM)

	// Step 7: Calculate base fee (AC-02)
	laneConfig := e.ds.Fees.Lanes[lane]
	baseFee := pricing.CalculateBaseFee(laneConfig, chargeableG)

	// Step 8: Calculate surcharge (AC-03)
	fuelBoard := e.ds.FuelPrices.Lookup(orderDate)
	var fuelPrice, refPrice int
	if vehicle == models.VehicleMotorbike {
		fuelPrice = fuelBoard.Petrol
		refPrice = e.ds.Fees.FuelSurcharge.PetrolReference
	} else {
		fuelPrice = fuelBoard.Diesel
		refPrice = e.ds.Fees.FuelSurcharge.DieselReference
	}
	surcharge := pricing.CalculateSurcharge(baseFee, fuelPrice, refPrice,
		e.ds.Fees.FuelSurcharge.SurchargePercentPerStep,
		e.ds.Fees.FuelSurcharge.CapPercent,
		e.ds.Fees.FuelSurcharge.StepSize)

	// Step 9: Calculate total
	total := pricing.CalculateTotal(baseFee, surcharge)

	// Step 10: Dispatch (AC-05)
	// Determine hub origin info
	originRegion := fromResolved.Region
	originProvince := fromResolved.Province
	originalProvince := raw.From.Province // original input province for hub slip detection

	dispResult, dispReason := dispatch.Dispatch(
		chargeableG, orderDate, laneConfig.TransitDays, extraDays,
		vehicle, originProvince, originRegion,
		originalProvince, e.ds, e.hubState,
	)
	if dispReason != nil {
		return e.marshalReject(raw.OrderID, *dispReason)
	}

	// Step 11: Build quote
	quote := models.NewQuotedOutput(
		raw.OrderID, lane, chargeableG, baseFee, surcharge, total,
		distanceKM, dispResult.PickupDate, dispResult.PromisedDate,
	)
	return e.marshal(quote)
}

func (e *Engine) resolveAddresses(raw *models.RawOrder, postMerger bool) (
	*models.ResolvedAddress, *models.ResolvedAddress, *models.RejectionReason) {

	fromAddr, reason := e.resolveOneAddress(&raw.From, postMerger)
	if reason != nil {
		return nil, nil, reason
	}
	toAddr, reason := e.resolveOneAddress(&raw.To, postMerger)
	if reason != nil {
		return nil, nil, reason
	}
	return fromAddr, toAddr, nil
}

func (e *Engine) resolveOneAddress(addr *models.RawAddress, postMerger bool) (
	*models.ResolvedAddress, *models.RejectionReason) {

	province := addr.Province
	district := addr.GetDistrict()
	ward := addr.GetWard()

	if !postMerger {
		// Pre-merger: no conversion, use as-is
		normProv := normalize.Name(province)
		region := e.ds.Regions[normProv]
		distStr := ""
		if district != nil {
			distStr = *district
		}
		wardStr := ""
		if ward != nil {
			wardStr = *ward
		}
		return &models.ResolvedAddress{
			Province: province,
			District: distStr,
			Ward:     wardStr,
			Region:   region,
			Depot:    province, // depot = old province for pre-merger
		}, nil
	}

	// Post-merger
	hasDistrict := district != nil && *district != ""

	if hasDistrict {
		// Old-format address: convert using mapping
		wardStr := ""
		if ward != nil {
			wardStr = *ward
		}
		key := normalize.Name(province) + "|" + normalize.Name(*district) + "|" + normalize.Name(wardStr)
		newAddr, ok := e.ds.Mapping.OldToNew[key]
		if !ok {
			r := models.ReasonUnknownWard
			return nil, &r
		}
		normNewProv := normalize.Name(newAddr.Province)
		region := e.ds.Regions[normNewProv]
		return &models.ResolvedAddress{
			Province: newAddr.Province,
			District: *district,
			Ward:     newAddr.Ward,
			Region:   region,
			Depot:    province, // depot = old province (the original province before mapping)
		}, nil
	}

	// New-format address: validate in mapping
	wardStr := ""
	if ward != nil {
		wardStr = *ward
	}
	newKey := normalize.Name(province) + "|" + normalize.Name(wardStr)
	oldProv, ok := e.ds.Mapping.NewAddrSet[newKey]
	if !ok {
		r := models.ReasonUnknownNewAddr
		return nil, &r
	}
	normProv := normalize.Name(province)
	region := e.ds.Regions[normProv]
	return &models.ResolvedAddress{
		Province: province,
		District: "",
		Ward:     wardStr,
		Region:   region,
		Depot:    oldProv, // depot = old province from mapping
	}, nil
}

func (e *Engine) marshalReject(orderID string, reason models.RejectionReason) []byte {
	quote := models.NewRejectedQuote(orderID, reason)
	return e.marshal(quote)
}

func (e *Engine) marshal(quote models.QuoteOutput) []byte {
	b, _ := json.Marshal(quote)
	return b
}

// setDuplicate takes a cached JSON output and sets "duplicate":true
func setDuplicate(cached []byte) []byte {
	// Parse, modify, re-marshal
	var quote models.QuoteOutput
	json.Unmarshal(cached, &quote)
	quote.Duplicate = true
	b, _ := json.Marshal(quote)
	return b
}
