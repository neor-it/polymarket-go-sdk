package data

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/neor-it/polymarket-go-sdk/pkg/types"
)

func TestRowsMatchPinnedOfficialOpenAPI(t *testing.T) {
	payload, err := os.ReadFile("testdata/openapi-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]json.RawMessage `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatal(err)
	}
	rows := []any{Position{}, Trade{}, Activity{}, Holder{}, MetaHolder{}, PortfolioValue{}, OpenInterest{}, LiveVolume{}, ConditionVolume{}, BuilderStanding{}, BuilderVolumePoint{}, LeaderboardEntry{}, LeaderboardUserEntry{}, BiggestWinner{}, UserStats{}, UserPnlPoint{}, UserPnlSeries{}, UserVolume{}, PricePoint{}, ApprovalContract{}, Approvals{}, Resolution{}, TokenRef{}, ServiceStatus{}, ServingFreshness{}, ServingMechanism{}, IngestionFreshness{}, CursorLag{}, ComboActivity{}, ComboPosition{}, ComboLeg{}, ComboLegMarket{}, ComboLegEvent{}}
	decimalType := reflect.TypeOf(types.Decimal{})
	for _, row := range rows {
		typ := reflect.TypeOf(row)
		t.Run(typ.Name(), func(t *testing.T) {
			schema, exists := document.Components.Schemas[typ.Name()]
			if !exists {
				t.Fatal("schema missing")
			}
			fields := map[string]reflect.Type{}
			for index := 0; index < typ.NumField(); index++ {
				field := typ.Field(index)
				fields[strings.Split(field.Tag.Get("json"), ",")[0]] = field.Type
			}
			for name, raw := range schema.Properties {
				field, ok := fields[name]
				if !ok {
					t.Errorf("documented field %s missing", name)
					continue
				}
				delete(fields, name)
				var property struct {
					Type json.RawMessage `json:"type"`
				}
				if err := json.Unmarshal(raw, &property); err != nil {
					t.Fatal(err)
				}
				var kinds []string
				if len(property.Type) > 0 && property.Type[0] == '[' {
					if err := json.Unmarshal(property.Type, &kinds); err != nil {
						t.Fatal(err)
					}
				} else {
					var kind string
					if len(property.Type) > 0 {
						if err := json.Unmarshal(property.Type, &kind); err != nil {
							t.Fatal(err)
						}
						kinds = []string{kind}
					}
				}
				nullable := false
				monetary := false
				for _, kind := range kinds {
					nullable = nullable || kind == "null"
					monetary = monetary || kind == "number"
				}
				if nullable && field.Kind() != reflect.Pointer && field.Kind() != reflect.Slice {
					t.Errorf("nullable field %s loses null distinction", name)
				}
				if field.Kind() == reflect.Pointer {
					field = field.Elem()
				}
				if monetary && field != decimalType {
					t.Errorf("financial field %s uses %v instead of decimal", name, field)
				}
			}
			for name := range fields {
				t.Errorf("undocumented field %s", name)
			}
		})
	}
}
