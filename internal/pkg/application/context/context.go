package context

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/diwise/api-temperature/internal/pkg/infrastructure/repositories/database"
	"github.com/diwise/api-temperature/internal/pkg/infrastructure/repositories/models"
	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
	ngsi "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld"
	"github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/types"
)

type contextSource struct {
	db database.Datastore
}

//CreateSource instantiates and returns a Fiware ContextSource that wraps the provided db interface
func CreateSource(db database.Datastore) ngsi.ContextSource {
	return &contextSource{db: db}
}

func convertDatabaseRecordToWaterQualityObserved(r *models.Temperature) *fiware.WaterQualityObserved {
	if r != nil {
		entity := fiware.NewWaterQualityObserved("temperature:"+r.Device, r.Latitude, r.Longitude, r.Timestamp2.Format(time.RFC3339))
		entity.Temperature = types.NewNumberProperty(math.Round(float64(r.Temp*10)) / 10)
		return entity
	}

	return nil
}

func convertDatabaseRecordToWeatherObserved(r *models.Temperature) *fiware.WeatherObserved {
	if r != nil {
		entity := fiware.NewWeatherObserved("temperature:"+r.Device, r.Latitude, r.Longitude, r.Timestamp2.Format(time.RFC3339))
		entity.Temperature = types.NewNumberProperty(math.Round(float64(r.Temp*10)) / 10)
		return entity
	}

	return nil
}

func (cs contextSource) CreateEntity(typeName, entityID string, req ngsi.Request) error {
	return errors.New("create entity not supported for type " + typeName)
}

func (cs contextSource) GetEntities(query ngsi.Query, callback ngsi.QueryEntitiesCallback) error {

	var temperatures []models.Temperature
	var err error

	if query == nil {
		return errors.New("GetEntities: query may not be nil")
	}

	includeAirTemperature := false
	includeWaterTemperature := false

	for _, typeName := range query.EntityTypes() {
		if typeName == "WeatherObserved" {
			includeAirTemperature = true
		} else if typeName == "WaterQualityObserved" {
			includeWaterTemperature = true
		}
	}

	if !includeAirTemperature && !includeWaterTemperature {
		// No provided type specified, but maybe the caller specified an attribute list instead?
		if queriedAttributesDoNotInclude(query.EntityAttributes(), "temperature") {
			return errors.New("GetEntities called without specifying a type that is provided by this service")
		}

		// Include both entity types as they both hold a temperature value
		includeAirTemperature = true
		includeWaterTemperature = true
	}

	if query.HasDeviceReference() {
		deviceID := strings.TrimPrefix(query.Device(), fiware.DeviceIDPrefix)
		temperatures, err = getTemperaturesWithDeviceID(cs.db, deviceID)
	} else if !query.IsGeoQuery() && !query.IsTemporalQuery() {
		temperatures, err = getLatestTemperaturesFrom(cs.db)
	} else if query.IsGeoQuery() && !query.IsTemporalQuery() {
		temperatures, err = getTemperaturesWithGeoQuery(cs.db, query.Geo(), query.PaginationLimit())
	} else if query.IsTemporalQuery() && !query.IsGeoQuery() {
		temperatures, err = getTemperaturesWithinTimespan(cs.db, query.Temporal(), query.PaginationLimit())
	} else if query.IsGeoQuery() && query.IsTemporalQuery() {
		temperatures, err = getTemperaturesAtTimeAndPlace(cs.db, query.Geo(), query.Temporal(), query.PaginationLimit())
	}

	if err == nil {
		for _, v := range temperatures {
			if !v.Water && includeAirTemperature {
				err = callback(convertDatabaseRecordToWeatherObserved(&v))
			} else if v.Water && includeWaterTemperature {
				err = callback(convertDatabaseRecordToWaterQualityObserved(&v))
			}
			if err != nil {
				break
			}
		}
	}

	return err
}

func (cs contextSource) GetProvidedTypeFromID(entityID string) (string, error) {
	return "", errors.New("not implemented")
}

func (cs contextSource) ProvidesAttribute(attributeName string) bool {
	return attributeName == "temperature"
}

func (cs contextSource) ProvidesEntitiesWithMatchingID(entityID string) bool {
	return strings.HasPrefix(entityID, "urn:ngsi-ld:WeatherObserved:") ||
		strings.HasPrefix(entityID, "urn:ngsi-ld:WaterQualityObserved:")
}

func (cs contextSource) ProvidesType(typeName string) bool {
	return typeName == "WeatherObserved" || typeName == "WaterQualityObserved"
}

func (cs contextSource) RetrieveEntity(entityID string, request ngsi.Request) (ngsi.Entity, error) {
	return nil, errors.New("retrieve entity not implemented")
}

func (cs contextSource) UpdateEntityAttributes(entityID string, req ngsi.Request) error {
	return errors.New("UpdateEntityAttributes is not supported by this service")
}

func getLatestTemperaturesFrom(db database.Datastore) ([]models.Temperature, error) {
	return db.GetLatestTemperatures()
}

func getTemperaturesWithDeviceID(db database.Datastore, deviceID string) ([]models.Temperature, error) {
	return db.GetTemperaturesWithDeviceID(deviceID)
}

func getTemperaturesWithinTimespan(db database.Datastore, tempQ ngsi.TemporalQuery, limit uint64) ([]models.Temperature, error) {
	from, to := tempQ.TimeSpan()
	return db.GetTemperaturesWithinTimespan(from, to, limit)
}

func getTemperaturesAtTimeAndPlace(db database.Datastore, geoQ ngsi.GeoQuery, tempQ ngsi.TemporalQuery, limit uint64) ([]models.Temperature, error) {
	from, to := tempQ.TimeSpan()

	if geoQ.GeoRel == ngsi.GeoSpatialRelationNearPoint {
		lon, lat, _ := geoQ.Point()
		distance, _ := geoQ.Distance()
		return db.GetTemperaturesNearPointAtTime(lat, lon, uint64(distance), from, to, limit)
	} else if geoQ.GeoRel == ngsi.GeoSpatialRelationWithinRect {
		lon0, lat0, lon1, lat1, err := geoQ.Rectangle()
		if err != nil {
			return nil, err
		}
		return db.GetTemperaturesWithinRectangleAtTime(lat0, lon0, lat1, lon1, from, to, limit)
	}

	return nil, fmt.Errorf("geo query relation %s is not supported", geoQ.GeoRel)
}

func getTemperaturesWithGeoQuery(db database.Datastore, geoQ ngsi.GeoQuery, limit uint64) ([]models.Temperature, error) {

	if geoQ.GeoRel == ngsi.GeoSpatialRelationNearPoint {
		lon, lat, _ := geoQ.Point()
		distance, _ := geoQ.Distance()
		return db.GetTemperaturesNearPoint(lat, lon, uint64(distance), limit)
	} else if geoQ.GeoRel == ngsi.GeoSpatialRelationWithinRect {
		lon0, lat0, lon1, lat1, err := geoQ.Rectangle()
		if err != nil {
			return nil, err
		}
		return db.GetTemperaturesWithinRect(lat0, lon0, lat1, lon1, limit)
	}

	return nil, fmt.Errorf("geo query relation %s is not supported", geoQ.GeoRel)
}

func queriedAttributesDoNotInclude(attributes []string, requiredAttribute string) bool {
	for _, attr := range attributes {
		if attr == requiredAttribute {
			return false
		}
	}

	return true
}
