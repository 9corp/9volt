package util

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"os"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyz1234567890"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
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

// From @stackoverflow; generate an optionally seeded, N length random string
func RandomString(n int, seed bool) string {
	if seed {
		rand.Seed(time.Now().Unix())
	}

	b := make([]byte, n)
	for i := 0; i < n; {
		if idx := int(rand.Int63() & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i++
		}
	}
	return string(b)
}

func GetMemberID(suffix string) string {
	hostname, _ := os.Hostname()
	return MD5Hash(hostname+":"+suffix, 8)
}

func StringSliceContains(stringSlice []string, data string) bool {
	for _, v := range stringSlice {
		if v == data {
			return true
		}
	}

	return false
}

// Return true if ANY element in s1 appears in s2, otherwise return false
func StringSliceInStringSlice(s1, s2 []string) bool {
	for _, v1 := range s1 {
		for _, v2 := range s2 {
			if v1 == v2 {
				return true
			}
		}
	}

	return false
}
