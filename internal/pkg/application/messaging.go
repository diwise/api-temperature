package application

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/streadway/amqp"

	"github.com/iot-for-tillgenglighet/api-temperature/internal/pkg/infrastructure/logging"
	"github.com/iot-for-tillgenglighet/api-temperature/internal/pkg/infrastructure/repositories/database"
	"github.com/iot-for-tillgenglighet/api-temperature/pkg/infrastructure/messaging/commands"
	"github.com/iot-for-tillgenglighet/messaging-golang/pkg/messaging"
	"github.com/iot-for-tillgenglighet/messaging-golang/pkg/messaging/telemetry"
)

//MessagingContext is an interface that allows mocking of messaging.Context parameters
type MessagingContext interface {
	PublishOnTopic(message messaging.TopicMessage) error
	NoteToSelf(message messaging.CommandMessage) error
}

func NewStoreWaterTemperatureCommandHandler(db database.Datastore, messenger MessagingContext) messaging.CommandHandler {
	return func(wrapper messaging.CommandMessageWrapper) error {
		cmd := &commands.StoreWaterTemperatureUpdate{}
		err := json.Unmarshal(wrapper.Body(), cmd)
		if err != nil {
			return fmt.Errorf("failed to unmarshal command! %s", err.Error())
		}

		_, err = db.AddTemperatureMeasurement(
			&cmd.Origin.Device,
			cmd.Origin.Latitude, cmd.Origin.Longitude,
			float64(math.Round(cmd.Temp*10)/10),
			true,
			cmd.Timestamp,
		)

		return err
	}
}

func NewTemperatureReceiver(log logging.Logger, db database.Datastore) messaging.TopicMessageHandler {
	return func(msg amqp.Delivery) {

		log.Infof("Message received from queue: %s", string(msg.Body))

		telTemp := &telemetry.Temperature{}
		err := json.Unmarshal(msg.Body, telTemp)

		if err != nil {
			log.Error("Failed to unmarshal message")
			return
		}

		if telTemp.Timestamp == "" {
			log.Info("Ignored temperature message with an empty timestamp.")
			return
		}

		db.AddTemperatureMeasurement(
			&telTemp.Origin.Device,
			telTemp.Origin.Latitude, telTemp.Origin.Longitude,
			float64(math.Round(telTemp.Temp*10)/10),
			false,
			telTemp.Timestamp,
		)
	}
}

func NewWaterTempReceiver(log logging.Logger, db database.Datastore) messaging.TopicMessageHandler {
	return func(msg amqp.Delivery) {

		log.Infof("Message received from queue: %s", string(msg.Body))

		telTemp := &telemetry.WaterTemperature{}
		err := json.Unmarshal(msg.Body, telTemp)

		if err != nil {
			log.Error("Failed to unmarshal message")
			return
		}

		if telTemp.Timestamp == "" {
			log.Info("Ignored water temperature message with an empty timestamp.")
			return
		}

		db.AddTemperatureMeasurement(
			&telTemp.Origin.Device,
			telTemp.Origin.Latitude, telTemp.Origin.Longitude,
			float64(math.Round(telTemp.Temp*10)/10),
			true,
			telTemp.Timestamp,
		)
	}
}
