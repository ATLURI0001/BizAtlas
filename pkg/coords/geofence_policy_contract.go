package coords

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

var idRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// ZoneType defines the enforcement nature of a geofence zone.
type ZoneType string

const (
	ZoneExclusion      ZoneType = "EXCLUSION"       // Entry strictly forbidden
	ZoneInclusion      ZoneType = "INCLUSION"       // Operations restricted to inside zone
	ZoneSpeedEnforced  ZoneType = "SPEED_ENFORCED"  // Speed limit strictly monitored
)

var (
	ErrInvalidPolicyID    = errors.New("invalid policyId: must match ^[a-zA-Z0-9_-]{1,64}$")
	ErrInvalidTenantID    = errors.New("invalid tenantId: must match ^[a-zA-Z0-9_-]{1,64}$")
	ErrEmptyZones         = errors.New("geofence policy must contain at least one valid zone")
	ErrInvalidZoneRadius  = errors.New("circular zone radius must be between 5.0 and 100000.0 meters")
	ErrTeleportationSpoof = errors.New("telemetry ping rejected: physical teleportation speed exceeds maximum threshold")
)

// GeofenceZoneSpec defines a spatial boundary and operational constraint.
type GeofenceZoneSpec struct {
	ZoneID            string      `json:"zone_id"`
	Name              string      `json:"name"`
	Type              ZoneType    `json:"type"`
	Center            Coordinates `json:"center"`
	RadiusMeters      float64     `json:"radius_meters"`
	PolygonVertices   []Coordinates `json:"polygon_vertices,omitempty"`
	MaxSpeedMps       float64     `json:"max_speed_mps"`
}

// GeofencePolicyContract represents a validated ingress security policy.
type GeofencePolicyContract struct {
	PolicyID string             `json:"policy_id"`
	TenantID string             `json:"tenant_id"`
	Zones    []GeofenceZoneSpec `json:"zones"`
}

// Validate verifies the invariants of the GeofencePolicyContract.
func (c *GeofencePolicyContract) Validate() error {
	if !idRegex.MatchString(strings.TrimSpace(c.PolicyID)) {
		return ErrInvalidPolicyID
	}
	if !idRegex.MatchString(strings.TrimSpace(c.TenantID)) {
		return ErrInvalidTenantID
	}
	if len(c.Zones) == 0 {
		return ErrEmptyZones
	}

	seenZones := make(map[string]struct{}, len(c.Zones))
	for i, z := range c.Zones {
		zID := strings.TrimSpace(z.ZoneID)
		if !idRegex.MatchString(zID) {
			return fmt.Errorf("zone %d: invalid zone_id %q", i, z.ZoneID)
		}
		if _, ok := seenZones[zID]; ok {
			return fmt.Errorf("zone %d: duplicate zone_id %q", i, zID)
		}
		seenZones[zID] = struct{}{}

		switch z.Type {
		case ZoneExclusion, ZoneInclusion, ZoneSpeedEnforced:
			// valid
		default:
			return fmt.Errorf("zone %d: unsupported zone type %q", i, z.Type)
		}

		if len(z.PolygonVertices) >= 3 {
			for vIdx, v := range z.PolygonVertices {
				if !v.IsValid() {
					return fmt.Errorf("zone %d vertex %d: invalid coordinates %s", i, vIdx, v)
				}
			}
		} else {
			if !z.Center.IsValid() {
				return fmt.Errorf("zone %d: center coordinates are invalid %s", i, z.Center)
			}
			if z.RadiusMeters < 5.0 || z.RadiusMeters > 100000.0 {
				return ErrInvalidZoneRadius
			}
		}
	}

	return nil
}

// TelemetryPing represents a single spatial ingress event from a fleet device.
type TelemetryPing struct {
	DeviceID  string      `json:"device_id"`
	Position  Coordinates `json:"position"`
	Timestamp time.Time   `json:"timestamp"`
	SpeedMps  float64     `json:"speed_mps"`
}

// ViolationType identifies the specific policy transgression.
type ViolationType string

