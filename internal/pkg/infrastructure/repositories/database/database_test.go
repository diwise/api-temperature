package database_test

import (
	"os"
	"testing"
	"time"

	"github.com/diwise/api-temperature/internal/pkg/infrastructure/logging"
	"github.com/diwise/api-temperature/internal/pkg/infrastructure/repositories/database"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestSomething(t *testing.T) {
	log := logging.NewLogger()
	db, _ := database.NewDatabaseConnection(log, database.NewSQLiteConnector())

	deviceName := "mydevice"
	db.AddTemperatureMeasurement(&deviceName, 64.278, 17.182, 12.7, true, time.Now().Format(time.RFC3339))

	temps, _ := db.GetTemperaturesNearPoint(62.389517, 17.306133, 1000, 5)
	if len(temps) != 0 {
		t.Errorf("number of returned temperatures differ from expectation. %d != %d", len(temps), 0)
	}
}
