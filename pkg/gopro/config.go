package gopro

import (
	"fmt"

	"github.com/spf13/viper"
)

const parent = "gopro"

func gpsMinAccuracyFromConfig() uint {
	key := fmt.Sprintf("%s.gps_accuracy", parent)
	viper.SetDefault(key, 500)
	return viper.GetUint(key)
}

func gpsMaxAltitudeFromConfig() float64 {
	key := fmt.Sprintf("%s.gps_max_altitude", parent)
	viper.SetDefault(key, 8000)
	return float64(viper.GetUint(key))
}

func gpsCountryCodesFromConfig() []string {
	key := fmt.Sprintf("%s.gps_country_codes", parent)
	viper.SetDefault(key, []string{}) // 3d lock, 2d lock
	return viper.GetStringSlice(key)
}

func gpsMaxCountryCodesFromConfig() int {
	key := fmt.Sprintf("%s.gps_max_country_codes", parent)
	viper.SetDefault(key, 5)
	return viper.GetInt(key)
}
