package util

import (
	"encoding/json"
	"time"
)

type CustomDuration time.Duration

func (cd *CustomDuration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	duration, err := time.ParseDuration(s)

	if err != nil {
		return err
	}

	*cd = (CustomDuration)(duration)

	return nil
}

func (cd *CustomDuration) String() string {
	return time.Duration(*cd).String()
}

func (cd *CustomDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(*cd).String())
}
