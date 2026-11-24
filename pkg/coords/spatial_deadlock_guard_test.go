package coords

import (
	"errors"
	"testing"
	"time"
)

func TestSpatialDeadlockGuard_BasicAcquireAndRelease(t *testing.T) {
	guard := NewSpatialDeadlockGuard(5, 500*time.Millisecond)

	status, err := guard.AcquirePartition("worker-1", "tile-north-east-10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != LockStatusAcquired {
		t.Errorf("expected LockStatusAcquired, got %s", status)
	}

	// Reentrant lock by same worker
	status, err = guard.AcquirePartition("worker-1", "tile-north-east-10")
	if err != nil || status != LockStatusAcquired {
		t.Errorf("reentrant lock should succeed, got status: %s, err: %v", status, err)
	}

	released, err := guard.ReleasePartition("worker-1", "tile-north-east-10")
	if err != nil || !released {
		t.Errorf("expected clean release, got released: %v, err: %v", released, err)
	}

	summary := guard.GetTelemetrySummary()
	if summary.ActivePartitionsLocked != 0 {
		t.Errorf("expected 0 active partitions after release, got %d", summary.ActivePartitionsLocked)
	}
}

func TestSpatialDeadlockGuard_ContentionQueueAndPromotion(t *testing.T) {
	guard := NewSpatialDeadlockGuard(5, 500*time.Millisecond)

	// Worker 1 acquires tile-A
	_, _ = guard.AcquirePartition("worker-1", "tile-A")

	// Worker 2 attempts to acquire tile-A -> queued
	status, err := guard.AcquirePartition("worker-2", "tile-A")
	if err != nil {
		t.Fatalf("unexpected error on enqueue: %v", err)
	}
	if status != LockStatusWaiting {
		t.Errorf("expected LockStatusWaiting, got %s", status)
	}

	summary := guard.GetTelemetrySummary()
	if summary.TotalWaiters != 1 {
		t.Errorf("expected 1 waiter, got %d", summary.TotalWaiters)
	}

	// Worker 1 releases tile-A -> Worker 2 promoted automatically
	released, _ := guard.ReleasePartition("worker-1", "tile-A")
	if !released {
		t.Errorf("expected release to succeed")
	}

	summary = guard.GetTelemetrySummary()
	if summary.TotalWaiters != 0 {
		t.Errorf("waiter should be promoted, got %d waiters", summary.TotalWaiters)
	}

	// Worker 2 now releases tile-A
	released, _ = guard.ReleasePartition("worker-2", "tile-A")
	if !released {
		t.Errorf("worker-2 should be able to release promoted partition")
	}
}

func TestSpatialDeadlockGuard_DeadlockCycleDetection(t *testing.T) {
	guard := NewSpatialDeadlockGuard(5, 500*time.Millisecond)

	// Worker 1 holds tile-1
	_, _ = guard.AcquirePartition("worker-1", "tile-1")
	// Worker 2 holds tile-2
	_, _ = guard.AcquirePartition("worker-2", "tile-2")

	// Worker 1 waits on tile-2 (held by Worker 2)
	status, err := guard.AcquirePartition("worker-1", "tile-2")
	if err != nil || status != LockStatusWaiting {
		t.Fatalf("expected worker-1 to wait on tile-2")
	}

	// Worker 2 now requests tile-1 (held by Worker 1) -> DEADLOCK CYCLE!
	status, err = guard.AcquirePartition("worker-2", "tile-1")
	if !errors.Is(err, ErrDeadlockDetected) {
		t.Errorf("expected ErrDeadlockDetected, got status: %s, err: %v", status, err)
	}
	if status != LockStatusPreempted {
		t.Errorf("expected LockStatusPreempted, got %s", status)
	}

	summary := guard.GetTelemetrySummary()
	if summary.DeadlocksDetected != 1 || summary.DeadlocksResolved != 1 {
		t.Errorf("expected 1 deadlock detected and resolved, got %+v", summary)
	}
}

func TestSpatialDeadlockGuard_ValidationAndLimits(t *testing.T) {
	guard := NewSpatialDeadlockGuard(2, 500*time.Millisecond)

	// Invalid worker ID
	_, err := guard.AcquirePartition("", "tile-valid")
	if !errors.Is(err, ErrInvalidWorkerID) {
		t.Errorf("expected ErrInvalidWorkerID, got %v", err)
	}

	// Invalid partition ID
	_, err = guard.AcquirePartition("worker-1", "invalid!partition@")
	if !errors.Is(err, ErrInvalidPartitionID) {
		t.Errorf("expected ErrInvalidPartitionID, got %v", err)
	}

	// Lock limit ceiling
	_, _ = guard.AcquirePartition("worker-limit", "tile-1")
	_, _ = guard.AcquirePartition("worker-limit", "tile-2")
	_, err = guard.AcquirePartition("worker-limit", "tile-3")
	if !errors.Is(err, ErrPartitionLimitExceeded) {
		t.Errorf("expected ErrPartitionLimitExceeded, got %v", err)
	}

	guard.Reset()
	if guard.GetTelemetrySummary().ActivePartitionsLocked != 0 {
		t.Errorf("expected clean state after reset")
	}
}
