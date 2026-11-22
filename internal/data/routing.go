package data

import (
	"container/heap"

	"mini-delivery/internal/models"
)

// ShortestPath returns the shortest distance between two depots (normalized names)
// Returns -1 if no route exists. Same depot = 0 km.
// Results are cached by sorted depot pair.
func (rg *RoadGraph) ShortestPath(depotA, depotB string) int {
	if depotA == depotB {
		return 0
	}
	key := cacheKey(depotA, depotB)
	if rg.Cache == nil {
		rg.Cache = make(map[string]int)
	}
	if dist, ok := rg.Cache[key]; ok {
		return dist
	}
	dist := rg.dijkstra(depotA, depotB)
	rg.Cache[key] = dist
	return dist
}

func cacheKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "|" + b
}

// DetermineVehicle returns vehicle based on distance and config
func DetermineVehicle(distanceKM, motorbikeMaxKM int) models.Vehicle {
	if distanceKM <= motorbikeMaxKM {
		return models.VehicleMotorbike
	}
	return models.VehicleTruck
}

// ExtraDays returns additional transit days for distance
func ExtraDays(distanceKM, extraDayKM int) int {
	if extraDayKM <= 0 {
		return 0
	}
	return distanceKM / extraDayKM
}

// --- Dijkstra implementation ---

type pqItem struct {
	node string
	dist int
	idx  int
}

type priorityQueue []*pqItem

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].dist < pq[j].dist }
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].idx = i
	pq[j].idx = j
}
func (pq *priorityQueue) Push(x interface{}) {
	item := x.(*pqItem)
	item.idx = len(*pq)
	*pq = append(*pq, item)
}
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*pq = old[:n-1]
	return item
}

func (rg *RoadGraph) dijkstra(start, end string) int {
	// If start has no edges, no route possible
	if _, ok := rg.Adj[start]; !ok {
		return -1
	}

	dist := make(map[string]int)
	dist[start] = 0

	pq := &priorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &pqItem{node: start, dist: 0})

	for pq.Len() > 0 {
		curr := heap.Pop(pq).(*pqItem)

		if curr.node == end {
			return curr.dist
		}

		if d, ok := dist[curr.node]; ok && curr.dist > d {
			continue // stale
		}

		for _, edge := range rg.Adj[curr.node] {
			newDist := curr.dist + edge.Dist
			if d, ok := dist[edge.To]; !ok || newDist < d {
				dist[edge.To] = newDist
				heap.Push(pq, &pqItem{node: edge.To, dist: newDist})
			}
		}
	}

	return -1 // no route found
}
