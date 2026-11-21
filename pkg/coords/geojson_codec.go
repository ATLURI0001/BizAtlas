package coords

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

var (
	// ErrInvalidGeoJSON indicates that the JSON payload violates RFC 7946 or contains malformed geometry.
	ErrInvalidGeoJSON = errors.New("invalid GeoJSON payload or unsupported geometry type")
	// ErrInvalidPolyline indicates an invalid or truncated polyline string.
	ErrInvalidPolyline = errors.New("invalid encoded polyline format")
	// ErrEmptyCoordinates indicates an empty slice of coordinates where at least one was required.
	ErrEmptyCoordinates = errors.New("coordinates slice cannot be empty")
)

// GeoJSONType constants
const (
	TypeFeatureCollection = "FeatureCollection"
	TypeFeature           = "Feature"
	TypePoint             = "Point"
	TypeLineString        = "LineString"
	TypePolygon           = "Polygon"
)

// GeoJSONGeometry represents an RFC 7946 geometry object.
type GeoJSONGeometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

// GeoJSONFeature represents an RFC 7946 Feature object.
type GeoJSONFeature struct {
	Type       string                 `json:"type"`
	ID         interface{}            `json:"id,omitempty"`
	Geometry   *GeoJSONGeometry       `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
	BBox       []float64              `json:"bbox,omitempty"`
}

// GeoJSONFeatureCollection represents an RFC 7946 FeatureCollection object.
type GeoJSONFeatureCollection struct {
	Type     string           `json:"type"`
	Features []GeoJSONFeature `json:"features"`
	BBox     []float64        `json:"bbox,omitempty"`
}

// PointToFeature converts a Coordinate into a GeoJSON Point Feature.
func PointToFeature(c Coordinates, props map[string]interface{}) (*GeoJSONFeature, error) {
	if !c.IsValid() {
		return nil, ErrInvalidCoordinates
	}
	rawCoords, err := json.Marshal([]float64{c.Longitude, c.Latitude})
	if err != nil {
		return nil, err
	}
	if props == nil {
		props = make(map[string]interface{})
	}
	return &GeoJSONFeature{
		Type: TypeFeature,
		Geometry: &GeoJSONGeometry{
			Type:        TypePoint,
			Coordinates: rawCoords,
		},
		Properties: props,
		BBox:       []float64{c.Longitude, c.Latitude, c.Longitude, c.Latitude},
	}, nil
}

// LineStringToFeature converts a sequence of Coordinates into a GeoJSON LineString Feature.
func LineStringToFeature(pts []Coordinates, props map[string]interface{}) (*GeoJSONFeature, error) {
	if len(pts) < 2 {
		return nil, errors.New("linestring requires at least 2 coordinate points")
	}
	coords := make([][2]float64, len(pts))
	var minLon, minLat, maxLon, maxLat float64
	for i, p := range pts {
		if !p.IsValid() {
			return nil, ErrInvalidCoordinates
		}
		coords[i] = [2]float64{p.Longitude, p.Latitude}
		if i == 0 {
			minLon, maxLon = p.Longitude, p.Longitude
			minLat, maxLat = p.Latitude, p.Latitude
		} else {
			if p.Longitude < minLon {
				minLon = p.Longitude
			}
			if p.Longitude > maxLon {
				maxLon = p.Longitude
			}
			if p.Latitude < minLat {
				minLat = p.Latitude
			}
			if p.Latitude > maxLat {
				maxLat = p.Latitude
			}
		}
	}
	rawCoords, err := json.Marshal(coords)
	if err != nil {
		return nil, err
	}
	if props == nil {
		props = make(map[string]interface{})
	}
	return &GeoJSONFeature{
		Type: TypeFeature,
		Geometry: &GeoJSONGeometry{
			Type:        TypeLineString,
			Coordinates: rawCoords,
		},
		Properties: props,
		BBox:       []float64{minLon, minLat, maxLon, maxLat},
	}, nil
}

// PolygonToFeature converts a Polygon into a GeoJSON Polygon Feature.
// RFC 7946 requires polygon linear rings to be closed (first and last coordinate identical).
func PolygonToFeature(poly *Polygon, props map[string]interface{}) (*GeoJSONFeature, error) {
	if poly == nil || len(poly.Vertices) < 3 {
		return nil, ErrDegeneratePolygon
	}

	ring := make([][2]float64, 0, len(poly.Vertices)+1)
	var minLon, minLat, maxLon, maxLat float64

	for i, v := range poly.Vertices {
		if !v.IsValid() {
			return nil, ErrInvalidCoordinates
		}
		ring = append(ring, [2]float64{v.Longitude, v.Latitude})
		if i == 0 {
			minLon, maxLon = v.Longitude, v.Longitude
			minLat, maxLat = v.Latitude, v.Latitude
		} else {
			if v.Longitude < minLon {
				minLon = v.Longitude
			}
			if v.Longitude > maxLon {
				maxLon = v.Longitude
			}
			if v.Latitude < minLat {
				minLat = v.Latitude
			}
			if v.Latitude > maxLat {
				maxLat = v.Latitude
			}
		}
	}

	// Ensure ring is closed
	first := ring[0]
	last := ring[len(ring)-1]
	if first[0] != last[0] || first[1] != last[1] {
		ring = append(ring, first)
	}

	// GeoJSON Polygon coordinates are array of linear rings (outer ring first)
	multiRings := [][][2]float64{ring}
	rawCoords, err := json.Marshal(multiRings)
	if err != nil {
		return nil, err
	}
	if props == nil {
		props = make(map[string]interface{})
	}

	return &GeoJSONFeature{
		Type: TypeFeature,
		Geometry: &GeoJSONGeometry{
			Type:        TypePolygon,
			Coordinates: rawCoords,
		},
		Properties: props,
		BBox:       []float64{minLon, minLat, maxLon, maxLat},
	}, nil
}

// EncodeFeatureCollection encodes a slice of GeoJSON features into a valid FeatureCollection JSON.
func EncodeFeatureCollection(features []GeoJSONFeature) (*GeoJSONFeatureCollection, error) {
	fc := &GeoJSONFeatureCollection{
		Type:     TypeFeatureCollection,
		Features: features,
	}
	if len(features) > 0 {
		var minLon, minLat, maxLon, maxLat float64
		hasBBox := false
		for _, f := range features {
			if len(f.BBox) == 4 {
				if !hasBBox {
					minLon, minLat, maxLon, maxLat = f.BBox[0], f.BBox[1], f.BBox[2], f.BBox[3]
					hasBBox = true
				} else {
					if f.BBox[0] < minLon {
						minLon = f.BBox[0]
					}
					if f.BBox[1] < minLat {
						minLat = f.BBox[1]
					}
					if f.BBox[2] > maxLon {
						maxLon = f.BBox[2]
					}
					if f.BBox[3] > maxLat {
						maxLat = f.BBox[3]
					}
				}
			}
		}
		if hasBBox {
			fc.BBox = []float64{minLon, minLat, maxLon, maxLat}
		}
	}
	return fc, nil
}

// DecodeFeaturePoints parses a GeoJSON Feature or FeatureCollection and extracts all Point coordinates.
func DecodeFeaturePoints(data []byte) ([]Coordinates, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidGeoJSON, err)
	}

	typeVal, _ := raw["type"].(string)
	var points []Coordinates

	switch typeVal {
	case TypeFeatureCollection:
		features, ok := raw["features"].([]interface{})
		if !ok {
			return nil, ErrInvalidGeoJSON
		}
		for _, featItem := range features {
			featMap, ok := featItem.(map[string]interface{})
			if !ok {
				continue
			}
			geomMap, ok := featMap["geometry"].(map[string]interface{})
			if !ok || geomMap == nil {
				continue
			}
			pts, err := extractCoordsFromGeometry(geomMap)
			if err == nil {
				points = append(points, pts...)
			}
		}
	case TypeFeature:
		geomMap, ok := raw["geometry"].(map[string]interface{})
		if !ok || geomMap == nil {
			return nil, ErrInvalidGeoJSON
		}
		pts, err := extractCoordsFromGeometry(geomMap)
		if err != nil {
			return nil, err
		}
		points = append(points, pts...)
	case TypePoint, TypeLineString, TypePolygon:
		pts, err := extractCoordsFromGeometry(raw)
		if err != nil {
			return nil, err
		}
		points = append(points, pts...)
	default:
		return nil, fmt.Errorf("%w: unexpected type %s", ErrInvalidGeoJSON, typeVal)
	}

	return points, nil
}

func extractCoordsFromGeometry(geom map[string]interface{}) ([]Coordinates, error) {
	gType, _ := geom["type"].(string)
	rawCoords := geom["coordinates"]
	if rawCoords == nil {
		return nil, ErrInvalidGeoJSON
	}

	var results []Coordinates
	switch gType {
	case TypePoint:
		arr, ok := rawCoords.([]interface{})
		if !ok || len(arr) < 2 {
			return nil, ErrInvalidGeoJSON
		}
		lon, ok1 := arr[0].(float64)
		lat, ok2 := arr[1].(float64)
		if !ok1 || !ok2 {
			return nil, ErrInvalidGeoJSON
		}
		pt := Coordinates{Latitude: lat, Longitude: lon}
		if !pt.IsValid() {
			return nil, ErrInvalidCoordinates
		}
		results = append(results, pt)
	case TypeLineString:
		arr, ok := rawCoords.([]interface{})
		if !ok {
			return nil, ErrInvalidGeoJSON
		}
		for _, item := range arr {
			ptArr, ok := item.([]interface{})
			if !ok || len(ptArr) < 2 {
				continue
			}
			lon, ok1 := ptArr[0].(float64)
			lat, ok2 := ptArr[1].(float64)
			if ok1 && ok2 {
				pt := Coordinates{Latitude: lat, Longitude: lon}
				if pt.IsValid() {
					results = append(results, pt)
				}
			}
		}
	case TypePolygon:
		rings, ok := rawCoords.([]interface{})
		if !ok {
			return nil, ErrInvalidGeoJSON
		}
		for _, ringItem := range rings {
			ringArr, ok := ringItem.([]interface{})
			if !ok {
				continue
			}
			for _, ptItem := range ringArr {
				ptArr, ok := ptItem.([]interface{})
				if !ok || len(ptArr) < 2 {
					continue
				}
				lon, ok1 := ptArr[0].(float64)
				lat, ok2 := ptArr[1].(float64)
				if ok1 && ok2 {
					pt := Coordinates{Latitude: lat, Longitude: lon}
					if pt.IsValid() {
						results = append(results, pt)
					}
				}
			}
		}
	default:
		return nil, fmt.Errorf("%w: unsupported geometry %s", ErrInvalidGeoJSON, gType)
	}

	return results, nil
}

// EncodePolyline encodes a sequence of Coordinates into a standard Google Encoded Polyline string (5-decimal precision).
func EncodePolyline(points []Coordinates) (string, error) {
	if len(points) == 0 {
		return "", ErrEmptyCoordinates
	}

	var sb strings.Builder
	var prevLat, prevLon int64

	for _, pt := range points {
		if !pt.IsValid() {
			return "", ErrInvalidCoordinates
		}

		latE5 := int64(math.Round(pt.Latitude * 1e5))
		lonE5 := int64(math.Round(pt.Longitude * 1e5))

		dLat := latE5 - prevLat
		dLon := lonE5 - prevLon

		prevLat = latE5
		prevLon = lonE5

		encodePolylineValue(&sb, dLat)
		encodePolylineValue(&sb, dLon)
	}

	return sb.String(), nil
}

func encodePolylineValue(sb *strings.Builder, val int64) {
	var sVal uint64
	if val < 0 {
		sVal = uint64(^(val << 1))
	} else {
		sVal = uint64(val << 1)
	}

	for sVal >= 0x20 {
		sb.WriteByte(byte((sVal & 0x1f) | 0x20) + 63)
		sVal >>= 5
	}
	sb.WriteByte(byte(sVal) + 63)
}

// DecodePolyline decodes a Google Encoded Polyline string into a sequence of Coordinates.
func DecodePolyline(encoded string) ([]Coordinates, error) {
	if len(encoded) == 0 {
		return nil, ErrInvalidPolyline
	}

	var points []Coordinates
	var lat, lon int64
	idx := 0
	length := len(encoded)

	for idx < length {
		dLat, nextIdx, err := decodePolylineChunk(encoded, idx)
		if err != nil {
			return nil, err
		}
		idx = nextIdx

		if idx >= length {
			return nil, ErrInvalidPolyline
		}

		dLon, nextIdx, err := decodePolylineChunk(encoded, idx)
		if err != nil {
			return nil, err
		}
		idx = nextIdx

		lat += dLat
		lon += dLon

		pt := Coordinates{
			Latitude:  float64(lat) / 1e5,
			Longitude: float64(lon) / 1e5,
		}
		if !pt.IsValid() {
			return nil, ErrInvalidCoordinates
		}
		points = append(points, pt)
	}

	return points, nil
}

func decodePolylineChunk(encoded string, start int) (int64, int, error) {
	var result int64
	var shift uint
	idx := start
	length := len(encoded)

	for {
		if idx >= length {
			return 0, idx, ErrInvalidPolyline
		}
		b := int64(encoded[idx]) - 63
		idx++
		if b < 0 || b > 63 {
			return 0, idx, ErrInvalidPolyline
		}
		result |= (b & 0x1f) << shift
		shift += 5
		if b < 0x20 {
			break
		}
	}

	// Zigzag decoding
	if (result & 1) != 0 {
		result = ^(result >> 1)
	} else {
		result = result >> 1
	}

	return result, idx, nil
}

// NormalizeCoordinates wraps longitude into [-180, 180] and clamps latitude into [-90, 90].
func NormalizeCoordinates(lat, lon float64) Coordinates {
	if lat > 90.0 {
		lat = 90.0
	} else if lat < -90.0 {
		lat = -90.0
	}

	// Wrap longitude
	for lon > 180.0 {
		lon -= 360.0
	}
	for lon < -180.0 {
		lon += 360.0
	}

	return Coordinates{Latitude: lat, Longitude: lon}
}
