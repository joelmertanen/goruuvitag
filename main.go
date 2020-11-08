package main

import (
	"bytes"
	"encoding/binary"

	"fmt"
	"log"
	"sync"
	"time"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
)

var isPoweredOn = false
var scanMutex = sync.Mutex{}

func beginScan(d gatt.Device) {
	scanMutex.Lock()
	for isPoweredOn {
		d.Scan(nil, true) //Scan for five seconds and then restart
		time.Sleep(5 * time.Second)
		d.StopScanning()
	}
	scanMutex.Unlock()
}

func onStateChanged(d gatt.Device, s gatt.State) {
	fmt.Println("State:", s)
	switch s {
	case gatt.StatePoweredOn:
		fmt.Println("scanning...")
		isPoweredOn = true
		go beginScan(d)
		return
	case gatt.StatePoweredOff:
		log.Println("REINIT ON POWER OFF")
		isPoweredOn = false
		d.Init(onStateChanged)
	default:
		log.Println("WARN: unhandled state: ", string(s))
	}
}

func onPeriphDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
	if (p.ID() != "FA:C2:A9:CF:DB:55") {
		return
	}

	fmt.Printf("\nPeripheral ID:%s, NAME:(%+v)\n", p.ID(), p.Device())
	fmt.Println("  TX Power Level    =", a.TxPowerLevel)
	fmt.Printf("%d\n", a.ManufacturerData)
	
	reader := bytes.NewReader(a.ManufacturerData)
	result := SensorFormat3{}
	err := binary.Read(reader, binary.BigEndian, &result)
	
	if err == nil {
		fmt.Printf("%+v\n", result)
	}
	
	if !IsRuuviTag(a.ManufacturerData) || err != nil {
		return
	}
	fmt.Printf("\nPeripheral ID:%s, NAME:(%s)\n", p.ID(), p.Name())
	fmt.Println("  TX Power Level    =", a.TxPowerLevel)
	ParseRuuviData(a.ManufacturerData, p.ID())
}

func main() {
	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	// Register handlers.
	d.Handle(gatt.PeripheralDiscovered(onPeriphDiscovered))
	d.Init(onStateChanged)
	select {}
}
