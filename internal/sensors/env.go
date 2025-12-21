package sensors

import (
	"fmt"
	"sync"

	"github.com/relabs-tech/inertial_computer/internal/config"
	"github.com/relabs-tech/inertial_computer/internal/env"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/devices/v3/bmxx80"
	"periph.io/x/host/v3"
)

var (
	bmpLeftDev  *bmxx80.Dev
	bmpRightDev *bmxx80.Dev
	bmpOnce     sync.Once
	bmpInitErr  error
)

// initBMP initializes both BMP sensors once
func initBMP() {
	bmpOnce.Do(func() {
		cfg := config.Get()

		// Initialize periph host
		if _, err := host.Init(); err != nil {
			bmpInitErr = fmt.Errorf("periph host init: %w", err)
			return
		}

		// Initialize left BMP
		busLeft, err := spireg.Open(cfg.BMPLeftSPIDevice)
		if err != nil {
			bmpInitErr = fmt.Errorf("left BMP SPI open: %w", err)
			return
		}

		bmpLeftDev, err = bmxx80.NewSPI(busLeft, &bmxx80.DefaultOpts)
		if err != nil {
			bmpInitErr = fmt.Errorf("left BMP init: %w", err)
			return
		}

		// Initialize right BMP
		busRight, err := spireg.Open(cfg.BMPRightSPIDevice)
		if err != nil {
			bmpInitErr = fmt.Errorf("right BMP SPI open: %w", err)
			return
		}

		bmpRightDev, err = bmxx80.NewSPI(busRight, &bmxx80.DefaultOpts)
		if err != nil {
			bmpInitErr = fmt.Errorf("right BMP init: %w", err)
			return
		}

		fmt.Println("BMP sensors initialized successfully")
	})
}

// ReadLeftEnv reads the LEFT BMP sensor (temp + pressure).
func ReadLeftEnv() (env.Sample, error) {
	initBMP()
	if bmpInitErr != nil {
		return env.Sample{}, bmpInitErr
	}

	var e physic.Env
	if err := bmpLeftDev.Sense(&e); err != nil {
		return env.Sample{}, fmt.Errorf("left BMP sense: %w", err)
	}

	pressurePa := float64(e.Pressure) / float64(physic.Pascal)
	return env.Sample{
		Source:       "left",
		Temperature:  e.Temperature.Celsius(),
		Pressure:     pressurePa,
		PressureMbar: pressurePa / 100.0, // 1 mbar = 100 Pa
		PressureHPa:  pressurePa / 100.0, // 1 hPa = 100 Pa (same as mbar)
	}, nil
}

// ReadRightEnv reads the RIGHT BMP sensor (temp + pressure).
func ReadRightEnv() (env.Sample, error) {
	initBMP()
	if bmpInitErr != nil {
		return env.Sample{}, bmpInitErr
	}

	var e physic.Env
	if err := bmpRightDev.Sense(&e); err != nil {
		return env.Sample{}, fmt.Errorf("right BMP sense: %w", err)
	}

	pressurePa := float64(e.Pressure) / float64(physic.Pascal)
	return env.Sample{
		Source:       "right",
		Temperature:  e.Temperature.Celsius(),
		Pressure:     pressurePa,
		PressureMbar: pressurePa / 100.0, // 1 mbar = 100 Pa
		PressureHPa:  pressurePa / 100.0, // 1 hPa = 100 Pa (same as mbar)
	}, nil
}
