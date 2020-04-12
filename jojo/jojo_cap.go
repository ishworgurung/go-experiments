// This package is a helper to add and remove common Linux capabilities.
// The common Linux capabilities are documented in `uapi/linux/capabilities.h`.
// To add the capability to bind to low port:
// 	capNetBind := new(CapNetBindService)
//	capNetBind.AddCap()
//	defer capNetBind.RemoveCap()

package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/ishworgurung/libcap/cap"
)

type Capable interface {
	AddCap() error
	RemoveCap()
}

type CapNetBindService struct {
	set *cap.Set
}

type CapRaw struct {
	set *cap.Set
}

// AddCap add the ability to bind the calling process to low (<1024) port.
func (capabilities CapNetBindService) AddCap() error {
	// Craft a duplicated capabilities
	dupCapabilities, err := cap.GetProc().Dup()
	if err != nil {
		return err
	}
	if on, err := dupCapabilities.GetFlag(cap.Permitted, cap.NET_BIND_SERVICE); !on {
		if err != nil {
			return err
		} else {
			return errors.New(fmt.Sprintf(
				"insufficient privilege to bind to low ports - want %q, have %q",
				cap.NET_BIND_SERVICE, dupCapabilities))
		}
	}
	if err := dupCapabilities.SetFlag(cap.Effective, true, cap.NET_BIND_SERVICE); err != nil {
		return errors.New(fmt.Sprintf(
			"unable to set capability: %q", err))
	}
	if err := dupCapabilities.SetProc(); err != nil {
		return errors.New(fmt.Sprintf(
			"unable to raise capabilities: %q", err))
	}
	capabilities.set = dupCapabilities
	return nil
}

func (capabilities CapNetBindService) RemoveCap() {
	if err := capabilities.set.SetProc(); err != nil {
		log.Fatal(err)
	}
}

// addCapNetBindService add the ability to bind the calling process to low (<1024) port.
func (capabilities CapRaw) AddCap() error {
	// TODO
	return nil
}

func (capabilities CapRaw) RemoveCap() {
	// TODO
}
