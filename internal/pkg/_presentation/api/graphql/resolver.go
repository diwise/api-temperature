// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.
package graphql

import (
	"context"
	"math"
	"time"

	"github.com/diwise/api-temperature/internal/pkg/infrastructure/repositories/database"
	"github.com/diwise/api-temperature/internal/pkg/infrastructure/repositories/models"
)

type Resolver struct{}

func (r *entityResolver) FindDeviceByID(ctx context.Context, id string) (*Device, error) {
	return &Device{ID: id}, nil
}

func convertDatabaseRecordToGQL(measurement *models.Temperature) *Temperature {
	if measurement != nil {
		temp := &Temperature{
			From: &Origin{
				Pos: &WGS84Position{
					Lat: measurement.Latitude,
					Lon: measurement.Longitude,
				},
				Device: &Device{
					ID: measurement.Device,
				},
			},
			When: measurement.Timestamp2.Format(time.RFC3339),
			Temp: math.Round(float64(measurement.Temp*10)) / 10,
		}

		return temp
	}

	return nil
}

func (r *queryResolver) Temperatures(ctx context.Context) ([]*Temperature, error) {
	db, err := database.GetFromContext(ctx)
	if err != nil {
		return nil, err
	}

	temperatures, err := db.GetTemperatures("", time.Time{}, time.Time{}, "", 0.0, 0.0, 0.0, 0.0, uint64(100))

	if err != nil {
		panic("Failed to query latest temperatures.")
	}

	tempcount := len(temperatures)

	if tempcount == 0 {
		return []*Temperature{}, nil
	}

	gqltemps := make([]*Temperature, 0, tempcount)

	for _, v := range temperatures {
		gqltemps = append(gqltemps, convertDatabaseRecordToGQL(&v))
	}

	return gqltemps, nil
}

func (r *Resolver) Entity() EntityResolver { return &entityResolver{r} }
func (r *Resolver) Query() QueryResolver   { return &queryResolver{r} }

type entityResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
