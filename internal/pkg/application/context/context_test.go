package context_test

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/diwise/api-temperature/internal/pkg/application/context"
	"github.com/diwise/api-temperature/internal/pkg/infrastructure/repositories/database"
	"github.com/diwise/api-temperature/internal/pkg/infrastructure/repositories/models"
	ngsi "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld"
)

const inTheWater bool = true
const inTheAir bool = false

var db database.Datastore

func TestMain(m *testing.M) {

	// Create a reusable datastore with some default records in it. Reuse is OK until we start mutating.
	db = createMockedDB(
		createTempRecord(12.4, inTheAir, "2020-10-26T21:51:13Z"),
		createTempRecord(3.1, inTheWater, "2020-10-26T21:53:21Z"),
		createTempRecord(11.7, inTheAir, "2020-10-26T21:54:09Z"),
	)

	os.Exit(m.Run())
}

func TestThatCreateEntityFailsWithError(t *testing.T) {
	src := context.CreateSource(nil)
	if src.CreateEntity("", "", nil) == nil {
		t.Error("Unexpected success returned from CreateEntity")
	}
}

func TestGetWeatherObservedEntities(t *testing.T) {
	src := context.CreateSource(db)

	callbackCount := 0
	const callbackExpectation = 2
	callback := func(e ngsi.Entity) error {
		callbackCount++
		return nil
	}

	if err := src.GetEntities(newMockQueryForTypes([]string{"WeatherObserved"}), callback); err != nil {
		t.Error("Unexpected error when calling GetEntities. ", err.Error())
	}

	if callbackCount != callbackExpectation {
		t.Error("Unexpected number of callbacks made. ", callbackCount, " != ", callbackExpectation)
	}
}

func TestGetWaterQualityObservedEntities(t *testing.T) {
	src := context.CreateSource(db)

	callbackCount := 0
	const callbackExpectation = 1
	callback := func(e ngsi.Entity) error {
		callbackCount++
		return nil
	}

	if err := src.GetEntities(newMockQueryForTypes([]string{"WaterQualityObserved"}), callback); err != nil {
		t.Error("Unexpected error when calling GetEntities. ", err.Error())
	}

	if callbackCount != callbackExpectation {
		t.Error("Unexpected number of callbacks made. ", callbackCount, " != ", callbackExpectation)
	}
}

func TestGetBothTypesOfEntities(t *testing.T) {
	src := context.CreateSource(db)

	callbackCount := 0
	const callbackExpectation = 3
	callback := func(e ngsi.Entity) error {
		callbackCount++
		return nil
	}

	if err := src.GetEntities(newMockQueryForTypes([]string{"WeatherObserved", "WaterQualityObserved"}), callback); err != nil {
		t.Error("Unexpected error when calling GetEntities. ", err.Error())
	}

	if callbackCount != callbackExpectation {
		t.Error("Unexpected number of callbacks made. ", callbackCount, " != ", callbackExpectation)
	}
}

func TestGetEntitiesFromAttributeList(t *testing.T) {
	src := context.CreateSource(db)

	callbackCount := 0
	const callbackExpectation = 3
	callback := func(e ngsi.Entity) error {
		callbackCount++
		return nil
	}

	if err := src.GetEntities(newMockQueryForAttributes([]string{"temperature"}), callback); err != nil {
		t.Error("Unexpected error when calling GetEntities. ", err.Error())
	}

	if callbackCount != callbackExpectation {
		t.Error("Unexpected number of callbacks made. ", callbackCount, " != ", callbackExpectation)
	}
}

func TestGetEntitiesWithDeviceQuery(t *testing.T) {
	src := context.CreateSource(db)

	callbackCount := 0
	const callbackExpectation = 2
	callback := func(e ngsi.Entity) error {
		callbackCount++
		return nil
	}

	if err := src.GetEntities(newMockQueryForDevice("deviceID", []string{"WeatherObserved"}), callback); err != nil {
		t.Error("Unexpected error when calling GetEntities. ", err.Error())
	}

	if callbackCount != callbackExpectation {
		t.Error("Unexpected number of callbacks made. ", callbackCount, " != ", callbackExpectation)
	}
}

func TestGetEntitiesOfUnknownTypeReturnsError(t *testing.T) {
	src := context.CreateSource(nil)
	if src.GetEntities(newMockQueryForTypes([]string{"UnknownType"}), nil) == nil {
		t.Error("Error")
	}
}

type mockDB struct {
	temps []models.TemperatureV2
}

func createMockedDB(records ...models.TemperatureV2) database.Datastore {
	db := &mockDB{}
	db.temps = append(db.temps, records...)
	return db
}

func (db *mockDB) AddTemperatureMeasurement(device *string, latitude, longitude, temp float64, water bool, when string) (*models.TemperatureV2, error) {
	return nil, nil
}

func (db *mockDB) GetTemperatures(deviceId string, from, to time.Time, geoSpatial string, lon0, lat0, lon1, lat1 float64, resultOffset, resultLimit uint64) ([]models.TemperatureV2, error) {
	return db.temps, nil
}

type mockQuery struct {
	device string
	attrs  []string
	types  []string
}

func newMockQueryForAttributes(attributeNames []string) mockQuery {
	return mockQuery{attrs: attributeNames}
}

func newMockQueryForTypes(typeNames []string) mockQuery {
	return mockQuery{types: typeNames}
}

func newMockQueryForDevice(deviceName string, typeNames []string) mockQuery {
	return mockQuery{device: deviceName, types: typeNames}
}

func (q mockQuery) Device() string {
	return q.device
}

func (q mockQuery) EntityAttributes() []string {
	return q.attrs
}

func (q mockQuery) EntityTypes() []string {
	return q.types
}

func (q mockQuery) Geo() ngsi.GeoQuery {
	return ngsi.GeoQuery{}
}

func (q mockQuery) IsGeoQuery() bool {
	return false
}

func (q mockQuery) Temporal() ngsi.TemporalQuery {
	return ngsi.TemporalQuery{}
}

func (q mockQuery) IsTemporalQuery() bool {
	return false
}

func (q mockQuery) HasDeviceReference() bool {
	return len(q.device) > 0
}

func (q mockQuery) PaginationLimit() uint64 {
	return 0
}

func (q mockQuery) PaginationOffset() uint64 {
	return 0
}

func (q mockQuery) Request() *http.Request {
	return nil
}

func createTempRecord(temp float32, water bool, when string) models.TemperatureV2 {
	t := models.TemperatureV2{}
	t.Temp = temp
	t.Water = water
	t.Timestamp, _ = time.Parse(time.RFC3339Nano, when)
	return t
}
