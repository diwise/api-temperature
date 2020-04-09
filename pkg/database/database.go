package database

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"

	"github.com/iot-for-tillgenglighet/api-temperature/pkg/models"
)

//Datastore is an interface that is used to inject the database into different handlers to improve testability
type Datastore interface {
	AddTemperatureMeasurement(device *string, latitude, longitude, temp float64, when string) (*models.Temperature, error)
	GetLatestTemperatures() ([]models.Temperature, error)
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

	return nil, errors.New("Failed to decode database from context")
}

type myDB struct {
	impl *gorm.DB
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

//NewDatabaseConnection initializes a new connection to the database and wraps it in a Datastore
func NewDatabaseConnection() (Datastore, error) {
	db := &myDB{}

	dbHost := os.Getenv("TEMPERATURE_DB_HOST")
	username := os.Getenv("TEMPERATURE_DB_USER")
	dbName := os.Getenv("TEMPERATURE_DB_NAME")
	password := os.Getenv("TEMPERATURE_DB_PASSWORD")
	sslMode := getEnv("TEMPERATURE_DB_SSLMODE", "require")

	dbURI := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=%s password=%s", dbHost, username, dbName, sslMode, password)

	for {
		log.Printf("Connecting to database host %s ...\n", dbHost)
		conn, err := gorm.Open("postgres", dbURI)
		if err != nil {
			log.Fatalf("Failed to connect to database %s \n", err)
			time.Sleep(3 * time.Second)
		} else {
			db.impl = conn
			db.impl.Debug().AutoMigrate(&models.Temperature{})
			break
		}
		defer conn.Close()
	}

	return db, nil
}

//AddTemperatureMeasurement takes a device, position and a temp and adds a record to the database
func (db *myDB) AddTemperatureMeasurement(device *string, latitude, longitude, temp float64, when string) (*models.Temperature, error) {

	measurement := &models.Temperature{
		Latitude:  latitude,
		Longitude: longitude,
		Temp:      float32(temp),
		Timestamp: when,
	}

	if device != nil {
		measurement.Device = *device
	}

	db.impl.Create(measurement)

	return measurement, nil
}

//GetLatestTemperatures returns the most recent value for all sensors that have reported
//a value during the last 24 hours
func (db *myDB) GetLatestTemperatures() ([]models.Temperature, error) {
	// Get temperatures from the last 24 hours
	queryStart := time.Now().UTC().AddDate(0, 0, -1).Format(time.RFC3339)

	latestTemperatures := []models.Temperature{}
	db.impl.Table("temperatures").Select("DISTINCT ON (device) *").Where("timestamp > ?", queryStart).Order("device, timestamp desc").Find(&latestTemperatures)
	return latestTemperatures, nil
}
