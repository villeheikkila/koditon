package sourcejson

type BackfillResult struct {
	Scanned int `json:"scanned"`
	Updated int `json:"updated"`
	Batches int `json:"batches"`
}
