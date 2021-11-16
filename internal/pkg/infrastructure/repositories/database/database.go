package database

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/diwise/api-temperature/internal/pkg/infrastructure/repositories/models"
	"github.com/rs/zerolog"
)

//Datastore is an interface that is used to inject the database into different handlers to improve testability
type Datastore interface {
	AddTemperatureMeasurement(device *string, latitude, longitude, temp float64, water bool, when string) (*models.TemperatureV2, error)
	GetTemperatures(deviceId string, from, to time.Time, geoSpatial string, lat0, lon0, lat1, lon1 float64, resultLimit uint64) ([]models.TemperatureV2, error)
}

var dbCtxKey = &databaseContextKey{"database"}

type databaseContextKey struct {
	name string
}

// Middleware packs a pointer to the datastore into context
func Middleware(db Datastore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), dbCtxKey, db)

			// and call the next with our new context
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

//GetFromContext extracts the database wrapper, if any, from the provided context
func GetFromContext(ctx context.Context) (Datastore, error) {
	db, ok := ctx.Value(dbCtxKey).(Datastore)
	if ok {
		return db, nil
	}

	return nil, errors.New("failed to decode database from context")
}

type myDB struct {
	impl *gorm.DB
	log  zerolog.Logger
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

//ConnectorFunc is used to inject a database connection method into NewDatabaseConnection
type ConnectorFunc func() (*gorm.DB, zerolog.Logger, error)

//NewPostgreSQLConnector opens a connection to a postgresql database
func NewPostgreSQLConnector(log zerolog.Logger) ConnectorFunc {
	dbHost := os.Getenv("TEMPERATURE_DB_HOST")
	username := os.Getenv("TEMPERATURE_DB_USER")
	dbName := os.Getenv("TEMPERATURE_DB_NAME")
	password := os.Getenv("TEMPERATURE_DB_PASSWORD")
	sslMode := getEnv("TEMPERATURE_DB_SSLMODE", "require")

	dbURI := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=%s password=%s", dbHost, username, dbName, sslMode, password)

	return func() (*gorm.DB, zerolog.Logger, error) {
		sublogger := log.With().Str("host", dbHost).Str("database", dbName).Logger()

		for {
			sublogger.Info().Msg("connecting to database host")
			db, err := gorm.Open(postgres.Open(dbURI), &gorm.Config{
				Logger: logger.New(
					&sublogger,
					logger.Config{
						SlowThreshold:             time.Second,
						LogLevel:                  logger.Info,
						IgnoreRecordNotFoundError: false,
						Colorful:                  false,
					},
				),
			})

			if err != nil {
				sublogger.Fatal().Msg("failed to connect to database")
				time.Sleep(3 * time.Second)
			} else {
				return db, sublogger, nil
			}
		}
	}
}

//NewSQLiteConnector opens a connection to a local sqlite database
func NewSQLiteConnector(log zerolog.Logger) ConnectorFunc {
	return func() (*gorm.DB, zerolog.Logger, error) {
		db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})

		if err == nil {
			db.Exec("PRAGMA foreign_keys = ON")
		}

		return db, log, err
	}
}

//NewDatabaseConnection initializes a new connection to the database and wraps it in a Datastore
func NewDatabaseConnection(connect ConnectorFunc) (Datastore, error) {
	impl, log, err := connect()
	if err != nil {
		return nil, err
	}

	db := &myDB{
		impl: impl.Debug(),
		log:  log,
	}

	db.impl.AutoMigrate(&models.Temperature{}, &models.TemperatureV2{})

	oldtemps := []models.Temperature{}
	result := db.impl.Order("timestamp2").Limit(100).Find(&oldtemps)
	migrationCount := 0
	duplicateCount := 0

	log.Info().Msg("checking for temperature data to be migrated ...")

	for result.Error == nil && len(oldtemps) > 0 {

		for _, old := range oldtemps {
			t := &models.TemperatureV2{
				Latitude:  old.Latitude,
				Longitude: old.Longitude,
				Device:    old.Device,
				Temp:      old.Temp,
				Water:     old.Water,
				Timestamp: old.Timestamp2,
			}

			r := db.impl.Create(t)
			if r.Error == nil {
				migrationCount++
			} else {
				duplicateCount++
			}

			db.impl.Delete(&old)
		}

		oldtemps = []models.Temperature{}
		result = db.impl.Order("timestamp2").Limit(100).Find(&oldtemps)
	}

	if migrationCount > 0 {
		log.Info().Msgf("migrated %d temperature values to the new table and removed %d duplicates", migrationCount, duplicateCount)
	} else {
		log.Info().Msg("no old temperature data found")
	}

	return db, nil
}

//AddTemperatureMeasurement takes a device, position and a temp and adds a record to the database
func (db *myDB) AddTemperatureMeasurement(device *string, latitude, longitude, temp float64, water bool, when string) (*models.TemperatureV2, error) {

	ts, err := time.Parse(time.RFC3339Nano, when)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp from %s : (%s)", when, err.Error())
	}

	measurement := &models.TemperatureV2{
		Latitude:  latitude,
		Longitude: longitude,
		Temp:      float32(temp),
		Water:     water,
		Timestamp: ts,
	}

	if device != nil {
		measurement.Device = *device
	}

	result := db.impl.Create(measurement)
	if result.Error != nil {
		return nil, fmt.Errorf("create failed: %s", result.Error.Error())
	}

	return measurement, nil
}

func (db *myDB) GetTemperatures(deviceId string, from, to time.Time, geoSpatial string, lat0, lon0, lat1, lon1 float64, resultLimit uint64) ([]models.TemperatureV2, error) {
	temps := []models.TemperatureV2{}
	gorm := db.impl.Order("timestamp")

	if deviceId != "" {
		gorm = gorm.Where("device = ?", deviceId)
	}

	if !from.IsZero() || !to.IsZero() {
		gorm = insertTemporalSQL(gorm, "timestamp", from, to)
		if gorm.Error != nil {
			return nil, gorm.Error
		}
	}

	if geoSpatial != "" {
		gorm = insertGeoSQL(gorm, lat0, lon0, lat1, lon1)
	}

	result := gorm.Limit(int(resultLimit)).Find(&temps)
	if result.Error != nil {
		return nil, result.Error
	}

	return temps, nil
}

func insertTemporalSQL(gorm *gorm.DB, property string, from, to time.Time) *gorm.DB {
	if !from.IsZero() {
		gorm = gorm.Where(fmt.Sprintf("%s >= ?", property), from)
		if gorm.Error != nil {
			return gorm
		}
	}

	if !to.IsZero() {
		gorm = gorm.Where(fmt.Sprintf("%s < ?", property), to)
	}

	return gorm
}

func insertGeoSQL(gorm *gorm.DB, nw_lat, nw_lon, se_lat, se_lon float64) *gorm.DB {
	// deal with in parameters, check if any of the coords seem dodgy? but what do we consider dodgy or nah
	gorm = gorm.Where(
		"latitude > ? AND latitude < ? AND longitude > ? AND longitude < ?",
		se_lat, nw_lat, nw_lon, se_lon,
	)

	return gorm
}
