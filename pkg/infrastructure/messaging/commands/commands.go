package commands

import (
	"github.com/diwise/messaging-golang/pkg/messaging/telemetry"
)

const (
	//StoreTemperatureUpdateType is the content type for ...
	StoreTemperatureUpdateType = "application/vnd-diwise-storetemperatureupdate+json"
	//StoreWaterTemperatureUpdateType is the content type for ...
	StoreWaterTemperatureUpdateType = "application/vnd-diwise-storewatertemperatureupdate+json"
)

//StoreTemperatureUpdate is a command that takes info about a temperature update and enqueues it for persistence
type StoreTemperatureUpdate struct {
	telemetry.Temperature
}

//ContentType returns the content type that this event will be sent as
func (stu *StoreTemperatureUpdate) ContentType() string {
	return StoreTemperatureUpdateType
}

//StoreTemperatureUpdate is a command that takes info about a temperature update and enqueues it for persistence
type StoreWaterTemperatureUpdate struct {
	telemetry.WaterTemperature
}

//ContentType returns the content type that this event will be sent as
func (stu *StoreWaterTemperatureUpdate) ContentType() string {
	return StoreWaterTemperatureUpdateType
}
