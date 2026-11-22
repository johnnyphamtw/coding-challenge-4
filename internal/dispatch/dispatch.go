package dispatch

import (
	"sync"
	"time"

	"mini-delivery/internal/data"
	"mini-delivery/internal/models"
	"mini-delivery/internal/normalize"
)

type HubState struct {
	mu             sync.Mutex
	remaining      map[string]int       // key: "normHub|YYYY-MM-DD" -> remaining grams
	firstAvailable map[string]time.Time // normHub -> earliest date that may have capacity > 0
	capacityLookup *data.BoardLookup[models.HubCapacity]
}

func NewHubState(capacityLookup *data.BoardLookup[models.HubCapacity]) *HubState {
	return &HubState{
		remaining:      make(map[string]int),
		firstAvailable: make(map[string]time.Time),
		capacityLookup: capacityLookup,
	}
}

type DispatchResult struct {
	PickupDate   time.Time
	PromisedDate time.Time
}

func (hs *HubState) getCapacity(normHub string, date time.Time) int {
	board := hs.capacityLookup.Lookup(date)
	if board == nil {
		return 0
	}
	return board.Hubs[normHub]
}

func (hs *HubState) getRemaining(normHub string, date time.Time) int {
	key := normHub + "|" + date.Format("2006-01-02")
	if rem, ok := hs.remaining[key]; ok {
		return rem
	}
	cap := hs.getCapacity(normHub, date)
	hs.remaining[key] = cap
	return cap
}

func (hs *HubState) deduct(normHub string, date time.Time, amount int) {
	key := normHub + "|" + date.Format("2006-01-02")
	rem := hs.getRemaining(normHub, date)
	newRem := rem - amount
	hs.remaining[key] = newRem

	// If this day is completely full (0 remaining) and matches firstAvailable, advance it
	if newRem <= 0 {
		firstAvail, ok := hs.firstAvailable[normHub]
		if !ok || !firstAvail.After(date) {
			nextDay := date.AddDate(0, 0, 1)
			for {
				nextKey := normHub + "|" + nextDay.Format("2006-01-02")
				if r, exists := hs.remaining[nextKey]; exists && r <= 0 {
					nextDay = nextDay.AddDate(0, 0, 1)
				} else {
					break
				}
			}
			hs.firstAvailable[normHub] = nextDay
		}
	}
}

func resolveHub(vehicle models.Vehicle, originProvince, originRegion, originalProvince string, candidateDay time.Time, ds *data.DataStore) string {
	if vehicle == models.VehicleTruck {
		return originRegion
	}
	// Motorbike
	if candidateDay.Before(ds.MergerDate) {
		return originalProvince
	}
	// Post-merger candidate day: check O(1) map first
	normOld := normalize.Name(originalProvince)
	if normNew, ok := ds.Mapping.OldToNewProv[normOld]; ok {
		return normNew
	}
	return originProvince
}

func Dispatch(
	chargeableG int,
	orderDate time.Time,
	transitDays int,
	extraDays int,
	vehicle models.Vehicle,
	originProvince string,
	originRegion string,
	originalProvince string,
	ds *data.DataStore,
	hubState *HubState,
) (*DispatchResult, *models.RejectionReason) {
	hubState.mu.Lock()
	defer hubState.mu.Unlock()

	// 1. Check daily capacity of the origin hub on orderDate
	orderHub := resolveHub(vehicle, originProvince, originRegion, originalProvince, orderDate, ds)
	normOrderHub := normalize.Name(orderHub)
	dailyCap := hubState.getCapacity(normOrderHub, orderDate)
	if dailyCap == 0 || chargeableG > dailyCap {
		reason := models.ReasonTooHeavy
		return nil, &reason
	}

	// 2. Search for earliest candidate pickup day
	candidateDay := orderDate
	for {
		dayHub := resolveHub(vehicle, originProvince, originRegion, originalProvince, candidateDay, ds)
		normDayHub := normalize.Name(dayHub)

		// Fast-forward candidateDay if this hub's earlier days are completely full
		if firstAvail, ok := hubState.firstAvailable[normDayHub]; ok && candidateDay.Before(firstAvail) {
			candidateDay = firstAvail
			// Re-resolve hub in case advancing date crossed merger boundary
			continue
		}

		rem := hubState.getRemaining(normDayHub, candidateDay)
		if rem >= chargeableG {
			hubState.deduct(normDayHub, candidateDay, chargeableG)
			pickupDate := candidateDay
			promisedDate := pickupDate.AddDate(0, 0, transitDays+extraDays)
			return &DispatchResult{
				PickupDate:   pickupDate,
				PromisedDate: promisedDate,
			}, nil
		}
		candidateDay = candidateDay.AddDate(0, 0, 1)
	}
}
