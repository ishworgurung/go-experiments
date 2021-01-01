package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/muka/go-bluetooth/api/beacon"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	"github.com/muka/go-bluetooth/bluez/profile/device"
	"github.com/rs/zerolog/log"
	eddystone "github.com/suapapa/go_eddystone"
)

type hciScan struct {
	adapterID   string
	beacon      bool
	mu          *sync.Mutex
	uniqDevices map[string]*bluetoothDev
}

type bluetoothDev struct {
	addr  string
	alias string
	rssi  int16
}

func (hci *hciScan) store(ev *adapter.DeviceDiscovered) error {
	if ev.Type == adapter.DeviceRemoved {
		log.Debug().Msgf("[dev removed] %s", ev.Path)
	}
	dev, err := device.NewDevice1(ev.Path)
	if err != nil {
		return err
	}
	if dev == nil {
		return fmt.Errorf("[new dev] %s: not found", ev.Path)
	}

	// HCI Bluetooth Device
	hci.update(dev.Properties.Address, dev.Properties.Alias, dev.Properties.RSSI)
	go func(ev *adapter.DeviceDiscovered) {
		if err := hci.handleBeacon(dev); err != nil {
			log.Error().Msgf("[beacon] %s: %s", ev.Path, err)
		}
	}(ev)
	return err
}

func (hci *hciScan) update(addr string, alias string, rssi int16) {
	dev := &bluetoothDev{
		addr:  addr,
		alias: alias,
		rssi:  rssi,
	}
	hci.mu.Lock()
	hci.uniqDevices[addr] = dev
	hci.mu.Unlock()
}

func (hci *hciScan) handleBeacon(dev *device.Device1) error {
	b, err := beacon.NewBeacon(dev)
	if err != nil {
		return err
	}

	beaconUpdated, err := b.WatchDeviceChanges(context.Background())
	if err != nil {
		return err
	}

	isBeacon := <-beaconUpdated
	if !isBeacon {
		log.Debug().Msgf("got something weird. is a beacon: %v", isBeacon)
		return nil
	}

	name := b.Device.Properties.Alias
	if name == "" {
		name = b.Device.Properties.Name
	}

	log.Debug().Msgf("Found beacon %s %s", b.Type, name)

	if b.IsEddystone() {
		ed := b.GetEddystone()
		switch ed.Frame {
		case eddystone.UID:
			log.Debug().Msgf(
				"Eddystone UID %s instance %s (%ddbi)",
				ed.UID,
				ed.InstanceUID,
				ed.CalibratedTxPower,
			)
			break
		case eddystone.TLM:
			log.Debug().Msgf(
				"Eddystone TLM temp:%.0f batt:%d last reboot:%d advertising pdu:%d (%ddbi)",
				ed.TLMTemperature,
				ed.TLMBatteryVoltage,
				ed.TLMLastRebootedTime,
				ed.TLMAdvertisingPDU,
				ed.CalibratedTxPower,
			)
			break
		case eddystone.URL:
			log.Debug().Msgf(
				"Eddystone URL %s (%ddbi)",
				ed.URL,
				ed.CalibratedTxPower,
			)
			break
		}

	}
	if b.IsIBeacon() {
		ibeacon := b.GetIBeacon()
		log.Debug().Msgf(
			"IBeacon %s (%ddbi) (major=%d minor=%d)",
			ibeacon.ProximityUUID,
			ibeacon.MeasuredPower,
			ibeacon.Major,
			ibeacon.Minor,
		)
	}

	return nil
}
