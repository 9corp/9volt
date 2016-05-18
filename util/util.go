package util

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"os"
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

func MD5Hash(data string, length int) string {
	hasher := md5.New()
	hasher.Write([]byte(data))
	hash := hex.EncodeToString(hasher.Sum(nil))

	if len(hash) < length {
		return hash
	}

	return hash[:length]
}

func GetMemberID(suffix string) string {
	hostname, _ := os.Hostname()
	return MD5Hash(hostname+":"+suffix, 8)
}
