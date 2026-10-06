package services_test

import (
	"math"
	"testing"

	"github.com/edoazn/absensi-go/internal/services"
)

var geo = services.NewGeolocationService()

func TestDistanceSamePointIsZero(t *testing.T) {
	if dist := geo.CalculateDistance(-6.2, 106.816666, -6.2, 106.816666); math.Abs(dist) > 1e-9 {
		t.Fatalf("same point distance = %v, want ~0", dist)
	}
}

func TestDistanceIsSymmetric(t *testing.T) {
	pairs := [][4]float64{
		{-6.2, 106.816666, -6.2015, 106.818},
		{0.0, 0.0, 0.5, 0.7},
		{51.5074, -0.1278, 48.8566, 2.3522},
	}
	for _, p := range pairs {
		forward := geo.CalculateDistance(p[0], p[1], p[2], p[3])
		backward := geo.CalculateDistance(p[2], p[3], p[0], p[1])
		if math.Abs(forward-backward) > 1e-6 {
			t.Fatalf("distance not symmetric: %v vs %v", forward, backward)
		}
	}
}

func TestDistanceIsDeterministic(t *testing.T) {
	first := geo.CalculateDistance(-6.175, 106.827, -6.3018, 106.6529)
	for range 5 {
		if again := geo.CalculateDistance(-6.175, 106.827, -6.3018, 106.6529); again != first {
			t.Fatalf("distance not deterministic: %v vs %v", first, again)
		}
	}
}

func TestOneDegreeLatitudeIsAbout111km(t *testing.T) {
	dist := geo.CalculateDistance(0, 100, 1, 100)
	expected := services.EarthRadiusMeters * math.Pi / 180
	if math.Abs(dist-expected) > 1 {
		t.Fatalf("1 degree lat = %v, want ~%v", dist, expected)
	}
}

func TestTriangleInequality(t *testing.T) {
	a := [2]float64{-6.20, 106.81}
	b := [2]float64{-6.21, 106.83}
	c := [2]float64{-6.19, 106.80}

	ab := geo.CalculateDistance(a[0], a[1], b[0], b[1])
	bc := geo.CalculateDistance(b[0], b[1], c[0], c[1])
	ac := geo.CalculateDistance(a[0], a[1], c[0], c[1])

	if ac > ab+bc+0.001 {
		t.Fatalf("triangle inequality violated: ac=%v > ab+bc=%v", ac, ab+bc)
	}
}

func TestIsWithinRadiusBoundaryInclusive(t *testing.T) {
	loc := [2]float64{-6.2, 106.816666}
	radius := 100.0

	north := geo.CalculateDistance(loc[0]+100.0/services.EarthRadiusMeters*180/math.Pi, loc[1], loc[0], loc[1])
	if north > radius+0.01 {
		t.Fatalf("boundary calibration off: %v", north)
	}
	if !geo.IsWithinRadius(loc[0], loc[1], loc[0], loc[1], radius) {
		t.Fatal("exact location must be within radius")
	}
	if geo.IsWithinRadius(loc[0]+0.01, loc[1], loc[0], loc[1], radius) {
		t.Fatal("0.01 degree (~1.1km) must be outside radius")
	}
}
