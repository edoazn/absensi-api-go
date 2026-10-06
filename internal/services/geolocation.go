package services

import "math"

const EarthRadiusMeters = 6371000.0

type GeolocationService struct{}

func NewGeolocationService() *GeolocationService {
	return &GeolocationService{}
}

func (g *GeolocationService) CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := degToRad(lat1)
	lon1Rad := degToRad(lon1)
	lat2Rad := degToRad(lat2)
	lon2Rad := degToRad(lon2)

	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	a := math.Pow(math.Sin(deltaLat/2), 2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Pow(math.Sin(deltaLon/2), 2)
	c := 2 * math.Asin(math.Sqrt(a))

	return EarthRadiusMeters * c
}

func (g *GeolocationService) IsWithinRadius(userLat, userLon, locationLat, locationLon, radius float64) bool {
	return g.CalculateDistance(userLat, userLon, locationLat, locationLon) <= radius
}

func degToRad(deg float64) float64 {
	return deg * math.Pi / 180
}
