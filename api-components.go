package powerwall

import (
	"context"
	"encoding/json"
	"fmt"
)

type SimpleDeviceStatus struct {
	BatteryEnergy     int                      `json:"battery"`
	BatteryFullEnergy int                      `json:"batteryFull"`
	PowerBattery      float64                  `json:"powerBattery"`
	PowerSolar        float64                  `json:"powerSolar"`
	Freq              float64                  `json:"freq"`
	Voltage           float64                  `json:"voltage"`
	MPPT              []SimpleDeviceMPPTStatus `json:"mppt"`
}

type SimpleDeviceMPPTStatus struct {
	Current float64 `json:"c"`
	Voltage float64 `json:"v"`
}

func (s SimpleDeviceMPPTStatus) FormatPower() (out string) {
	w := s.Current * s.Voltage
	if w < 1.0 {
		return "-"
	}
	return fmt.Sprintf("%.0fw", w)
}

type rawSignal struct {
	Name  string   `json:"name"`
	Value *float64 `json:"value,omitzero"`
}

type rawSignalArray []rawSignal

func (ra rawSignalArray) ToMap() map[string]float64 {
	signalMap := make(map[string]float64)
	for _, s := range ra {
		if s.Value != nil {
			signalMap[s.Name] = *s.Value
		}
	}
	return signalMap
}

// GetSimpleDeviceStatus reads a [SimpleDeviceStatus] struct from an individual device.
// For a PW3, this contains its charge etc plus the status of its MPPTs.
// You must pass individual DINs (get from [SimpleStatus]).
func GetSimpleDeviceStatus(ctx context.Context, td *TEDApi, din string) (status *SimpleDeviceStatus, err error) {
	var out []byte
	out, err = td.QueryDevice(ctx, QueryComponents, din)
	if err != nil {
		return
	}

	type componentPart struct {
		ActiveAlerts []struct {
			Name string `json:"name"`
		} `json:"activeAlerts"`
		Signals rawSignalArray `json:"signals"`
	}

	type componentsResponse struct {
		Components struct {
			BMS []componentPart `json:"bms"`
			PCH []componentPart `json:"pch"`
		} `json:"components"`
	}

	var r componentsResponse
	json.Unmarshal(out, &r)

	// var m map[string]any
	// json.Unmarshal(out, &m)
	// b, _ := json.MarshalIndent(m, "", "  ")
	// log.Printf("RAW components: %s => %s", din, string(b))

	if len(r.Components.BMS) < 1 {
		return nil, fmt.Errorf("could not get BMS data from device")
	}

	// battery energy
	bmsMap := r.Components.BMS[0].Signals.ToMap()
	energyKw, ok1 := bmsMap["BMS_nominalEnergyRemaining"]
	fullEnergyKw, ok2 := bmsMap["BMS_nominalFullPackEnergy"]
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("could not get battery energy data")
	}

	status = &SimpleDeviceStatus{
		BatteryEnergy:     int(energyKw * 1000.0),
		BatteryFullEnergy: int(fullEnergyKw * 1000.0),
	}

	// solar status (PW3 only probably)

	switch len(r.Components.PCH) {
	case 0:
		// nothing
	case 1:
		// ok
		only := r.Components.PCH[0].Signals.ToMap()

		status.PowerSolar = only["PCH_SlowPvPowerSum"]
		status.PowerBattery = only["PCH_BatteryPower"]
		status.Freq = only["PCH_AcFrequency"]
		status.Voltage = only["PCH_AcVoltageAB"] // also has to N from A/B

		// in Australia this is sold as 3, but they're just pairs of two doing half duty each
		for i := range 6 {
			char := ('A' + i)
			current, ok1 := only[fmt.Sprintf("PCH_PvCurrent%c", char)]
			voltage, ok2 := only[fmt.Sprintf("PCH_PvVoltage%c", char)]
			if !ok1 && !ok2 {
				break
			}

			status.MPPT = append(status.MPPT, SimpleDeviceMPPTStatus{
				Current: current,
				Voltage: voltage,
			})
		}

	case 2:
		return nil, fmt.Errorf("got multiple PCH: %v", len(r.Components.PCH))
	}

	return status, nil
}
