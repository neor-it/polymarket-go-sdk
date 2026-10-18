package data

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// UnmarshalJSON requires the published data envelope and preserves documented null misses.
// Only pointer payloads represent nullable endpoint contracts, such as user stats.
func (response *Envelope[T]) UnmarshalJSON(payload []byte) error {
	var wire struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &wire); err != nil {
		return errors.Wrap(err, "decode data envelope")
	}
	if len(wire.Data) == 0 {
		return errors.New("data envelope is missing required data")
	}
	if isJSONNull(wire.Data) && reflect.TypeOf((*T)(nil)).Elem().Kind() != reflect.Pointer {
		return errors.New("non-nullable data envelope contains null")
	}
	var decoded T
	if err := json.Unmarshal(wire.Data, &decoded); err != nil {
		return errors.Wrap(err, "decode envelope data")
	}
	response.Data = decoded
	return nil
}

// UnmarshalJSON rejects incomplete pages instead of presenting them as an empty final page.
func (response *Page[T]) UnmarshalJSON(payload []byte) error {
	var wire struct {
		Data       json.RawMessage `json:"data"`
		Pagination json.RawMessage `json:"pagination"`
	}
	if err := json.Unmarshal(payload, &wire); err != nil {
		return errors.Wrap(err, "decode cursor page")
	}
	data := bytes.TrimSpace(wire.Data)
	if len(data) == 0 || data[0] != '[' {
		return errors.New("cursor page data must be an array")
	}
	if len(wire.Pagination) == 0 || isJSONNull(wire.Pagination) {
		return errors.New("cursor page pagination is required and must not be null")
	}
	var decoded Page[T]
	if err := json.Unmarshal(wire.Data, &decoded.Data); err != nil {
		return errors.Wrap(err, "decode cursor page rows")
	}
	if err := json.Unmarshal(wire.Pagination, &decoded.Pagination); err != nil {
		return errors.Wrap(err, "decode cursor page pagination")
	}
	*response = decoded
	return nil
}

// UnmarshalJSON validates required pagination metadata before exposing a continuation.
func (pagination *Pagination) UnmarshalJSON(payload []byte) error {
	var wire struct {
		Limit      *int64  `json:"limit"`
		Offset     *int64  `json:"offset"`
		HasMore    *bool   `json:"has_more"`
		NextCursor *string `json:"next_cursor"`
	}
	if err := json.Unmarshal(payload, &wire); err != nil {
		return errors.Wrap(err, "decode pagination metadata")
	}
	if wire.Limit == nil || wire.Offset == nil || wire.HasMore == nil {
		return errors.New("pagination limit, offset and has_more are required")
	}
	if *wire.Limit < 0 || *wire.Offset < 0 {
		return errors.New("pagination limit and offset must not be negative")
	}
	if *wire.HasMore && (wire.NextCursor == nil || strings.TrimSpace(*wire.NextCursor) == "") {
		return errors.New("pagination has_more requires a nonempty next_cursor")
	}
	if !*wire.HasMore && wire.NextCursor != nil {
		return errors.New("pagination next_cursor requires has_more")
	}
	*pagination = Pagination{Limit: *wire.Limit, Offset: *wire.Offset, HasMore: *wire.HasMore, NextCursor: wire.NextCursor}
	return nil
}

func isJSONNull(payload []byte) bool { return bytes.Equal(bytes.TrimSpace(payload), []byte("null")) }
