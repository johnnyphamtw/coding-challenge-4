package data

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"mini-delivery/internal/models"
	"mini-delivery/internal/normalize"
)

// DataStore holds all loaded reference data
type DataStore struct {
	Fees           models.FeesConfig
	Provinces      map[string]bool          // normalized province names
	Regions        map[string]string        // normalized province → region name
	FuelPrices     *BoardLookup[models.FuelPrices]
	Capacity       *BoardLookup[models.HubCapacity]
	MergerDate     time.Time
	Mapping        *MappingData
	Roads          *RoadGraph
	RoadsDate      time.Time
	DistanceConfig DistanceConfig
}

type DistanceConfig struct {
	MotorbikeMaxKM int
	ExtraDayKM     int
}

// MappingData holds address mapping lookups
type MappingData struct {
	// OldToNew: key = "normProv|normDist|normWard" → NewAddr
	OldToNew map[string]NewAddr
	// NewAddrSet: key = "normProv|normWard" → OldProvince (original name for depot)
	NewAddrSet map[string]string
	// OldProvForNewProv: maps normalized new province to set of normalized old provinces that merged into it
	OldProvForNewProv map[string]map[string]bool
	// OldToNewProv: maps normalized old province directly to normalized new province
	OldToNewProv map[string]string
}

type NewAddr struct {
	Province string // canonical new province
	Ward     string // canonical new ward
}

// RoadGraph holds the adjacency list for shortest path
type RoadGraph struct {
	Adj   map[string][]Edge // normalized province → edges
	Cache map[string]int    // "a|b" (sorted) → distance, -1 = no route
}

type Edge struct {
	To   string
	Dist int
}

func LoadAll(dir string) (*DataStore, error) {
	ds := &DataStore{
		Provinces: make(map[string]bool),
		Regions:   make(map[string]string),
	}

	// Load fees
	if err := loadFees(dir, ds); err != nil {
		return nil, fmt.Errorf("fees: %w", err)
	}
	// Load provinces
	if err := loadProvinces(dir, ds); err != nil {
		return nil, fmt.Errorf("provinces: %w", err)
	}
	// Load regions
	if err := loadRegions(dir, ds); err != nil {
		return nil, fmt.Errorf("regions: %w", err)
	}
	// Load fuel prices
	if err := loadFuelPrices(dir, ds); err != nil {
		return nil, fmt.Errorf("fuel-prices: %w", err)
	}
	// Load capacity
	if err := loadCapacity(dir, ds); err != nil {
		return nil, fmt.Errorf("capacity: %w", err)
	}
	// Load mapping
	if err := loadMapping(dir, ds); err != nil {
		return nil, fmt.Errorf("mapping: %w", err)
	}
	// Load roads
	if err := loadRoads(dir, ds); err != nil {
		return nil, fmt.Errorf("roads: %w", err)
	}

	return ds, nil
}

func loadFees(dir string, ds *DataStore) error {
	raw, err := os.ReadFile(filepath.Join(dir, "fees.json"))
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &ds.Fees)
}

func loadProvinces(dir string, ds *DataStore) error {
	raw, err := os.ReadFile(filepath.Join(dir, "provinces.json"))
	if err != nil {
		return err
	}
	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return err
	}
	for _, n := range names {
		ds.Provinces[normalize.Name(n)] = true
	}
	return nil
}

func loadRegions(dir string, ds *DataStore) error {
	raw, err := os.ReadFile(filepath.Join(dir, "regions.json"))
	if err != nil {
		return err
	}
	var regionMap map[string]string
	if err := json.Unmarshal(raw, &regionMap); err != nil {
		return err
	}
	for prov, region := range regionMap {
		ds.Regions[normalize.Name(prov)] = region
	}
	return nil
}

func loadFuelPrices(dir string, ds *DataStore) error {
	raw, err := os.ReadFile(filepath.Join(dir, "fuel-prices.json"))
	if err != nil {
		return err
	}
	var entries []struct {
		EffectiveFrom string `json:"effective_from"`
		Petrol        int    `json:"petrol"`
		Diesel        int    `json:"diesel"`
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return err
	}
	var boards []Board[models.FuelPrices]
	for _, e := range entries {
		d, err := time.Parse("2006-01-02", e.EffectiveFrom)
		if err != nil {
			return err
		}
		boards = append(boards, Board[models.FuelPrices]{
			Date: d,
			Data: models.FuelPrices{Petrol: e.Petrol, Diesel: e.Diesel},
		})
	}
	ds.FuelPrices = NewBoardLookup(boards)
	return nil
}

