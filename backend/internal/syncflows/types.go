package syncflows

import "time"

type BatchRunResult struct {
	Total    int
	Success  int
	Failed   int
	Errors   []error
	Duration time.Duration
}
