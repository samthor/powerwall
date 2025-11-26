package powerwall

import (
	"fmt"
	"math"
)

func FormatPowerTable(watts float64) (s string) {
	if math.Abs(watts) <= 10 {
		return "     - kW"
	}
	return fmt.Sprintf("%6.2f kW", watts/1000.0)
}
