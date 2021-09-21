package application

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/rs/zerolog"
	"github.com/streadway/amqp"

	"github.com/diwise/api-temperature/internal/pkg/infrastructure/repositories/database"
	"github.com/diwise/api-temperature/pkg/infrastructure/messaging/commands"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/messaging-golang/pkg/messaging/telemetry"
)

//MessagingContext is an interface that allows mocking of messaging.Context parameters
type MessagingContext interface {
	PublishOnTopic(message messaging.TopicMessage) error
	NoteToSelf(message messaging.CommandMessage) error
}

func NewStoreWaterTemperatureCommandHandler(db database.Datastore, messenger MessagingContext) messaging.CommandHandler {
	return func(wrapper messaging.CommandMessageWrapper, log zerolog.Logger) error {
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

func NewTemperatureReceiver(db database.Datastore) messaging.TopicMessageHandler {
	return func(msg amqp.Delivery, log zerolog.Logger) {

		log.Info().Str("body", string(msg.Body)).Msg("message received from queue")

		telTemp := &telemetry.Temperature{}
		err := json.Unmarshal(msg.Body, telTemp)

		if err != nil {
			log.Error().Err(err).Msg("failed to unmarshal message")
			return
		}

		if telTemp.Timestamp == "" {
			log.Warn().Msg("ignored temperature message with an empty timestamp")
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

func NewWaterTempReceiver(db database.Datastore) messaging.TopicMessageHandler {
	return func(msg amqp.Delivery, log zerolog.Logger) {

		log.Info().Str("body", string(msg.Body)).Msg("message received from queue")

		telTemp := &telemetry.WaterTemperature{}
		err := json.Unmarshal(msg.Body, telTemp)

		if err != nil {
			log.Error().Err(err).Msg("failed to unmarshal message")
			return
		}

		if telTemp.Timestamp == "" {
			log.Warn().Msg("ignored water temperature message with an empty timestamp.")
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
