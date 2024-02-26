package gorql

import (
	"time"
)

// convert float to int.
func convertInt(v interface{}) interface{} {
	return int(v.(float64))
}

// convert string to time object.
func convertTime(layout string) func(interface{}) interface{} {
	return func(v interface{}) interface{} {
		t, _ := time.Parse(layout, v.(string))
		return t
	}
}

// nop converter.
func valueFn(v interface{}) interface{} {
	return v
}

// layouts holds all standard time.Time layouts.
var layouts = map[string]string{
	"ANSIC":       time.ANSIC,
	"UnixDate":    time.UnixDate,
	"RubyDate":    time.RubyDate,
	"RFC822":      time.RFC822,
	"RFC822Z":     time.RFC822Z,
	"RFC850":      time.RFC850,
	"RFC1123":     time.RFC1123,
	"RFC1123Z":    time.RFC1123Z,
	"RFC3339":     time.RFC3339,
	"RFC3339Nano": time.RFC3339Nano,
	"Kitchen":     time.Kitchen,
	"Stamp":       time.Stamp,
	"StampMilli":  time.StampMilli,
	"StampMicro":  time.StampMicro,
	"StampNano":   time.StampNano,
}
