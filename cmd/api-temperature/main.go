package main

import (
	"strings"

	"github.com/diwise/api-temperature/internal/pkg/application"
	"github.com/diwise/api-temperature/internal/pkg/infrastructure/repositories/database"
	"github.com/rs/zerolog/log"

	"github.com/diwise/api-temperature/pkg/infrastructure/messaging/commands"

	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/messaging-golang/pkg/messaging/telemetry"
)

func main() {

	serviceName := "api-temperature"

	logger := log.With().Str("service", strings.ToLower(serviceName)).Logger()
	logger.Info().Msg("starting up ...")

	config := messaging.LoadConfiguration(serviceName, logger)
	messenger, _ := messaging.Initialize(config)

	defer messenger.Close()

	// Make sure that we have a proper connection to the database ...
	db, _ := database.NewDatabaseConnection(database.NewPostgreSQLConnector(logger))

	// ... before we start listening for temperature telemetry
	messenger.RegisterTopicMessageHandler(
		(&telemetry.Temperature{}).TopicName(),
		application.NewTemperatureReceiver(db),
	)
	messenger.RegisterTopicMessageHandler(
		(&telemetry.WaterTemperature{}).TopicName(),
		application.NewWaterTempReceiver(db),
	)

	messenger.RegisterCommandHandler(
		commands.StoreTemperatureUpdateType,
		application.NewStoreTemperatureCommandHandler(db, messenger),
	)

	messenger.RegisterCommandHandler(
		commands.StoreWaterTemperatureUpdateType,
		application.NewStoreWaterTemperatureCommandHandler(db, messenger),
	)

	application.CreateRouterAndStartServing(logger, db)
}
