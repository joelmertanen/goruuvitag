package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
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
	log.Println("State:", s)
	switch s {
	case gatt.StatePoweredOn:
		log.Println("Scanning...")
		isPoweredOn = true
		go beginScan(d)
		return
	case gatt.StatePoweredOff:
		log.Println("REINIT ON POWER OFF")
		isPoweredOn = false
		d.Init(onStateChanged)
	default:
		log.Println("WARN: unhandled state: ", fmt.Sprint(s))
	}
}

func onPeripheralDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
	if !IsRuuviTag(a.ManufacturerData) {
		return
	}

	log.Printf("Peripheral ID:%s, NAME:(%s)\n", p.ID(), p.Name())
	sensorData, err := ParseRuuviData(a.ManufacturerData, p.ID())
	if err != nil {
		log.Fatal(err)
		return
	}

	StoreSensorData(sensorData)
}

func createSysInfoSender() chan bool {
	SendSysInfo()
	log.Println("Sent system info")

	sysInfoTicker := time.NewTicker(10 * time.Second)
	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-sysInfoTicker.C:
				SendSysInfo()
				log.Println("Sent system info")
			case <-quit:
				sysInfoTicker.Stop()
				return
			}
		}
	}()
	return quit
}

func main() {
	InitializeClient()
	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open bluetooth device, err: %s\n", err)
		os.Exit(1)
	}

	stopSysInfo := createSysInfoSender()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs

		log.Println("Shutting down...")
		stopSysInfo <- true
		CleanUp()
		os.Exit(0)
	}()

	// Register handlers.
	d.Handle(gatt.PeripheralDiscovered(onPeripheralDiscovered))
	d.Init(onStateChanged)

	// run until os.Exit gets called in the signal handler
	select {}
}
