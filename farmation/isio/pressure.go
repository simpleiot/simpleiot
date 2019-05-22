package isio

// We have a pressure sensor with ratiometric output
// from 0.5V to 4.5V for 5V excitation. 250PSI

// CalcPressure calculates pressure from ratiometric
// pressure sensor
func CalcPressure(ref, sense, fullScale float64) float64 {
	min := 0.5 * ref / 5
	max := 4.5 * ref / 5
	span := max - min
	v := sense - min
	return v * fullScale / span
}
