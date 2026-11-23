package coords

import (
	"testing"
	"time"
)

func TestGeofencePolicyContract_Validate(t *testing.T) {
	validPolicy := GeofencePolicyContract{
		PolicyID: "policy-eu-central-01",
		TenantID: "tenant-fleet-alpha",
		Zones: []GeofenceZoneSpec{
			{
				ZoneID:       "zone-warehouse-01",
				Name:         "Central Warehouse Perimeter",
				Type:         ZoneInclusion,
				Center:       Coordinates{Latitude: 52.5200, Longitude: 13.4050},
				RadiusMeters: 500.0,
			},
			{
				ZoneID:       "zone-restricted-harbor",
				Name:         "Harbor Fuel Terminal",
				Type:         ZoneExclusion,
				Center:       Coordinates{Latitude: 52.5300, Longitude: 13.4150},
				RadiusMeters: 200.0,
			},
		},
	}

	if err := validPolicy.Validate(); err != nil {
		t.Fatalf("expected valid policy, got: %v", err)
	}

	badIDPolicy := validPolicy
	badIDPolicy.PolicyID = "invalid policy id with spaces!"
	if err := badIDPolicy.Validate(); err != ErrInvalidPolicyID {
		t.Errorf("expected ErrInvalidPolicyID, got: %v", err)
	}

	emptyZonesPolicy := validPolicy
	emptyZonesPolicy.Zones = nil
	if err := emptyZonesPolicy.Validate(); err != ErrEmptyZones {
		t.Errorf("expected ErrEmptyZones, got: %v", err)
	}

	badRadiusPolicy := validPolicy
	badRadiusPolicy.Zones = []GeofenceZoneSpec{
		{
			ZoneID:       "zone-tiny",
			Type:         ZoneExclusion,
			Center:       Coordinates{Latitude: 10, Longitude: 10},
			RadiusMeters: 1.0, // < 5.0m
		},
	}
	if err := badRadiusPolicy.Validate(); err != ErrInvalidZoneRadius {
		t.Errorf("expected ErrInvalidZoneRadius, got: %v", err)
	}
}

func TestGeofencePolicyContract_EvaluatePing(t *testing.T) {
	policy := GeofencePolicyContract{
		PolicyID: "fleet-ruleset",
		TenantID: "tenant-logistics",
		Zones: []GeofenceZoneSpec{
			{
				ZoneID:       "forbidden-zone",
				Type:         ZoneExclusion,
				Center:       Coordinates{Latitude: 40.7128, Longitude: -74.0060}, // NYC City Hall
				RadiusMeters: 300.0,
			},
			{
				ZoneID:       "speed-trap",
				Type:         ZoneSpeedEnforced,
				Center:       Coordinates{Latitude: 40.7200, Longitude: -74.0100},
				RadiusMeters: 1000.0,
				MaxSpeedMps:  15.0, // ~54 km/h
			},
		},
	}

	if err := policy.Validate(); err != nil {
		t.Fatalf("policy validation error: %v", err)
	}

	now := time.Now()

	// 1. Normal safe ping outside exclusion zone
	safePing := TelemetryPing{
		DeviceID:  "van-101",
		Position:  Coordinates{Latitude: 40.7500, Longitude: -73.9800},
		Timestamp: now,
		SpeedMps:  12.0,
	}
	verdict1, err := policy.EvaluatePing(safePing, nil)
	if err != nil {
		t.Fatalf("unexpected eval error: %v", err)
	}
	if !verdict1.Allowed || verdict1.Violations[0] != ViolationNone {
		t.Errorf("expected safe ping allowed, got: %+v", verdict1)
	}

	// 2. Exclusion breach: ping inside forbidden zone
	breachPing := TelemetryPing{
		DeviceID:  "van-102",
		Position:  Coordinates{Latitude: 40.71285, Longitude: -74.00605}, // ~10m from center
		Timestamp: now,
		SpeedMps:  5.0,
	}
	verdict2, err := policy.EvaluatePing(breachPing, nil)
	if err != nil {
		t.Fatalf("unexpected eval error: %v", err)
	}
	if verdict2.Allowed || verdict2.Violations[0] != ViolationExclusionBreach {
		t.Errorf("expected exclusion breach, got: %+v", verdict2)
	}

	// 3. Speed violation inside speed-enforced zone
	speedingPing := TelemetryPing{
		DeviceID:  "van-103",
		Position:  Coordinates{Latitude: 40.7200, Longitude: -74.0100},
		Timestamp: now,
		SpeedMps:  25.0, // > 15 m/s
	}
	verdict3, err := policy.EvaluatePing(speedingPing, nil)
	if err != nil {
		t.Fatalf("unexpected eval error: %v", err)
	}
	if verdict3.Allowed || verdict3.Violations[0] != ViolationSpeedExceeded {
		t.Errorf("expected speeding violation, got: %+v", verdict3)
	}

	// 4. GPS Teleportation spoofing detection
	prevPing := TelemetryPing{
		DeviceID:  "van-104",
		Position:  Coordinates{Latitude: 40.7128, Longitude: -74.0060}, // NYC
		Timestamp: now.Add(-1 * time.Second),
	}
	teleportPing := TelemetryPing{
		DeviceID:  "van-104",
		Position:  Coordinates{Latitude: 51.5074, Longitude: -0.1278}, // London in 1 second!
		Timestamp: now,
	}
	_, err = policy.EvaluatePing(teleportPing, &prevPing)
	if err != ErrTeleportationSpoof {
		t.Errorf("expected ErrTeleportationSpoof, got: %v", err)
	}
}
