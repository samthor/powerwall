package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/samthor/powerwall"
)

var (
	flagTeslaSecret = flag.String("gw_pw", "", "Powerwall secret")
	flagMin         = flag.Float64("min", 100, "minimum wH to report status for")
	flagRemote      = flag.String("host", "192.168.91.1:443", "default Tesla remote")
)

func main() {
	flag.Parse()

	api := &powerwall.TEDApi{Secret: *flagTeslaSecret, Remote: *flagRemote}
	status, err := powerwall.GetSimpleStatus(context.Background(), api)
	if err != nil {
		log.Fatalf("could not read status: %v", err)
	}

	byDevice := map[string]powerwall.SimpleDeviceStatus{}
	if len(status.BatteryBlocks) > 1 {
		// demo fetches individual devices only if you have multiple
		for _, din := range status.BatteryBlocks {
			res, err := powerwall.GetSimpleDeviceStatus(context.Background(), api, din)
			if err != nil {
				log.Fatalf("failed to lookup device %s: %v", din, err)
			}
			byDevice[din] = *res
		}
	}

	batteryDuration := "effectively idle"

	batteryIsCharging := status.PowerBattery < 0.0
	var batterySuffix string

	// show battery time to full/empty if > 100w of change
	if math.Abs(status.PowerBattery) > *flagMin {
		if batteryIsCharging {
			uncharged := status.BatteryFullEnergy - status.BatteryEnergy
			duration := time.Duration(float64(time.Hour) * float64(uncharged) / -status.PowerBattery)
			batteryDuration = fmt.Sprintf("%v until full", duration.Round(time.Second))
			batterySuffix = " (charging)"
		} else {
			duration := time.Duration(float64(time.Hour) * float64(status.BatteryEnergy) / status.PowerBattery)
			batteryDuration = fmt.Sprintf("%v until empty", duration.Round(time.Second))
			batterySuffix = " (discharging)"
		}
	}

	var siteSuffix string
	if status.PowerSite < -*flagMin {
		siteSuffix = " (exporting)"
	} else if status.PowerSite > *flagMin {
		siteSuffix = " (importing)"
	}

	log.Printf("")
	log.Printf("System (Island=%v, Shutdown=%v)", status.Island, status.Shutdown)
	log.Printf("%.2f%% (%.2f / %.2f kWh), %s", float64(status.BatteryEnergy)/float64(status.BatteryFullEnergy)*100.0, float64(status.BatteryEnergy)/1000.0, float64(status.BatteryFullEnergy)/1000.0, batteryDuration)
	log.Printf("")
	log.Printf("SOLAR   %s", powerwall.FormatPowerTable(status.PowerSolar))
	log.Printf("BATTERY %s%s", powerwall.FormatPowerTable(status.PowerBattery), batterySuffix)
	log.Printf("GATE    %s%s", powerwall.FormatPowerTable(status.PowerSite), siteSuffix)
	log.Printf("LOAD    %s", powerwall.FormatPowerTable(status.PowerLoad))
	log.Printf("")
	for i, phase := range status.Phase {
		log.Printf("[%d] %6.2fv %5.2fHz", (i + 1), phase.VoltageLoad, phase.FreqLoad)
	}

	for _, din := range status.BatteryBlocks {
		status := byDevice[din]

		var mpptParts []string
		for _, mppt := range status.MPPT {
			mpptParts = append(mpptParts, mppt.FormatPower())
		}

		log.Printf("")
		log.Printf("[%s] %.2f%% (%.2f / %.2f kWh)", din, float64(status.BatteryEnergy)/float64(status.BatteryFullEnergy)*100.0, float64(status.BatteryEnergy)/1000.0, float64(status.BatteryFullEnergy)/1000.0)
		log.Printf("")
		log.Printf("  SOLAR   %s (%s)", powerwall.FormatPowerTable(status.PowerSolar), strings.Join(mpptParts, " "))
		log.Printf("  BATTERY %s", powerwall.FormatPowerTable(status.PowerBattery))
		log.Printf("")
		log.Printf("  %6.2fv %5.2fHz", status.Voltage, status.Freq)
	}
	log.Printf("")

	// b, err := json.MarshalIndent(status, "", "  ")
	// if err != nil {
	// 	log.Fatalf("couldn't JSON-encode status: %v", err)
	// }
	// fmt.Printf("%s\n", string(b))
}
