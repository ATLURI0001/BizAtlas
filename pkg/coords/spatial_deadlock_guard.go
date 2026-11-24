package coords

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidWorkerID   = errors.New("workerID must be a non-empty string between 2 and 64 characters")
	ErrInvalidPartitionID = errors.New("partitionID must match '^[a-zA-Z0-9_\\-]{2,64}$'")
	ErrDeadlockDetected  = errors.New("circular dependency detected in spatial partition wait-for graph")
	ErrAcquisitionTimeout = errors.New("lock acquisition timeout exceeded deadline")
	ErrPartitionLocked   = errors.New("partition is currently locked by another worker")
	ErrPartitionLimitExceeded = errors.New("worker has exceeded max simultaneous partition locks")
)

var (
	partitionRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]{2,64}$`)
)

type PartitionLockStatus string

const (
	LockStatusAcquired PartitionLockStatus = "ACQUIRED"
	LockStatusWaiting  PartitionLockStatus = "WAITING"
	LockStatusReleased PartitionLockStatus = "RELEASED"
	LockStatusPreempted PartitionLockStatus = "PREEMPTED"
)

// PartitionLockEntry tracks ownership and waiting queues for a spatial tile partition.
type PartitionLockEntry struct {
	PartitionID string
	OwnerWorker string
	AcquiredAt  time.Time
	WaitQueue   []string // List of workerIDs waiting on this partition
}

// DeadlockTelemetrySummary provides operational health statistics on lock contention.
type DeadlockTelemetrySummary struct {
	ActivePartitionsLocked int
	TotalWaiters           int
	DeadlocksDetected      int
	DeadlocksResolved      int
	TimeoutsTripped        int
}

// SpatialDeadlockGuard coordinates thread-safe partition locking and detects circular waits.
type SpatialDeadlockGuard struct {
	mu                   sync.RWMutex
	maxLocksPerWorker    int
	defaultTimeout       time.Duration
	partitionLocks       map[string]*PartitionLockEntry // partitionID -> Entry
	workerHeldPartitions map[string]map[string]struct{} // workerID -> set of partitionIDs
	workerWaitingOn      map[string]string              // workerID -> partitionID waiting for

	// Telemetry counters
	deadlocksDetected int
	deadlocksResolved int
	timeoutsTripped   int
}

// NewSpatialDeadlockGuard initializes a new deadlock guard for spatial tile partitions.
func NewSpatialDeadlockGuard(maxLocksPerWorker int, defaultTimeout time.Duration) *SpatialDeadlockGuard {
	if maxLocksPerWorker <= 0 {
		maxLocksPerWorker = 10
	}
	if defaultTimeout <= 0 {
		defaultTimeout = 500 * time.Millisecond
	}

	return &SpatialDeadlockGuard{
		maxLocksPerWorker:    maxLocksPerWorker,
		defaultTimeout:       defaultTimeout,
		partitionLocks:       make(map[string]*PartitionLockEntry),
		workerHeldPartitions: make(map[string]map[string]struct{}),
		workerWaitingOn:      make(map[string]string),
	}
}

// AcquirePartition attempts to acquire a lock on a spatial partition, detecting circular waits.
func (g *SpatialDeadlockGuard) AcquirePartition(workerID, partitionID string) (PartitionLockStatus, error) {
	trimmedWorker := strings.TrimSpace(workerID)
	if len(trimmedWorker) < 2 || len(trimmedWorker) > 64 {
		return "", ErrInvalidWorkerID
	}
	trimmedPartition := strings.TrimSpace(partitionID)
	if !partitionRegex.MatchString(trimmedPartition) {
		return "", fmt.Errorf("%w: '%s'", ErrInvalidPartitionID, trimmedPartition)
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// Check per-worker held lock ceiling
	held := g.workerHeldPartitions[trimmedWorker]
	if held != nil && len(held) >= g.maxLocksPerWorker {
		return "", ErrPartitionLimitExceeded
	}

	entry, exists := g.partitionLocks[trimmedPartition]
	if !exists || entry.OwnerWorker == "" {
		// Available immediately
		if entry == nil {
			entry = &PartitionLockEntry{
				PartitionID: trimmedPartition,
				OwnerWorker: trimmedWorker,
				AcquiredAt:  time.Now(),
				WaitQueue:   make([]string, 0),
			}
			g.partitionLocks[trimmedPartition] = entry
		} else {
			entry.OwnerWorker = trimmedWorker
			entry.AcquiredAt = time.Now()
		}

		if g.workerHeldPartitions[trimmedWorker] == nil {
			g.workerHeldPartitions[trimmedWorker] = make(map[string]struct{})
		}
		g.workerHeldPartitions[trimmedWorker][trimmedPartition] = struct{}{}
		delete(g.workerWaitingOn, trimmedWorker)

		return LockStatusAcquired, nil
	}

	if entry.OwnerWorker == trimmedWorker {
		// Reentrant lock by same worker
		return LockStatusAcquired, nil
	}

	// Partition is held by another worker: construct dependency edge
	g.workerWaitingOn[trimmedWorker] = trimmedPartition

	// Deadlock check: does adding workerWaitingOn create a cycle?
	if cycle := g.detectCycle(trimmedWorker); len(cycle) > 0 {
		g.deadlocksDetected++
		delete(g.workerWaitingOn, trimmedWorker)
		g.deadlocksResolved++
		return LockStatusPreempted, fmt.Errorf("%w: cycle path: %s", ErrDeadlockDetected, strings.Join(cycle, " -> "))
	}

	// Enqueue waiter
	entry.WaitQueue = append(entry.WaitQueue, trimmedWorker)
	return LockStatusWaiting, nil
}

// detectCycle runs depth-first search on the wait-for graph starting from currentWorker.
func (g *SpatialDeadlockGuard) detectCycle(startWorker string) []string {
	visited := make(map[string]bool)
	path := make([]string, 0)

	var dfs func(currWorker string) bool
	dfs = func(currWorker string) bool {
		visited[currWorker] = true
		path = append(path, currWorker)

		waitingPartition, isWaiting := g.workerWaitingOn[currWorker]
		if isWaiting {
			entry := g.partitionLocks[waitingPartition]
			if entry != nil && entry.OwnerWorker != "" {
				nextWorker := entry.OwnerWorker
				if nextWorker == startWorker {
					path = append(path, nextWorker)
					return true
				}
				if !visited[nextWorker] {
					if dfs(nextWorker) {
						return true
					}
				}
			}
		}

		path = path[:len(path)-1]
		return false
	}

	if dfs(startWorker) {
		return path
	}
	return nil
}

// ReleasePartition releases a lock on a spatial partition and promotes the next waiter.
func (g *SpatialDeadlockGuard) ReleasePartition(workerID, partitionID string) (bool, error) {
	trimmedWorker := strings.TrimSpace(workerID)
	trimmedPartition := strings.TrimSpace(partitionID)

	g.mu.Lock()
	defer g.mu.Unlock()

	entry, exists := g.partitionLocks[trimmedPartition]
	if !exists || entry.OwnerWorker != trimmedWorker {
		return false, nil
	}

	// Remove from held set
	if held, ok := g.workerHeldPartitions[trimmedWorker]; ok {
		delete(held, trimmedPartition)
		if len(held) == 0 {
			delete(g.workerHeldPartitions, trimmedWorker)
		}
	}

	if len(entry.WaitQueue) > 0 {
		// Promote next waiter
		nextWorker := entry.WaitQueue[0]
		entry.WaitQueue = entry.WaitQueue[1:]
		entry.OwnerWorker = nextWorker
		entry.AcquiredAt = time.Now()

		if g.workerHeldPartitions[nextWorker] == nil {
			g.workerHeldPartitions[nextWorker] = make(map[string]struct{})
		}
		g.workerHeldPartitions[nextWorker][trimmedPartition] = struct{}{}
		delete(g.workerWaitingOn, nextWorker)

		return true, nil
	}

	// No waiters, clean up entry
	delete(g.partitionLocks, trimmedPartition)
	return true, nil
}

// GetTelemetrySummary returns current operational diagnostics and contention counters.
func (g *SpatialDeadlockGuard) GetTelemetrySummary() DeadlockTelemetrySummary {
	g.mu.RLock()
	defer g.mu.RUnlock()

	totalWaiters := 0
	for _, entry := range g.partitionLocks {
		totalWaiters += len(entry.WaitQueue)
	}

	return DeadlockTelemetrySummary{
		ActivePartitionsLocked: len(g.partitionLocks),
		TotalWaiters:           totalWaiters,
		DeadlocksDetected:      g.deadlocksDetected,
		DeadlocksResolved:      g.deadlocksResolved,
		TimeoutsTripped:        g.timeoutsTripped,
	}
}

// Reset clears all locks and counters.
func (g *SpatialDeadlockGuard) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.partitionLocks = make(map[string]*PartitionLockEntry)
	g.workerHeldPartitions = make(map[string]map[string]struct{})
	g.workerWaitingOn = make(map[string]string)
	g.deadlocksDetected = 0
	g.deadlocksResolved = 0
	g.timeoutsTripped = 0
}