func loadCapacity(dir string, ds *DataStore) error {
	raw, err := os.ReadFile(filepath.Join(dir, "capacity.json"))
	if err != nil {
		return err
	}
	var entries []struct {
		EffectiveFrom string         `json:"effective_from"`
		Hubs          map[string]int `json:"hubs"`
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return err
	}
	var boards []Board[models.HubCapacity]
	for _, e := range entries {
		d, err := time.Parse("2006-01-02", e.EffectiveFrom)
		if err != nil {
			return err
		}
		normalized := make(map[string]int)
		for hub, kg := range e.Hubs {
			// Convert kg to grams
			normalized[normalize.Name(hub)] = kg * 1000
		}
		boards = append(boards, Board[models.HubCapacity]{
			Date: d,
			Data: models.HubCapacity{Hubs: normalized},
		})
	}
	ds.Capacity = NewBoardLookup(boards)
	return nil
}

func loadMapping(dir string, ds *DataStore) error {
	f, err := os.Open(filepath.Join(dir, "mapping.csv"))
	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	ds.Mapping = &MappingData{
		OldToNew:          make(map[string]NewAddr),
		NewAddrSet:        make(map[string]string),
		OldProvForNewProv: make(map[string]map[string]bool),
		OldToNewProv:      make(map[string]string),
	}

	// Find column indices from header
	header := records[0]
	colIdx := make(map[string]int)
	for i, h := range header {
		colIdx[h] = i
	}

	for _, row := range records[1:] {
		oldProv := row[colIdx["old_province"]]
		oldDist := row[colIdx["old_district"]]
		oldWard := row[colIdx["old_ward"]]
		newProv := row[colIdx["new_province"]]
		newWard := row[colIdx["new_ward"]]
		effFrom := row[colIdx["effective_from"]]

		// Parse merger date from first data row
		if ds.MergerDate.IsZero() {
			d, err := time.Parse("2006-01-02", effFrom)
			if err != nil {
				return err
			}
			ds.MergerDate = d
		}

		// Build old → new lookup
		oldKey := normalize.Name(oldProv) + "|" + normalize.Name(oldDist) + "|" + normalize.Name(oldWard)
		ds.Mapping.OldToNew[oldKey] = NewAddr{Province: newProv, Ward: newWard}

		// Build new address set (province, ward) → old province
		newKey := normalize.Name(newProv) + "|" + normalize.Name(newWard)
		ds.Mapping.NewAddrSet[newKey] = oldProv

		// Track which old provinces map to which new provinces
		normNewProv := normalize.Name(newProv)
		normOldProv := normalize.Name(oldProv)
		ds.Mapping.OldToNewProv[normOldProv] = normNewProv
		if ds.Mapping.OldProvForNewProv[normNewProv] == nil {
			ds.Mapping.OldProvForNewProv[normNewProv] = make(map[string]bool)
		}
		ds.Mapping.OldProvForNewProv[normNewProv][normOldProv] = true
	}

	return nil
}

func loadRoads(dir string, ds *DataStore) error {
	raw, err := os.ReadFile(filepath.Join(dir, "roads.json"))
	if err != nil {
		return err
	}
	var data struct {
		EffectiveFrom  string `json:"effective_from"`
		MotorbikeMaxKM int    `json:"motorbike_max_km"`
		ExtraDayKM     int    `json:"extra_day_km"`
		Roads          []struct {
			From string `json:"from"`
			To   string `json:"to"`
			KM   int    `json:"km"`
		} `json:"roads"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}

	d, err := time.Parse("2006-01-02", data.EffectiveFrom)
	if err != nil {
		return err
	}
	ds.RoadsDate = d
	ds.DistanceConfig = DistanceConfig{
		MotorbikeMaxKM: data.MotorbikeMaxKM,
		ExtraDayKM:     data.ExtraDayKM,
	}

	ds.Roads = &RoadGraph{
		Adj:   make(map[string][]Edge),
		Cache: make(map[string]int),
	}
	for _, r := range data.Roads {
		from := normalize.Name(r.From)
		to := normalize.Name(r.To)
		ds.Roads.Adj[from] = append(ds.Roads.Adj[from], Edge{To: to, Dist: r.KM})
		ds.Roads.Adj[to] = append(ds.Roads.Adj[to], Edge{To: from, Dist: r.KM})
	}

	return nil
}
