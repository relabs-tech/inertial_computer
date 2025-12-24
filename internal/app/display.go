package app

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/devices/v3/ssd1306"
	"periph.io/x/devices/v3/ssd1306/image1bit"
	"periph.io/x/host/v3"

	"github.com/relabs-tech/inertial_computer/internal/config"
	"github.com/relabs-tech/inertial_computer/internal/gps"
	"github.com/relabs-tech/inertial_computer/internal/imu"
	"github.com/relabs-tech/inertial_computer/internal/orientation"
)

// DisplayData holds the latest data for display
type DisplayData struct {
	mu sync.RWMutex

	// IMU raw data
	imuRawLeft      imu.IMURaw
	haveIMURawLeft  bool
	imuRawRight     imu.IMURaw
	haveIMURawRight bool

	// Orientation data
	poseLeft      orientation.Pose
	havePoseLeft  bool
	poseRight     orientation.Pose
	havePoseRight bool

	// GPS data
	gpsPos  gps.Position
	haveGPS bool
}

func RunDisplay() error {
	cfg := config.Get()

	// Initialize periph
	if _, err := host.Init(); err != nil {
		return fmt.Errorf("failed to initialize periph: %w", err)
	}

	// Open I2C bus
	bus, err := i2creg.Open("")
	if err != nil {
		return fmt.Errorf("failed to open I2C bus: %w", err)
	}
	defer bus.Close()

	// Initialize left display
	leftDisplay, err := ssd1306.NewI2C(bus, cfg.DisplayLeftI2CAddr, &ssd1306.DefaultOpts)
	if err != nil {
		return fmt.Errorf("failed to initialize left display: %w", err)
	}
	log.Printf("display: left display initialized at 0x%02X", cfg.DisplayLeftI2CAddr)

	// Initialize right display
	rightDisplay, err := ssd1306.NewI2C(bus, cfg.DisplayRightI2CAddr, &ssd1306.DefaultOpts)
	if err != nil {
		return fmt.Errorf("failed to initialize right display: %w", err)
	}
	log.Printf("display: right display initialized at 0x%02X", cfg.DisplayRightI2CAddr)

	// Show splash screens
	if err := showLeftSplash(leftDisplay); err != nil {
		log.Printf("display: error showing left splash: %v", err)
	}
	if err := showRightSplash(rightDisplay); err != nil {
		log.Printf("display: error showing right splash: %v", err)
	}

	// Data storage
	data := &DisplayData{}

	// Connect to MQTT
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.MQTTBroker).
		SetClientID(cfg.MQTTClientIDDisplay)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	log.Printf("display: connected to MQTT broker at %s", cfg.MQTTBroker)

	// Subscribe to topics based on display content configuration
	if err := subscribeForContent(client, cfg.DisplayLeftContent, data, cfg); err != nil {
		return fmt.Errorf("failed to subscribe for left display: %w", err)
	}
	if err := subscribeForContent(client, cfg.DisplayRightContent, data, cfg); err != nil {
		return fmt.Errorf("failed to subscribe for right display: %w", err)
	}

	// Display update loop
	ticker := time.NewTicker(time.Duration(cfg.DisplayUpdateInterval) * time.Millisecond)
	defer ticker.Stop()

	log.Println("display: starting update loop")

	for range ticker.C {
		// Read data without copying the mutex
		data.mu.RLock()
		snapshot := DisplayData{
			imuRawLeft:      data.imuRawLeft,
			haveIMURawLeft:  data.haveIMURawLeft,
			imuRawRight:     data.imuRawRight,
			haveIMURawRight: data.haveIMURawRight,
			poseLeft:        data.poseLeft,
			havePoseLeft:    data.havePoseLeft,
			poseRight:       data.poseRight,
			havePoseRight:   data.havePoseRight,
			gpsPos:          data.gpsPos,
			haveGPS:         data.haveGPS,
		}
		data.mu.RUnlock()

		// Update left display
		if err := updateDisplay(leftDisplay, cfg.DisplayLeftContent, &snapshot); err != nil {
			log.Printf("display: error updating left display: %v", err)
		}

		// Update right display
		if err := updateDisplay(rightDisplay, cfg.DisplayRightContent, &snapshot); err != nil {
			log.Printf("display: error updating right display: %v", err)
		}
	}

	return nil
}

