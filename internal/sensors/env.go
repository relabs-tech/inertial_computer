package sensors

import (
	"fmt"
	"sync"
	"time"

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

// standbyTimeToDuration converts standby time config values to time.Duration
// Based on BMP280 datasheet standby times
func standbyTimeToDuration(val byte) time.Duration {
	switch val {
	case 0:
		return 500 * time.Microsecond // 0.5ms
	case 1:
		return 62500 * time.Microsecond // 62.5ms
	case 2:
		return 125 * time.Millisecond
	case 3:
		return 250 * time.Millisecond
	case 4:
		return 500 * time.Millisecond
	case 5:
		return 1000 * time.Millisecond
	case 6:
		return 2000 * time.Millisecond
	case 7:
		return 4000 * time.Millisecond
	default:
		return 0
	}
}

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

		leftOpts := bmxx80.Opts{
			Temperature: bmxx80.Oversampling(cfg.BMPLeftTempOSR),
			Pressure:    bmxx80.Oversampling(cfg.BMPLeftPressureOSR),
			Filter:      bmxx80.Filter(cfg.BMPLeftIIRFilter),
			Standby:     standbyTimeToDuration(cfg.BMPLeftStandbyTime),
		}

		bmpLeftDev, err = bmxx80.NewSPI(busLeft, &leftOpts)
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

		rightOpts := bmxx80.Opts{
			Temperature: bmxx80.Oversampling(cfg.BMPRightTempOSR),
			Pressure:    bmxx80.Oversampling(cfg.BMPRightPressureOSR),
			Filter:      bmxx80.Filter(cfg.BMPRightIIRFilter),
			Standby:     standbyTimeToDuration(cfg.BMPRightStandbyTime),
		}

		bmpRightDev, err = bmxx80.NewSPI(busRight, &rightOpts)
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
