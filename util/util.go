package util

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"os"
	"strings"
	"time"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyz1234567890"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
)

type CustomDuration time.Duration

func (cd *CustomDuration) UnmarshalJSON(data []byte) error {
	// we got a string
	if data[0] == '"' {
		sd := string(data[1 : len(data)-1])
		duration, err := time.ParseDuration(sd)
		if err != nil {
			return err
		}

		*cd = (CustomDuration)(duration)
		return nil
	}

	// not a string so it must be a number
	var id int64
	id, err := json.Number(string(data)).Int64()
	if err != nil {
		return err
	}

	duration := time.Duration(id)

	*cd = (CustomDuration)(duration)

	return nil
}

func (cd *CustomDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(*cd).String())
}

func (cd *CustomDuration) String() string {
	return time.Duration(*cd).String()
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

func SplitTags(tags string) []string {
	// If tags are empty, return empty slice
	if tags == "" {
		return []string{}
	}

	tags = strings.Replace(tags, " ", "", -1)
	return strings.Split(tags, ",")
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

// Helper for fetching keys in a map
func GetMapKeys(inputMap map[string][]byte) []string {
	keys := make([]string, len(inputMap))

	for k := range inputMap {
		keys = append(keys, k)
	}

	return keys
}