const (
	ViolationNone             ViolationType = "NONE"
	ViolationExclusionBreach  ViolationType = "EXCLUSION_BREACH"
	ViolationInclusionEscape  ViolationType = "INCLUSION_ESCAPE"
	ViolationSpeedExceeded    ViolationType = "SPEED_EXCEEDED"
)

// EvaluationVerdict details policy compliance for an evaluated ping.
type EvaluationVerdict struct {
	Allowed        bool            `json:"allowed"`
	Violations     []ViolationType `json:"violations"`
	ViolatingZones []string        `json:"violating_zones"`
	SanitizedPing  TelemetryPing   `json:"sanitized_ping"`
}

// EvaluatePing validates and evaluates an incoming telemetry ping against the policy.
// Optional prevPing detects GPS teleportation/spoofing (>150 m/s or ~540 km/h).
func (c *GeofencePolicyContract) EvaluatePing(ping TelemetryPing, prevPing *TelemetryPing) (*EvaluationVerdict, error) {
	if !ping.Position.IsValid() {
		return nil, ErrInvalidCoordinates
	}
	if !idRegex.MatchString(strings.TrimSpace(ping.DeviceID)) {
		return nil, fmt.Errorf("invalid device_id %q", ping.DeviceID)
	}

	// GPS spoofing / teleportation check
	if prevPing != nil && prevPing.Position.IsValid() && !prevPing.Timestamp.IsZero() {
		deltaT := ping.Timestamp.Sub(prevPing.Timestamp).Seconds()
		if deltaT > 0 {
			distKm, err := HaversineDistance(prevPing.Position, ping.Position)
			if err == nil {
				distMeters := distKm * 1000.0
				apparentSpeed := distMeters / deltaT
				if apparentSpeed > 150.0 { // Faster than commercial aircraft taxi/transit
					return nil, ErrTeleportationSpoof
				}
			}
		}
	}

	verdict := &EvaluationVerdict{
		Allowed:       true,
		SanitizedPing: ping,
	}

	for _, z := range c.Zones {
		inside := isInsideZone(ping.Position, z)

		switch z.Type {
		case ZoneExclusion:
			if inside {
				verdict.Allowed = false
				verdict.Violations = append(verdict.Violations, ViolationExclusionBreach)
				verdict.ViolatingZones = append(verdict.ViolatingZones, z.ZoneID)
			}
		case ZoneInclusion:
			if !inside {
				verdict.Allowed = false
				verdict.Violations = append(verdict.Violations, ViolationInclusionEscape)
				verdict.ViolatingZones = append(verdict.ViolatingZones, z.ZoneID)
			}
		case ZoneSpeedEnforced:
			if inside && z.MaxSpeedMps > 0 && ping.SpeedMps > z.MaxSpeedMps {
				verdict.Allowed = false
				verdict.Violations = append(verdict.Violations, ViolationSpeedExceeded)
				verdict.ViolatingZones = append(verdict.ViolatingZones, z.ZoneID)
			}
		}
	}

	if len(verdict.Violations) == 0 {
		verdict.Violations = []ViolationType{ViolationNone}
	}

	return verdict, nil
}

func isInsideZone(p Coordinates, z GeofenceZoneSpec) bool {
	if len(z.PolygonVertices) >= 3 {
		poly, err := NewPolygon(z.PolygonVertices)
		if err != nil {
			return false
		}
		inside, err := poly.ContainsPoint(p)
		if err != nil {
			return false
		}
		return inside
	}

	// Circular zone
	distKm, err := HaversineDistance(p, z.Center)
	if err != nil {
		return false
	}
	distMeters := distKm * 1000.0
	return distMeters <= z.RadiusMeters
}

// DistanceInMeters helper calculates haversine distance in meters.
func DistanceInMeters(p1, p2 Coordinates) (float64, error) {
	distKm, err := HaversineDistance(p1, p2)
	if err != nil {
		return 0.0, err
	}
	return distKm * 1000.0, nil
}

// Ensure math is used
var _ = math.Abs
