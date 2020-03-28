package data

import (
	nmea "github.com/adrianmo/go-nmea"
)

// GpsPos describes location and fix information from a GPS
type GpsPos struct {
	Lat    float64 `json:"lat"`
	Long   float64 `json:"long"`
	Fix    string  `json:"fix"`
	NumSat int64   `json:"numSat"`
}

// FromGPGGA converts a GPGGA string to a position/fix
func (p *GpsPos) FromGPGGA(gpgga nmea.GPGGA) {
	p.Lat = gpgga.Latitude
	p.Long = gpgga.Longitude
	p.Fix = gpgga.FixQuality
	p.NumSat = gpgga.NumSatellites
}
