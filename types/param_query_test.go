package types

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFiltersGetNumberRange(t *testing.T) {
	tests := []struct {
		name    string
		filters Filters
		want    []*float64
	}{
		{name: "json number array", filters: Filters{"price": []interface{}{1.5, float64(10)}}, want: numberRange(1.5, 10)},
		{name: "mixed numeric types", filters: Filters{"price": []interface{}{int64(-1), uint8(10)}}, want: numberRange(-1, 10)},
		{name: "numeric strings", filters: Filters{"price": []interface{}{" 1.25 ", "99"}}, want: numberRange(1.25, 99)},
		{name: "open lower bound", filters: Filters{"price": []interface{}{nil, "99"}}, want: []*float64{nil, float64Pointer(99)}},
		{name: "open upper bound", filters: Filters{"price": []interface{}{"1.25", nil}}, want: []*float64{float64Pointer(1.25), nil}},
		{name: "empty bounds", filters: Filters{"price": []interface{}{"", "  "}}, want: nil},
		{name: "nil bounds", filters: Filters{"price": []interface{}{nil, nil}}, want: nil},
		{name: "missing key", filters: Filters{}, want: nil},
		{name: "wrong length", filters: Filters{"price": []interface{}{1}}, want: nil},
		{name: "unsupported slice type", filters: Filters{"price": []int{1, 10}}, want: nil},
		{name: "non-number", filters: Filters{"price": []interface{}{1, "invalid"}}, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filters.GetNumberRange("price"); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GetNumberRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFiltersGetNumberRangeAfterUnmarshal(t *testing.T) {
	var filters Filters
	if err := json.Unmarshal([]byte(`{"price":[1.25,99]}`), &filters); err != nil {
		t.Fatal(err)
	}

	want := numberRange(1.25, 99)
	if got := filters.GetNumberRange("price"); !reflect.DeepEqual(got, want) {
		t.Fatalf("GetNumberRange() = %v, want %v", got, want)
	}
}

func numberRange(from, to float64) []*float64 {
	return []*float64{&from, &to}
}

func float64Pointer(value float64) *float64 {
	return &value
}
