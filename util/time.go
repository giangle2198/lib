package util

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

const (
	DateYYYYMMDDFormat         = "20060102"
	DateYYYYMMDDHH24MISSFormat = "20060102150405"
)

func FormatTime(inputTime, inFormat, outFormat string) (string, error) {
	if inputTime == "" {
		return "", nil
	}
	if inFormat == "" {
		inFormat = DateYYYYMMDDFormat
	}
	timeParser, err := time.Parse(inFormat, inputTime)
	if err != nil {
		return "", errors.New("can't format time")
	}
	if outFormat == "" {
		outFormat = DateYYYYMMDDHH24MISSFormat
	}
	return timeParser.Format(outFormat), nil
}

// ConvertDateTimeToTimeStamp return timestamp in string
func ConvertDateTimeToTimeStamp(dateTime string) (string, error) {
	if strings.EqualFold(strings.TrimSpace(dateTime), "") {
		return "", nil
	}

	timeConverted, err := time.Parse(time.RFC3339, dateTime)
	if err != nil {
		return dateTime, err
	}
	return strconv.FormatInt(timeConverted.Unix(), 10), nil
}

func InTimeSpan(start, end, check time.Time) bool {
	if start.Before(end) {
		return !check.Before(start) && !check.After(end)
	}
	if start.Equal(end) {
		return check.Equal(start)
	}
	return !start.After(check) || !end.Before(check)
}