func subscribeForContent(client mqtt.Client, content string, data *DisplayData, cfg *config.Config) error {
	switch content {
	case "imu_raw_left":
		token := client.Subscribe(cfg.TopicIMULeft, 0, func(_ mqtt.Client, msg mqtt.Message) {
			var raw imu.IMURaw
			if err := json.Unmarshal(msg.Payload(), &raw); err != nil {
				log.Printf("display: imu_raw_left unmarshal error: %v", err)
				return
			}
			data.mu.Lock()
			data.imuRawLeft = raw
			data.haveIMURawLeft = true
			data.mu.Unlock()
		})
		token.Wait()
		if token.Error() != nil {
			return token.Error()
		}
		log.Printf("display: subscribed to %s", cfg.TopicIMULeft)

	case "imu_raw_right":
		token := client.Subscribe(cfg.TopicIMURight, 0, func(_ mqtt.Client, msg mqtt.Message) {
			var raw imu.IMURaw
			if err := json.Unmarshal(msg.Payload(), &raw); err != nil {
				log.Printf("display: imu_raw_right unmarshal error: %v", err)
				return
			}
			data.mu.Lock()
			data.imuRawRight = raw
			data.haveIMURawRight = true
			data.mu.Unlock()
		})
		token.Wait()
		if token.Error() != nil {
			return token.Error()
		}
		log.Printf("display: subscribed to %s", cfg.TopicIMURight)

	case "orientation_left":
		token := client.Subscribe(cfg.TopicPoseLeft, 0, func(_ mqtt.Client, msg mqtt.Message) {
			var p orientation.Pose
			if err := json.Unmarshal(msg.Payload(), &p); err != nil {
				log.Printf("display: orientation_left unmarshal error: %v", err)
				return
			}
			data.mu.Lock()
			data.poseLeft = p
			data.havePoseLeft = true
			data.mu.Unlock()
		})
		token.Wait()
		if token.Error() != nil {
			return token.Error()
		}
		log.Printf("display: subscribed to %s", cfg.TopicPoseLeft)

	case "orientation_right":
		token := client.Subscribe(cfg.TopicPoseRight, 0, func(_ mqtt.Client, msg mqtt.Message) {
			var p orientation.Pose
			if err := json.Unmarshal(msg.Payload(), &p); err != nil {
				log.Printf("display: orientation_right unmarshal error: %v", err)
				return
			}
			data.mu.Lock()
			data.poseRight = p
			data.havePoseRight = true
			data.mu.Unlock()
		})
		token.Wait()
		if token.Error() != nil {
			return token.Error()
		}
		log.Printf("display: subscribed to %s", cfg.TopicPoseRight)

	case "gps":
		token := client.Subscribe(cfg.TopicGPSPosition, 0, func(_ mqtt.Client, msg mqtt.Message) {
			var pos gps.Position
			if err := json.Unmarshal(msg.Payload(), &pos); err != nil {
				log.Printf("display: gps unmarshal error: %v", err)
				return
			}
			data.mu.Lock()
			data.gpsPos = pos
			data.haveGPS = true
			data.mu.Unlock()
		})
		token.Wait()
		if token.Error() != nil {
			return token.Error()
		}
		log.Printf("display: subscribed to %s", cfg.TopicGPSPosition)

	default:
		return fmt.Errorf("unknown display content type: %s", content)
	}

	return nil
}

func updateDisplay(dev *ssd1306.Dev, content string, data *DisplayData) error {
	switch content {
	case "imu_raw_left":
		return updateIMURawDisplay(dev, data.imuRawLeft, data.haveIMURawLeft, "Left")
	case "imu_raw_right":
		return updateIMURawDisplay(dev, data.imuRawRight, data.haveIMURawRight, "Right")
	case "orientation_left":
		return updateOrientationDisplay(dev, data.poseLeft, data.havePoseLeft)
	case "orientation_right":
		return updateOrientationDisplay(dev, data.poseRight, data.havePoseRight)
	case "gps":
		return updateGPSDisplay(dev, data.gpsPos, data.haveGPS)
	default:
		return fmt.Errorf("unknown display content type: %s", content)
	}
}

func updateIMURawDisplay(dev *ssd1306.Dev, raw imu.IMURaw, haveData bool, label string) error {
	img := image1bit.NewVerticalLSB(image.Rect(0, 0, 128, 64))

	// Blank image
	for i := 0; i < 1024; i++ {
		img.Pix[i] = 0
	}

	drawer := &font.Drawer{
		Dst:  img,
		Src:  &image.Uniform{image1bit.On},
		Face: basicfont.Face7x13,
	}

	if !haveData {
		drawer.Dot = fixed.P(0, 26)
		drawer.DrawBytes([]byte("IMU " + label))
		drawer.Dot = fixed.P(0, 39)
		drawer.DrawBytes([]byte("Waiting..."))
	} else {
		// Accel
		drawer.Dot = fixed.P(0, 13)
		drawer.DrawBytes([]byte(fmt.Sprintf("A:%5d %5d", raw.Ax, raw.Ay)))

		drawer.Dot = fixed.P(0, 26)
		drawer.DrawBytes([]byte(fmt.Sprintf("  %5d", raw.Az)))

		// Gyro
		drawer.Dot = fixed.P(0, 39)
		drawer.DrawBytes([]byte(fmt.Sprintf("G:%5d %5d", raw.Gx, raw.Gy)))

		drawer.Dot = fixed.P(0, 52)
		drawer.DrawBytes([]byte(fmt.Sprintf("  %5d", raw.Gz)))
	}

	return dev.Draw(dev.Bounds(), img, image.Point{})
}

