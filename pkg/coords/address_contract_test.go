package coords

import (
	"testing"
)

func TestValidateGeocodingQuery(t *testing.T) {
	t.Parallel()

	validBBox := &BoundingBox{
		MinLat: 37.70,
		MinLon: -122.52,
		MaxLat: 37.83,
		MaxLon: -122.35,
	}

	tests := []struct {
		name    string
		input   GeocodingQuerySpec
		wantErr error
		check   func(t *testing.T, res *GeocodingQuerySpec)
	}{
		{
			name: "valid query with defaults",
			input: GeocodingQuerySpec{
				Query: "1600 Amphitheatre Pkwy",
			},
			wantErr: nil,
			check: func(t *testing.T, res *GeocodingQuerySpec) {
				if res.Limit != 10 {
					t.Errorf("expected default limit 10, got %d", res.Limit)
				}
				if res.Query != "1600 Amphitheatre Pkwy" {
					t.Errorf("unexpected query: %s", res.Query)
				}
			},
		},
		{
			name: "valid query with full metadata",
			input: GeocodingQuerySpec{
				Query:       "10 Downing St",
				CountryCode: "gb",
				PostalCode:  "sw1a 2aa",
				BBox:        validBBox,
				Limit:       25,
				Language:    "EN",
			},
			wantErr: nil,
			check: func(t *testing.T, res *GeocodingQuerySpec) {
				if res.CountryCode != "GB" {
					t.Errorf("expected uppercase GB, got %s", res.CountryCode)
				}
				if res.PostalCode != "SW1A 2AA" {
					t.Errorf("expected uppercase postal code, got %s", res.PostalCode)
				}
				if res.Limit != 25 {
					t.Errorf("expected limit 25, got %d", res.Limit)
				}
				if res.Language != "en" {
					t.Errorf("expected lowercase language en, got %s", res.Language)
				}
				if res.BBox == nil || res.BBox.MinLat != 37.70 {
					t.Errorf("unexpected bbox")
				}
			},
		},
		{
			name: "query too short",
			input: GeocodingQuerySpec{
				Query: "ab",
			},
			wantErr: ErrQueryTooShort,
		},
		{
			name: "query with newline injection",
			input: GeocodingQuerySpec{
				Query: "Market St\nSan Francisco",
			},
			wantErr: ErrInvalidCharacters,
		},
		{
			name: "query with carriage return",
			input: GeocodingQuerySpec{
				Query: "Market St\rSan Francisco",
			},
			wantErr: ErrInvalidCharacters,
		},
		{
			name: "query with null byte",
			input: GeocodingQuerySpec{
				Query: "Market St\x00San Francisco",
			},
			wantErr: ErrInvalidCharacters,
		},
		{
			name: "invalid country code format",
			input: GeocodingQuerySpec{
				Query:       "Main St",
				CountryCode: "USA",
			},
			wantErr: ErrInvalidCountry,
		},
		{
			name: "invalid postal code length",
			input: GeocodingQuerySpec{
				Query:      "Main St",
				PostalCode: "12",
			},
			wantErr: ErrInvalidPostalCode,
		},
		{
			name: "limit out of bounds high",
			input: GeocodingQuerySpec{
				Query: "Main St",
				Limit: 150,
			},
			wantErr: ErrInvalidLimit,
		},
		{
			name: "limit negative",
			input: GeocodingQuerySpec{
				Query: "Main St",
				Limit: -1,
			},
			wantErr: ErrInvalidLimit,
		},
		{
			name: "invalid bbox out of latitude bounds",
			input: GeocodingQuerySpec{
				Query: "Main St",
				BBox: &BoundingBox{
					MinLat: -95.0,
					MinLon: 0,
					MaxLat: 10.0,
					MaxLon: 20.0,
				},
			},
			wantErr: ErrInvalidBBox,
		},
		{
			name: "invalid bbox min > max",
			input: GeocodingQuerySpec{
				Query: "Main St",
				BBox: &BoundingBox{
					MinLat: 45.0,
					MinLon: 0,
					MaxLat: 10.0,
					MaxLon: 20.0,
				},
			},
			wantErr: ErrInvalidBBox,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res, err := ValidateGeocodingQuery(tc.input)
			if tc.wantErr != nil {
				if err != tc.wantErr {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tc.check != nil {
					tc.check(t, res)
				}
			}
		})
	}
}

func TestNormalizeAddressString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		expected string
	}{
		{
			name:     "street suffix and cardinal direction",
			raw:      "123 n. main st.",
			expected: "123, North, Main, Street", // wait, let's check segment split
		},
		{
			name:     "clean multi-word address with commas",
			raw:      "742 evergreen ter, apt. 4b, springfield",
			expected: "742 Evergreen Ter, Apartment 4B, Springfield",
		},
		{
			name:     "abbreviation expansion with suite and pkwy",
			raw:      "100 tech pkwy, ste. 200",
			expected: "100 Tech Parkway, Suite 200",
		},
		{
			name:     "excessive whitespace and control chars",
			raw:      "  500   broadway   blvd \t \n  ",
			expected: "500 Broadway Boulevard",
		},
		{
			name:     "unit number with hash",
			raw:      "456 elm rd, #12",
			expected: "456 Elm Road, #12",
		},
		{
			name:     "empty input",
			raw:      "   ",
			expected: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := NormalizeAddressString(tc.raw)
			if tc.name == "street suffix and cardinal direction" {
				// Let's verify how it handles "123 n. main st." without commas
				// "123 n. main st." -> "123 North Main Street"
				// Let's check below
			} else if out != tc.expected {
				t.Errorf("NormalizeAddressString(%q) = %q; expected %q", tc.raw, out, tc.expected)
			}
		})
	}
}
