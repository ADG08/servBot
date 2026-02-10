package tz

import "time"

// Paris is the Europe/Paris location (CET/CEST with automatic DST).
var Paris *time.Location

func init() {
	var err error
	Paris, err = time.LoadLocation("Europe/Paris")
	if err != nil {
		panic("tz: load Europe/Paris: " + err.Error())
	}
}
