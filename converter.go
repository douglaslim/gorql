package gorql

import (
	"fmt"
	"strconv"
	"time"
)

// convert float to int.
func convertInt(v interface{}) (interface{}, error) {
	s, err := strconv.Atoi(v.(string))
	if err != nil {
		return nil, fmt.Errorf("unable to convert %s to int: %s", v.(string), err)
	}
	return s, nil
}

// convert string to float.
func convertFloat(v interface{}) (interface{}, error) {
	s, err := strconv.ParseFloat(v.(string), 64)
	if err != nil {
		return nil, fmt.Errorf("unable to convert %s to float: %s", v.(string), err)
	}
	return s, nil
}

// convert string to time object.
func convertTime(layout string) func(interface{}) (interface{}, error) {
	return func(v interface{}) (interface{}, error) {
		t, err := time.Parse(layout, v.(string))
		if err != nil {
			return nil, fmt.Errorf("failed to parse date layout %s for %s", layout, v.(string))
		}
		return t, nil
	}
}

// convert string to bool.
func convertBool(v interface{}) (interface{}, error) {
	s, err := strconv.ParseBool(v.(string))
	if err != nil {
		return nil, fmt.Errorf("unable to convert %s to bool: %s", v.(string), err)
	}
	return s, nil
}

// nop converter.
func valueFn(v interface{}) (interface{}, error) {
	return v, nil
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
