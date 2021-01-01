package main

import (
	"flag"
	"math"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	adapterID := flag.String("adapter_id", "hci0", "-adapter_id hci0")
	flag.Parse()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	hciScanner := New(*adapterID, false)
	if err := hciScanner.Scan(); err != nil {
		log.Err(err).Msg("error")
	}
}

func New(adapterID string, beacon bool) *hciScan {
	return &hciScan{
		adapterID:   adapterID,
		beacon:      beacon,
		mu:          &sync.Mutex{},
		uniqDevices: make(map[string]*bluetoothDev, 2),
	}
}

func (hci *hciScan) Scan() error {
	defer func() {
		if err := api.Exit(); err != nil {
			log.Debug().Err(err)
		}
	}()

	deviceAdapter, err := adapter.GetAdapter(hci.adapterID)
	if err != nil {
		return err
	}

	log.Debug().Msg("Flush cached devices")
	err = deviceAdapter.FlushDevices()
	if err != nil {
		return err
	}

	log.Debug().Msg("Start discovery")
	discovery, cancel, err := api.Discover(deviceAdapter, nil)
	if err != nil {
		return err
	}
	defer cancel()

	t := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-t.C:
			log.Info().Msg("Bluetooth Devices")
			for _, k := range sortHCIAddresses(hci) {
				dev := hci.uniqDevices[k]
				if dev != nil {
					// rough approximation
					distanceMt := math.Pow(10.0, (-69.0-float64(hci.uniqDevices[k].rssi))/(10.0*2.0))
					log.Info().Msgf("addr=%16s,  alias=%36s,  rssi=%3d,  distance(m)=%.3f",
						hci.uniqDevices[k].addr, hci.uniqDevices[k].alias,
						hci.uniqDevices[k].rssi, distanceMt)
				}
			}
			log.Info().Msg("")
		case ev := <-discovery:
			if ev == nil {
				return nil
			}
			if err := hci.store(ev); err != nil {
				log.Error().Err(err)
			}
		}
	}

}

func sortHCIAddresses(hci *hciScan) []string {
	keys := make([]string, 0, len(hci.uniqDevices))
	for _, v := range hci.uniqDevices {
		if v != nil {
			keys = append(keys, v.addr)
		}
	}
	sort.Strings(keys)
	return keys
}
