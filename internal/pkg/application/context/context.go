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

	temperatures, err = getTemperatures(cs.db, query)
	if err != nil {
		return fmt.Errorf("something went wrong when retrieving temperatures from database: %s", err)
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

func getTemperatures(db database.Datastore, query ngsi.Query) ([]models.Temperature, error) {
	deviceID := ""
	if query.Device() != "" {
		deviceID = strings.TrimPrefix(query.Device(), fiware.DeviceIDPrefix)
	}

	// get temperatures from past 24 hours by default
	from := time.Now().UTC().AddDate(0, 0, -1)
	to := time.Now().UTC()
	if query.IsTemporalQuery() {
		from, to = query.Temporal().TimeSpan()
	}

	limit := query.PaginationLimit()

	if query.IsGeoQuery() {
		geo := query.Geo()
		if geo.GeoRel == ngsi.GeoSpatialRelationNearPoint {
			lon, lat, _ := geo.Point()
			distance, _ := geo.Distance()

			nw_lat, nw_lon, se_lat, se_lon := getApproximatePoint(lat, lon, uint64(distance))

			return db.GetTemperatures(deviceID, from, to, geo.GeoRel, nw_lat, nw_lon, se_lat, se_lon, limit)
		} else if geo.GeoRel == ngsi.GeoSpatialRelationWithinRect {
			nw_lat, nw_lon, se_lat, se_lon, err := geo.Rectangle()
			if err != nil {
				return nil, err
			}
			return db.GetTemperatures(deviceID, from, to, geo.GeoRel, nw_lat, nw_lon, se_lat, se_lon, limit)
		}
	}

	return db.GetTemperatures(deviceID, from, to, "", 0.0, 0.0, 0.0, 0.0, query.PaginationLimit())
}

func queriedAttributesDoNotInclude(attributes []string, requiredAttribute string) bool {
	for _, attr := range attributes {
		if attr == requiredAttribute {
			return false
		}
	}

	return true
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
