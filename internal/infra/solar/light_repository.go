package solar

import (
	"context"
	"math"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	julianUnixEpoch      = 2440587.5
	julianYear2000       = 2451545.0
	secondsPerDay        = 86400.0
	earthAxialTilt       = 23.4397
	sunriseSolarAltitude = -0.833
)

type NoSunriseError struct {
	Date time.Time
}

func (e NoSunriseError) Error() string {
	return "there is no sunrise or sunset on " + e.Date.Format(time.DateOnly)
}

type LightRepository struct {
	latitude, longitude float64
	location            *time.Location
	tracer              trace.Tracer
}

func (l LightRepository) Find(ctx context.Context, date time.Time) (*status.Light, error) {
	_, span := l.tracer.Start(ctx, "SolarLightRepository.Find")
	defer span.End()
	sunrise, sunset, err := l.sunTimes(date.In(l.location))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	light, err := status.NewLight(sunrise, sunset)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "light calculated")
	return light, nil
}

func (l LightRepository) sunTimes(date time.Time) (sunrise, sunset time.Time, err error) {
	midnightUTC := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	julianDate := float64(midnightUTC.Unix())/secondsPerDay + julianUnixEpoch
	dayNumber := math.Ceil(julianDate - julianYear2000 + 0.0008)
	meanSolarTime := dayNumber - l.longitude/360
	meanAnomaly := normalizeDegrees(357.5291 + 0.98560028*meanSolarTime)
	center := 1.9148*sinDegrees(meanAnomaly) + 0.02*sinDegrees(2*meanAnomaly) + 0.0003*sinDegrees(3*meanAnomaly)
	eclipticLongitude := normalizeDegrees(meanAnomaly + center + 180 + 102.9372)
	solarTransit := julianYear2000 + meanSolarTime + 0.0053*sinDegrees(meanAnomaly) - 0.0069*sinDegrees(2*eclipticLongitude)
	sinDeclination := sinDegrees(eclipticLongitude) * sinDegrees(earthAxialTilt)
	cosDeclination := math.Cos(math.Asin(sinDeclination))
	cosHourAngle := (sinDegrees(sunriseSolarAltitude) - sinDegrees(l.latitude)*sinDeclination) /
		(cosDegrees(l.latitude) * cosDeclination)
	if cosHourAngle < -1 || cosHourAngle > 1 {
		return time.Time{}, time.Time{}, NoSunriseError{Date: date}
	}
	hourAngle := math.Acos(cosHourAngle) * 180 / math.Pi
	sunrise = l.fromJulian(solarTransit - hourAngle/360)
	sunset = l.fromJulian(solarTransit + hourAngle/360)
	return sunrise, sunset, nil
}

func (l LightRepository) fromJulian(julian float64) time.Time {
	seconds := (julian - julianUnixEpoch) * secondsPerDay
	return time.Unix(0, int64(seconds*float64(time.Second))).In(l.location).Truncate(time.Second)
}

func normalizeDegrees(degrees float64) float64 {
	return math.Mod(math.Mod(degrees, 360)+360, 360)
}

func sinDegrees(degrees float64) float64 {
	return math.Sin(degrees * math.Pi / 180)
}

func cosDegrees(degrees float64) float64 {
	return math.Cos(degrees * math.Pi / 180)
}

func NewLightRepository(latitude, longitude float64, location *time.Location, tracer trace.Tracer) LightRepository {
	return LightRepository{latitude: latitude, longitude: longitude, location: location, tracer: tracer}
}
