package cells

import (
	"encoding/json"

	"github.com/paulmach/orb"
)

func jsonUnmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }

func orbPt(lon, lat float64) orb.Point { return orb.Point{lon, lat} }
