package powerwall

import (
	"context"
	"encoding/json"
	"fmt"
)

type SimpleStatus struct {
	Leader            string         `json:"dinLeader"`
	Shutdown          bool           `json:"shutdown"`
	Island            bool           `json:"island"`
	BatteryEnergy     int            `json:"battery"`
	BatteryFullEnergy int            `json:"batteryFull"`
	PowerBattery      float64        `json:"powerBattery"`
	PowerSite         float64        `json:"powerSite"`
	PowerLoad         float64        `json:"powerLoad"`
	PowerSolar        float64        `json:"powerSolar"`
	PowerSolarRGM     float64        `json:"powerSolarRGM"`
	PowerGenerator    float64        `json:"powerGenerator"`
	PowerConductor    float64        `json:"powerConductor"`
	BatteryBlocks     []string       `json:"batteryBlocks"`
	Phase             [3]SimplePhase `json:"phase"`

	// Devices []SimpleStatusDevice `json:"devices"`
}

// type SimpleStatusDevice struct {
// 	DIN  string `json:"din"`
// 	Type string `json:"type"`
// }

type SimplePhase struct {
	FreqLoad    float64 `json:"freqLoad"`
	FreqMain    float64 `json:"freqMain"`
	VoltageLoad float64 `json:"voltageLoad"`
	VoltageMain float64 `json:"voltageMain"`
}

// GetSimpleStatus reads a [SimpleStatus] struct from your Powerwall system.
// This contains aggregate information from the leader.
func GetSimpleStatus(ctx context.Context, td *TEDApi) (status *SimpleStatus, err error) {
	out, err := td.Query(ctx, QueryStatus)
	if err != nil {
		return nil, err
	}

	type statusResponse struct {
		Control struct {
			Alerts struct {
				Active []string `json:"active"`
			} `json:"alerts"`
			BatteryBlocks []struct {
				DIN            string   `json:"din"`
				DisableReasons []string `json:"disableReasons"`
			} `json:"batteryBlocks"`
			Islanding struct {
				ContactorClosed    bool     `json:"contactorClosed"`
				CustomerIslandMode string   `json:"customerIslandMode"`
				DisableReasons     []string `json:"disableReasons"`
				GridOK             bool     `json:"gridOK"`
				MicroGridOK        bool     `json:"microGridOK"`
			} `json:"islanding"`
			MeterAggregates []struct {
				Location   string  `json:"location"`
				RealPowerW float64 `json:"realPowerW"`
			} `json:"meterAggregates"`
			PVInverters []struct {
				// TODO: not sure what goes here - maybe for PW2?
			} `json:"pvInverters"`
			SiteShutdown struct {
				IsShutdown bool     `json:"isShutDown"`
				Reasons    []string `json:"reasons"`
			} `json:"siteShutdown"`
			SystemStatus struct {
				// these usually show up as int but rarely have e.g., .0000000004
				NominalEnergyRemainingWh float64 `json:"nominalEnergyRemainingWh"`
				NominalFullPackEnergyWh  float64 `json:"nominalFullPackEnergyWh"`
			} `json:"systemStatus"`
		} `json:"control"`

		EsCan struct {
			Bus struct {
				Islander struct {
					AcMeasurements map[string]any `json:"ISLAND_AcMeasurements"`
				} `json:"ISLANDER"`
			} `json:"bus"`
		} `json:"esCan"`
	}
	var response statusResponse

	err = json.Unmarshal(out, &response)
	if err != nil {
		return nil, err
	}

	// var raw map[string]any
	// json.Unmarshal(out, &raw)
	// nice, _ := json.MarshalIndent(raw, "", "  ")
	// fmt.Printf("%s\n", string(nice))

	powerFor := func(s string) float64 {
		for _, m := range response.Control.MeterAggregates {
			if m.Location == s {
				return m.RealPowerW
			}
		}
		return 0.0
	}

	din, err := td.getDIN(ctx)
	if err != nil {
		return nil, err
	}

	status = &SimpleStatus{
		Leader:            din,
		Shutdown:          response.Control.SiteShutdown.IsShutdown,
		Island:            !response.Control.Islanding.ContactorClosed,
		BatteryEnergy:     int(response.Control.SystemStatus.NominalEnergyRemainingWh),
		BatteryFullEnergy: int(response.Control.SystemStatus.NominalFullPackEnergyWh),
		PowerBattery:      powerFor("BATTERY"),
		PowerSite:         powerFor("SITE"),
		PowerLoad:         powerFor("LOAD"),
		PowerSolar:        powerFor("SOLAR"),
		PowerSolarRGM:     powerFor("SOLAR_RGM"),
		PowerGenerator:    powerFor("GENERATOR"),
		PowerConductor:    powerFor("CONDUCTOR"),
	}
	for _, bb := range response.Control.BatteryBlocks {
		status.BatteryBlocks = append(status.BatteryBlocks, bb.DIN)
	}

	for i := range 3 {
		phase := i + 1

		ac := response.EsCan.Bus.Islander.AcMeasurements
		floatFor := func(s string) (out float64) {
			out, _ = ac[s].(float64)
			return
		}

		status.Phase[i] = SimplePhase{
			FreqLoad:    floatFor(fmt.Sprintf("ISLAND_FreqL%d_Load", phase)),
			FreqMain:    floatFor(fmt.Sprintf("ISLAND_FreqL%d_Main", phase)),
			VoltageLoad: floatFor(fmt.Sprintf("ISLAND_VL%dN_Load", phase)),
			VoltageMain: floatFor(fmt.Sprintf("ISLAND_VL%dN_Main", phase)),
		}

	}

	return status, nil
}

// func GetStatus(ctx context.Context, td *TEDApi) (err error) {
// 	out, err := td.Query(ctx, QueryStatus)
// 	if err != nil {
// 		return err
// 	}

// 	nice, _ := json.MarshalIndent(out, "", "  ")
// 	log.Printf("status=\n%s\n", nice)

// 	return err
// }

// func GetConfig(ctx context.Context, td *TEDApi) (err error) {
// 	out, err := td.Config(ctx, "config.json")
// 	if err != nil {
// 		return err
// 	}

// 	log.Printf("config=\n%s\n", out)
// 	return nil
// }
