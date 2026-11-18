package api

import (
	"testing"
)

func TestScrapeRequest_Validate(t *testing.T) {
	t.Run("valid minimal request", func(t *testing.T) {
		req := &ScrapeRequest{
			Keyword: "coffee shop",
		}
		req.SetDefaults()
		if err := req.Validate(); err != nil {
			t.Fatalf("expected valid request, got: %v", err)
		}
		if req.Lang != "en" || req.MaxDepth != 1 || req.Timeout != 300 {
			t.Fatalf("unexpected defaults: %+v", req)
		}
	})

	t.Run("whitespace keyword rejected", func(t *testing.T) {
		req := &ScrapeRequest{
			Keyword: "   \t\n ",
		}
		req.SetDefaults()
		if err := req.Validate(); err == nil {
			t.Fatal("expected error for whitespace keyword, got nil")
		}
	})

	t.Run("valid geo coordinates with whitespace trimming", func(t *testing.T) {
		req := &ScrapeRequest{
			Keyword:        "museum",
			GeoCoordinates: "  40.7128 , -74.0060  ",
		}
		req.SetDefaults()
		if err := req.Validate(); err != nil {
			t.Fatalf("expected valid geo coordinates, got: %v", err)
		}
	})

	t.Run("malformed geo coordinates comma count", func(t *testing.T) {
		req := &ScrapeRequest{
			Keyword:        "museum",
			GeoCoordinates: "40.7128,-74.0060,10.0",
		}
		if err := req.Validate(); err == nil {
			t.Fatal("expected error for extra comma, got nil")
		}
	})

	t.Run("malformed geo coordinates empty parts", func(t *testing.T) {
		req := &ScrapeRequest{
			Keyword:        "museum",
			GeoCoordinates: "40.7128,",
		}
		if err := req.Validate(); err == nil {
			t.Fatal("expected error for missing longitude, got nil")
		}
	})

	t.Run("latitude out of range", func(t *testing.T) {
		req := &ScrapeRequest{
			Keyword:        "museum",
			GeoCoordinates: "95.0,20.0",
		}
		if err := req.Validate(); err == nil {
			t.Fatal("expected error for lat > 90, got nil")
		}
	})

	t.Run("longitude out of range", func(t *testing.T) {
		req := &ScrapeRequest{
			Keyword:        "museum",
			GeoCoordinates: "45.0,195.0",
		}
		if err := req.Validate(); err == nil {
			t.Fatal("expected error for lon > 180, got nil")
		}
	})

	t.Run("fast mode requirements", func(t *testing.T) {
		req := &ScrapeRequest{
			Keyword:  "bakery",
			FastMode: true,
		}
		if err := req.Validate(); err == nil {
			t.Fatal("expected error for missing geo in fast mode, got nil")
		}

		req.GeoCoordinates = "40.7128,-74.0060"
		if err := req.Validate(); err == nil {
			t.Fatal("expected error for missing zoom in fast mode, got nil")
		}

		req.Zoom = 14
		if err := req.Validate(); err != nil {
			t.Fatalf("expected valid fast mode, got: %v", err)
		}
	})

	t.Run("timeout boundary checks", func(t *testing.T) {
		req := &ScrapeRequest{
			Keyword: "park",
			Timeout: -1,
		}
		if err := req.Validate(); err == nil {
			t.Fatal("expected error for negative timeout, got nil")
		}

		req.Timeout = 500
		if err := req.Validate(); err == nil {
			t.Fatal("expected error for timeout > 300, got nil")
		}

		req.Timeout = 120
		if err := req.Validate(); err != nil {
			t.Fatalf("expected valid timeout, got: %v", err)
		}
	})
}
