package gqbd

import (
	"encoding/json"
	"math"
	"testing"
)

// FuzzAppendJSON checks that the Elasticsearch and Qdrant body encoder writes
// exactly what encoding/json would for the values it encodes itself.
func FuzzAppendJSON(f *testing.F) {
	f.Add("plain", 1.5, uint64(0))
	f.Add("<a href=\"x\">&  \x00\x1f\xff", 1e21, math.Float64bits(1e-7))
	f.Add("한국어 ✓", math.Copysign(0, -1), math.Float64bits(float64(float32(3.4e38))))
	f.Fuzz(func(t *testing.T, s string, f64 float64, bits uint64) {
		values := []interface{}{
			s, f64, math.Float64frombits(bits), float32(math.Float64frombits(bits)),
			[]string{s, ""}, []float32{float32(f64)},
			map[string]interface{}{s: f64, "k": []interface{}{nil, true, s, int64(-7), uint8(9)}},
		}
		for _, v := range values {
			want, wantErr := json.Marshal(v)
			got, gotErr := appendJSON(nil, v)
			if (gotErr != nil) != (wantErr != nil) {
				t.Fatalf("appendJSON(%#v) error = %v, json.Marshal error = %v", v, gotErr, wantErr)
			}
			if wantErr == nil && string(got) != string(want) {
				t.Fatalf("appendJSON(%#v) = %s, json.Marshal = %s", v, got, want)
			}
		}
	})
}
