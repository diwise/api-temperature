package main

import (
	log "github.com/sirupsen/logrus"

	_ "github.com/jinzhu/gorm/dialects/postgres"

	"github.com/iot-for-tillgenglighet/api-temperature/pkg/database"
	"github.com/iot-for-tillgenglighet/api-temperature/pkg/handler"
	"github.com/iot-for-tillgenglighet/messaging-golang/pkg/messaging"
	"github.com/iot-for-tillgenglighet/messaging-golang/pkg/messaging/telemetry"
)

func main() {

	serviceName := "api-temperature"

	log.Infof("Starting up %s ...", serviceName)

	config := messaging.LoadConfiguration(serviceName)
	messenger, _ := messaging.Initialize(config)

	defer messenger.Close()

	// Make sure that we have a proper connection to the database ...
	database.ConnectToDB()

	// ... before we start listening for temperature telemetry
	messenger.RegisterTopicMessageHandler(
		(&telemetry.Temperature{}).TopicName(),
		receiveTemperature,
	)

	handler.Router()
}
