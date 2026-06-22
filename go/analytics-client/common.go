package analytics

import "time"

type TripleWindow struct {
	AllTime *time.Duration `json:"all_time"`
	Monthly *time.Duration `json:"monthly"`
	Weekly  *time.Duration `json:"weekly"`
}

func blankTripleWindow() TripleWindow {
	return TripleWindow{
		AllTime: nil,
		Monthly: nil,
		Weekly:  nil,
	}
}