func updateOrientationDisplay(dev *ssd1306.Dev, pose orientation.Pose, haveData bool) error {
	img := image1bit.NewVerticalLSB(image.Rect(0, 0, 128, 64))

	// Blank image
	for i := 0; i < 1024; i++ {
		img.Pix[i] = 0
	}

	drawer := &font.Drawer{
		Dst:  img,
		Src:  &image.Uniform{image1bit.On},
		Face: basicfont.Face7x13,
	}

	if !haveData {
		drawer.Dot = fixed.P(0, 26)
		drawer.DrawBytes([]byte("Orientation"))
		drawer.Dot = fixed.P(0, 39)
		drawer.DrawBytes([]byte("Waiting..."))
	} else {
		// Roll
		drawer.Dot = fixed.P(0, 13)
		drawer.DrawBytes([]byte(fmt.Sprintf("R: %6.1f", pose.Roll)))

		// Pitch
		drawer.Dot = fixed.P(0, 26)
		drawer.DrawBytes([]byte(fmt.Sprintf("P: %6.1f", pose.Pitch)))

		// Yaw
		drawer.Dot = fixed.P(0, 39)
		drawer.DrawBytes([]byte(fmt.Sprintf("Y: %6.1f", pose.Yaw)))
	}

	return dev.Draw(dev.Bounds(), img, image.Point{})
}

func updateGPSDisplay(dev *ssd1306.Dev, pos gps.Position, haveData bool) error {
	img := image1bit.NewVerticalLSB(image.Rect(0, 0, 128, 64))

	// Blank image
	for i := 0; i < 1024; i++ {
		img.Pix[i] = 0
	}

	drawer := &font.Drawer{
		Dst:  img,
		Src:  &image.Uniform{image1bit.On},
		Face: basicfont.Face7x13,
	}

	if !haveData {
		drawer.Dot = fixed.P(0, 26)
		drawer.DrawBytes([]byte("GPS Position"))
		drawer.Dot = fixed.P(0, 39)
		drawer.DrawBytes([]byte("Waiting..."))
	} else {
		// Latitude
		drawer.Dot = fixed.P(0, 13)
		latDir := "N"
		lat := pos.Latitude
		if lat < 0 {
			latDir = "S"
			lat = -lat
		}
		drawer.DrawBytes([]byte(fmt.Sprintf("%.4f%s", lat, latDir)))

		// Longitude
		drawer.Dot = fixed.P(0, 26)
		lonDir := "E"
		lon := pos.Longitude
		if lon < 0 {
			lonDir = "W"
			lon = -lon
		}
		drawer.DrawBytes([]byte(fmt.Sprintf("%.4f%s", lon, lonDir)))

		// Altitude
		drawer.Dot = fixed.P(0, 39)
		drawer.DrawBytes([]byte(fmt.Sprintf("Alt: %.0fm", pos.Altitude)))
	}

	return dev.Draw(dev.Bounds(), img, image.Point{})
}

func showLeftSplash(dev *ssd1306.Dev) error {
	img := image1bit.NewVerticalLSB(image.Rect(0, 0, 128, 64))

	// Blank image
	for i := 0; i < 1024; i++ {
		img.Pix[i] = 0
	}

	drawer := &font.Drawer{
		Dst:  img,
		Src:  &image.Uniform{image1bit.On},
		Face: basicfont.Face7x13,
	}

	drawer.Dot = fixed.P(10, 26)
	drawer.DrawBytes([]byte("Inertial Pi"))

	drawer.Dot = fixed.P(5, 43)
	drawer.DrawBytes([]byte("Looking for"))

	drawer.Dot = fixed.P(25, 56)
	drawer.DrawBytes([]byte("sats"))

	return dev.Draw(dev.Bounds(), img, image.Point{})
}

func showRightSplash(dev *ssd1306.Dev) error {
	img := image1bit.NewVerticalLSB(image.Rect(0, 0, 128, 64))

	// Blank image
	for i := 0; i < 1024; i++ {
		img.Pix[i] = 0
	}

	drawer := &font.Drawer{
		Dst:  img,
		Src:  &image.Uniform{image1bit.On},
		Face: basicfont.Face7x13,
	}

	drawer.Dot = fixed.P(5, 26)
	drawer.DrawBytes([]byte("Daniel Alarcon"))

	drawer.Dot = fixed.P(10, 43)
	drawer.DrawBytes([]byte("Strapdown"))

	drawer.Dot = fixed.P(25, 56)
	drawer.DrawBytes([]byte("Tests"))

	return dev.Draw(dev.Bounds(), img, image.Point{})
}
