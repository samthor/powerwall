package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
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
	log.Printf("Battery: %.2f%% (%.2f / %.2f kWh), %s", float64(status.BatteryEnergy)/float64(status.BatteryFullEnergy)*100.0, float64(status.BatteryEnergy)/1000.0, float64(status.BatteryFullEnergy)/1000.0, batteryDuration)
	log.Printf("")
	log.Printf("SOLAR   %6.2f kW", status.PowerSolar/1000.0)
	log.Printf("LOAD    %6.2f kW", status.PowerLoad/1000.0)
	log.Printf("GATE    %6.2f kW%s", status.PowerSite/1000.0, siteSuffix)
	log.Printf("BATTERY %6.2f kW%s", status.PowerBattery/1000.0, batterySuffix)
	log.Printf("")
	for i, phase := range status.Phase {
		log.Printf("[%d] %6.2fv %5.2fHz", (i + 1), phase.VoltageLoad, phase.FreqLoad)
	}
	log.Printf("")

	// b, err := json.MarshalIndent(status, "", "  ")
	// if err != nil {
	// 	log.Fatalf("couldn't JSON-encode status: %v", err)
	// }
	// fmt.Printf("%s\n", string(b))
}
