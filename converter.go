package gorql

import (
	"strconv"
	"time"
)

// convert float to int.
func convertInt(v interface{}) interface{} {
	if s, err := strconv.Atoi(v.(string)); err == nil {
		return s
	}
	return v
}

// convert string to float.
func convertFloat(v interface{}) interface{} {
	if s, err := strconv.ParseFloat(v.(string), 64); err == nil {
		return s
	}
	return v
}

// convert string to time object.
func convertTime(layout string) func(interface{}) interface{} {
	return func(v interface{}) interface{} {
		t, _ := time.Parse(layout, v.(string))
		return t
	}
}

// convert string to bool.
func convertBool(v interface{}) interface{} {
	if s, err := strconv.ParseBool(v.(string)); err == nil {
		return s
	}
	return v
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
