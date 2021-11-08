package database_test

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/diwise/api-temperature/internal/pkg/infrastructure/repositories/database"
	"github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestThatGetTemperaturesWorksWithDeviceIDAndTimeSpan(t *testing.T) {
	log := log.Logger
	db, _ := database.NewDatabaseConnection(database.NewSQLiteConnector(log))

	time1 := time.Now().UTC()
	time2 := time.Now().UTC().Add(2 * time.Hour)
	time3 := time.Now().UTC().Add(3 * time.Hour)

	deviceName := "mydevice"
	db.AddTemperatureMeasurement(&deviceName, 64.278, 17.182, 12.7, true, time2.Format(time.RFC3339))

	temps, _ := db.GetTemperatures(deviceName, time1, time3, "", 0.0, 0.0, 0.0, 0.0, 1)
	if len(temps) != 1 {
		t.Errorf("number of returned temperatures differ from expectation. %d != %d", len(temps), 1)
	}
}

func TestThatGetTemperaturesWorksWithTimeSpanAndNearPoint(t *testing.T) {
	log := log.Logger
	db, _ := database.NewDatabaseConnection(database.NewSQLiteConnector(log))

	time1 := time.Now().UTC()
	time2 := time.Now().UTC().Add(2 * time.Hour)
	time3 := time.Now().UTC().Add(3 * time.Hour)

	deviceName := "mydevice"
	db.AddTemperatureMeasurement(&deviceName, 64.278, 17.182, 12.7, true, time2.Format(time.RFC3339))

	geoQ := ngsi.GeoQuery{
		Coordinates: []float64{64.2775, 17.1815},
		GeoRel:      "near",
	}

	nw_lat, nw_lon, se_lat, se_lon := getApproximatePoint(geoQ.Coordinates[0], geoQ.Coordinates[1], 1000)

	temps, _ := db.GetTemperatures("", time1, time3, geoQ.GeoRel, nw_lat, nw_lon, se_lat, se_lon, 1)
	if len(temps) != 1 {
		t.Errorf("number of returned temperatures differ from expectation. %d != %d", len(temps), 1)
	}
}

func TestThatGetTemperaturesWorksWithTimeSpanAndWithinRectangle(t *testing.T) {
	log := log.Logger
	db, _ := database.NewDatabaseConnection(database.NewSQLiteConnector(log))

	time1 := time.Now().UTC()
	time2 := time.Now().UTC().Add(2 * time.Hour)
	time3 := time.Now().UTC().Add(3 * time.Hour)

	deviceName := "mydevice"
	db.AddTemperatureMeasurement(&deviceName, 63.278, 17.185, 12.7, true, time2.Format(time.RFC3339))

	geoQ := ngsi.GeoQuery{
		Coordinates: []float64{64.2775, 17.1815, 62.4354, 17.4748},
		GeoRel:      "within",
	}

	nw_lat, nw_lon := geoQ.Coordinates[0], geoQ.Coordinates[1]
	se_lat, se_lon := geoQ.Coordinates[2], geoQ.Coordinates[3]

	temps, _ := db.GetTemperatures("", time1, time3, geoQ.GeoRel, nw_lat, nw_lon, se_lat, se_lon, 1)
	if len(temps) != 1 {
		t.Errorf("number of returned temperatures differ from expectation. %d != %d", len(temps), 1)
	}
}

func getApproximatePoint(latitude, longitude float64, distance uint64) (nwLat, neLon, seLat, seLon float64) {
	// Make a crude estimation of the coordinate offset based on the distance
	d := float64(distance)
	lat_delta := (180.0 / math.Pi) * (d / 6378137.0)
	lon_delta := (180.0 / math.Pi) * (d / 6378137.0) / math.Cos(math.Pi/180.0*latitude)

	nw_lat := latitude + lat_delta
	nw_lon := longitude - lon_delta
	se_lat := latitude - lat_delta
	se_lon := longitude + lon_delta

	// TODO: This is not correct, but a good enough first approximation for the MVP. We should make use of PostGIS
	// and do a correct search for matches within a radius. Not within a "square" like this.
	return nw_lat, nw_lon, se_lat, se_lon
}
