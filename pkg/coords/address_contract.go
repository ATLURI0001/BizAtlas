package coords

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	ErrQueryTooShort     = errors.New("geocoding query must contain at least 3 characters")
	ErrQueryTooLong      = errors.New("geocoding query cannot exceed 256 characters")
	ErrInvalidCharacters = errors.New("query contains invalid control characters or line breaks")
	ErrInvalidCountry    = errors.New("country code must be a valid 2-letter ISO 3166-1 alpha-2 code")
	ErrInvalidPostalCode = errors.New("postal code contains invalid characters or length out of range [3, 10]")
	ErrInvalidLimit      = errors.New("limit must be between 1 and 100")
	ErrInvalidBBox       = errors.New("bounding box coordinates are invalid or out of Earth bounds")
)

var (
	countryCodeRegex = regexp.MustCompile(`^[A-Za-z]{2}$`)
	postalCodeRegex  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 -]{1,8}[A-Za-z0-9]$`)
	multiSpaceRegex  = regexp.MustCompile(`\s+`)
)

// BoundingBox represents a rectangular geographic search boundary.
type BoundingBox struct {
	MinLat float64 `json:"min_lat"`
	MinLon float64 `json:"min_lon"`
	MaxLat float64 `json:"max_lat"`
	MaxLon float64 `json:"max_lon"`
}

// IsValid validates whether the bounding box fits standard coordinate constraints.
func (b BoundingBox) IsValid() bool {
	if b.MinLat < -90.0 || b.MinLat > 90.0 || b.MaxLat < -90.0 || b.MaxLat > 90.0 {
		return false
	}
	if b.MinLon < -180.0 || b.MinLon > 180.0 || b.MaxLon < -180.0 || b.MaxLon > 180.0 {
		return false
	}
	return b.MinLat <= b.MaxLat && b.MinLon <= b.MaxLon
}

// GeocodingQuerySpec defines the contract specification for incoming geocoding search queries.
type GeocodingQuerySpec struct {
	Query       string       `json:"query"`
	CountryCode string       `json:"country_code,omitempty"`
	PostalCode  string       `json:"postal_code,omitempty"`
	BBox        *BoundingBox `json:"bbox,omitempty"`
	Limit       int          `json:"limit,omitempty"`
	Language    string       `json:"language,omitempty"`
}

// ValidateGeocodingQuery validates and normalizes an incoming geocoding query spec.
func ValidateGeocodingQuery(spec GeocodingQuerySpec) (*GeocodingQuerySpec, error) {
	// Sanitize and check query string
	trimmedQuery := strings.TrimSpace(spec.Query)

	for _, r := range trimmedQuery {
		if unicode.IsControl(r) || r == '\n' || r == '\r' || r == '\t' || r == 0 {
			return nil, ErrInvalidCharacters
		}
	}

	if len(trimmedQuery) < 3 {
		return nil, ErrQueryTooShort
	}
	if len(trimmedQuery) > 256 {
		return nil, ErrQueryTooLong
	}

	sanitized := &GeocodingQuerySpec{
		Query: trimmedQuery,
	}

	// Validate country code if provided
	if spec.CountryCode != "" {
		trimmedCountry := strings.TrimSpace(spec.CountryCode)
		if !countryCodeRegex.MatchString(trimmedCountry) {
			return nil, ErrInvalidCountry
		}
		sanitized.CountryCode = strings.ToUpper(trimmedCountry)
	}

	// Validate postal code if provided
	if spec.PostalCode != "" {
		trimmedPostal := strings.TrimSpace(spec.PostalCode)
		if len(trimmedPostal) < 3 || len(trimmedPostal) > 10 || !postalCodeRegex.MatchString(trimmedPostal) {
			return nil, ErrInvalidPostalCode
		}
		sanitized.PostalCode = strings.ToUpper(trimmedPostal)
	}

	// Validate bounding box if provided
	if spec.BBox != nil {
		if !spec.BBox.IsValid() {
			return nil, ErrInvalidBBox
		}
		sanitized.BBox = &BoundingBox{
			MinLat: spec.BBox.MinLat,
			MinLon: spec.BBox.MinLon,
			MaxLat: spec.BBox.MaxLat,
			MaxLon: spec.BBox.MaxLon,
		}
	}

	// Limit bounds check
	if spec.Limit == 0 {
		sanitized.Limit = 10
	} else if spec.Limit < 1 || spec.Limit > 100 {
		return nil, ErrInvalidLimit
	} else {
		sanitized.Limit = spec.Limit
	}

	// Language normalization
	if spec.Language != "" {
		sanitized.Language = strings.ToLower(strings.TrimSpace(spec.Language))
	}

	return sanitized, nil
}

// Suffix mappings for US/International standard address normalization
var standardSuffixes = map[string]string{
	"st":    "Street",
	"st.":   "Street",
	"str":   "Street",
	"ave":   "Avenue",
	"ave.":  "Avenue",
	"blvd":  "Boulevard",
	"blvd.": "Boulevard",
	"rd":    "Road",
	"rd.":   "Road",
	"dr":    "Drive",
	"dr.":   "Drive",
	"ln":    "Lane",
	"ln.":   "Lane",
	"ct":    "Court",
	"ct.":   "Court",
	"pl":    "Place",
	"pl.":   "Place",
	"pkwy":  "Parkway",
	"pkwy.": "Parkway",
	"hwy":   "Highway",
	"hwy.":  "Highway",
	"apt":   "Apartment",
	"apt.":  "Apartment",
	"ste":   "Suite",
	"ste.":  "Suite",
	"fl":    "Floor",
	"fl.":   "Floor",
	"n":     "North",
	"n.":    "North",
	"s":     "South",
	"s.":    "South",
	"e":     "East",
	"e.":    "East",
	"w":     "West",
	"w.":    "West",
	"ne":    "Northeast",
	"ne.":   "Northeast",
	"nw":    "Northwest",
	"nw.":   "Northwest",
	"se":    "Southeast",
	"se.":   "Southeast",
	"sw":    "Southwest",
	"sw.":   "Southwest",
}

// NormalizeAddressString standardizes address abbreviations, removes duplicate spacing,
// and enforces canonical casing for geocoding consistency.
func NormalizeAddressString(raw string) string {
	// Strip control characters
	var cleaned strings.Builder
	for _, r := range raw {
		if !unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			cleaned.WriteRune(r)
		}
	}

	trimmed := strings.TrimSpace(cleaned.String())
	if trimmed == "" {
		return ""
	}

	// Normalize spaces
	squashed := multiSpaceRegex.ReplaceAllString(trimmed, " ")

	// Split tokens by commas to respect segment boundaries
	segments := strings.Split(squashed, ",")
	normSegments := make([]string, 0, len(segments))

	for _, seg := range segments {
		segTrim := strings.TrimSpace(seg)
		if segTrim == "" {
			continue
		}

		words := strings.Split(segTrim, " ")
		normWords := make([]string, 0, len(words))

		for _, word := range words {
			lowerWord := strings.ToLower(word)
			if expanded, exists := standardSuffixes[lowerWord]; exists {
				normWords = append(normWords, expanded)
			} else {
				// Title case the word unless it starts with '#' or contains digits
				if strings.HasPrefix(word, "#") {
					normWords = append(normWords, word)
				} else if containsDigit(word) {
					normWords = append(normWords, strings.ToUpper(word))
				} else {
					normWords = append(normWords, capitalizeWord(word))
				}
			}
		}

		normSegments = append(normSegments, strings.Join(normWords, " "))
	}

	return strings.Join(normSegments, ", ")
}

func containsDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func capitalizeWord(w string) string {
	if w == "" {
		return ""
	}
	runes := []rune(w)
	return fmt.Sprintf("%c%s", unicode.ToUpper(runes[0]), strings.ToLower(string(runes[1:])))
}
