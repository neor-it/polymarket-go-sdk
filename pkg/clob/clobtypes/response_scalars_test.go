package clobtypes

import (
	"encoding/json"
	"testing"
)

const exactEarning = "9007199254740993.000000000000000001"

func TestEarningsAcceptNumbersWithoutPrecisionLoss(t *testing.T) {
	for _, body := range []string{`{"earnings":` + exactEarning + `,"asset_rate":0}`, `{"earnings":"` + exactEarning + `","asset_rate":"0"}`} {
		var user UserEarning
		var total TotalUserEarning
		var nested Earning
		if err := json.Unmarshal([]byte(body), &user); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(body), &total); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(body), &nested); err != nil {
			t.Fatal(err)
		}
		if user.Earnings != exactEarning || total.Earnings != exactEarning || nested.Earnings != exactEarning || total.AssetRate != "0" {
			t.Fatal("earnings precision or explicit zero lost")
		}
	}
	for _, body := range []string{`{"earnings":null}`, `{}`, `{"earnings":"NaN"}`} {
		var value UserEarning
		if json.Unmarshal([]byte(body), &value) == nil {
			t.Fatalf("invalid earnings accepted: %s", body)
		}
	}
}
func TestServerTimeRejectsUnavailableValues(t *testing.T) {
	for _, body := range []string{`null`, `{}`, `{"timestamp":null}`, `-1`, `{"timestamp":1.5}`} {
		var value TimeResponse
		if json.Unmarshal([]byte(body), &value) == nil {
			t.Fatalf("invalid time accepted: %s", body)
		}
	}
}
