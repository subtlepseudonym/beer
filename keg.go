package main

type Keg struct {
	Type   string  `json:"type"`
	Volume float64 `json:"volume"`
}

var (
	KegCorny = Keg{ // cornelius
		Type:   "corny",
		Volume: 18.93,
	}
	KegSixtel = Keg{ // sixth-barrel
		Type:   "sixtel",
		Volume: 19.55,
	}
	KegQuarter = Keg{ // pony
		Type:   "quarter",
		Volume: 29.34,
	}
	KegHalf = Keg{ // full-size
		Type:   "half-barrel",
		Volume: 58.67,
	}
)
