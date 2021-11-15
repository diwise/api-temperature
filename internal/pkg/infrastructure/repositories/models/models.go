package models

import (
	"time"

	"gorm.io/gorm"
)

//Temperature defines the structure for our temperatures table
type Temperature struct {
	gorm.Model
	Latitude   float64
	Longitude  float64
	Device     string
	Temp       float32
	Water      bool
	Timestamp  string
	Timestamp2 time.Time `gorm:"default:'1970-01-01T12:00:00Z'"`
}

//TemperatureV2 defines the structure for our new temperatures table
type TemperatureV2 struct {
	gorm.Model
	Latitude  float64
	Longitude float64
	Device    string `gorm:"index;index:device_at_time,unique"`
	Temp      float32
	Water     bool
	Timestamp time.Time `gorm:"index:device_at_time,unique"`
}
