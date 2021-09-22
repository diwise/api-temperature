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

func TestThatGettingTemperaturesByTimespanWorks(t *testing.T) {
	log := logging.NewLogger()
	db, _ := database.NewDatabaseConnection(log, database.NewSQLiteConnector())

	time1 := time.Now().UTC()
	time2 := time.Now().UTC().Add(2 * time.Hour)
	time3 := time.Now().UTC().Add(3 * time.Hour)

	deviceName := "mydevice"
	db.AddTemperatureMeasurement(&deviceName, 64.278, 17.182, 12.7, true, time2.Format(time.RFC3339))

	temps, _ := db.GetTemperaturesWithinTimespan(time1, time3, 1)
	if len(temps) != 1 {
		t.Errorf("number of returned temperatures differ from expectation. %d != %d", len(temps), 1)
	}
}

func TestGettingTemperaturesNearPointAtTimeWorks(t *testing.T) {
	log := logging.NewLogger()
	db, _ := database.NewDatabaseConnection(log, database.NewSQLiteConnector())

	from := time.Now().UTC()
	time2 := time.Now().UTC().Add(2 * time.Hour)
	to := time.Now().UTC().Add(2 * time.Hour)

	lat := 64.278
	lon := 17.182

	deviceName := "mydevice2"
	db.AddTemperatureMeasurement(&deviceName, lat, lon, 12.7, false, time2.Format(time.RFC3339))

	temps, _ := db.GetTemperaturesNearPointAtTime(lat, lon, 1, 1, from, to)
	if len(temps) != 1 {
		t.Errorf("number of returned temperatures differ from expectation. %d != %d", len(temps), 1)
	}
}

func TestGettingTemperaturesWithinRectangleAtTimeWorks(t *testing.T) {
	log := logging.NewLogger()
	db, _ := database.NewDatabaseConnection(log, database.NewSQLiteConnector())

	from := time.Now().UTC()
	time2 := time.Now().UTC().Add(2 * time.Hour)
	to := time.Now().UTC().Add(3 * time.Hour)

	lat0 := 62.278
	lon0 := 17.182
	lat1 := 62.383
	lon1 := 17.382
	lat2 := 62.4354
	lon2 := 17.4748

	deviceName := "mydevice3"
	db.AddTemperatureMeasurement(&deviceName, lat1, lon1, 12.7, false, time2.Format(time.RFC3339))

	temps, _ := db.GetTemperaturesWithinRectangleAtTime(lat2, lon0, lat0, lon2, 1, from, to)
	if len(temps) != 1 {
		t.Errorf("number of returned temperatures differ from expectation. %d != %d", len(temps), 1)
	}
}
