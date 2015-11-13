// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package boomer

import (
	"sort"
	"time"
)

const (
	barChar = "∎"
)

type Report struct {
	AvgTotal float64 `json:"avg_total"`
	Fastest  float64 `json:"fastest"`
	Slowest  float64 `json:"slowest"`
	Average  float64 `json:"average"`
	RPS      float64 `json:"rps"`

	// duration in ms
	TotalDuration int          `json:"total_duration"`
	Errors        []Error      `json:"errors"`
	StatusCodes   []StatusCode `json:"status_codes"`
	Percentiales  []Percential `json:"percentiales"`
	Histogram     []Bucket     `json:"histogram"`

	Lats      []float64 `json:"lats"`
	SizeTotal int64     `json:"size_total"`

	errorDist      map[string]int
	statusCodeDist map[int]int
	results        chan *result
	total          time.Duration
	output         string
}

type Percential struct {
	Percent int     `json:"percent"`
	Count   float64 `json:"count"`
}

type Bucket struct {
	Bucket float64 `json:"bucket"`
	Count  int     `json:"count"`
}

type StatusCode struct {
	Code  int `json:"code"`
	Count int `json:"count"`
}

type Error struct {
	Error string `json:"error"`
	Count int    `json:"count"`
}

func newReport(size int, results chan *result, output string, total time.Duration) *Report {
	return &Report{
		output:         output,
		results:        results,
		total:          total,
		statusCodeDist: make(map[int]int),
		errorDist:      make(map[string]int),
	}
}

func (r *Report) finalize() {
	for {
		select {
		case res := <-r.results:
			if res.err != nil {
				r.errorDist[res.err.Error()]++
			} else {
				r.Lats = append(r.Lats, res.duration.Seconds()*1000)
				r.AvgTotal += res.duration.Seconds()
				r.statusCodeDist[res.statusCode]++
				if res.contentLength > 0 {
					r.SizeTotal += res.contentLength
				}
			}
		default:
			r.RPS = float64(len(r.Lats)) / r.total.Seconds()
			r.Average = r.AvgTotal / float64(len(r.Lats))
			sort.Float64s(r.Lats)
			if len(r.Lats) == 0 {
				return
			}

			r.Fastest = r.Lats[0]
			r.Slowest = r.Lats[len(r.Lats)-1]
			r.printStatusCodes()
			r.printStatusCodes()
			r.printLatencies()
			r.printHistogram()
			return
		}
	}
}

func (r *Report) printLatencies() {
	pctls := []int{10, 25, 50, 75, 90, 95, 99}
	data := make([]float64, len(pctls))
	j := 0
	for i := 0; i < len(r.Lats) && j < len(pctls); i++ {
		current := i * 100 / len(r.Lats)
		if current >= pctls[j] {
			data[j] = r.Lats[i]
			j++
		}
	}

	for i := 0; i < len(pctls); i++ {
		if data[i] > 0 {
			r.Percentiales = append(r.Percentiales, Percential{
				Percent: pctls[i],
				Count:   data[i],
			})
		}
	}
}

func (r *Report) printHistogram() {
	bc := 10
	buckets := make([]float64, bc+1)
	counts := make([]int, bc+1)
	bs := (r.Slowest - r.Fastest) / float64(bc)
	for i := 0; i < bc; i++ {
		buckets[i] = r.Fastest + bs*float64(i)
	}
	buckets[bc] = r.Slowest
	var bi int
	var max int
	for i := 0; i < len(r.Lats); {
		if r.Lats[i] <= buckets[bi] {
			i++
			counts[bi]++
			if max < counts[bi] {
				max = counts[bi]
			}
		} else if bi < len(buckets)-1 {
			bi++
		}
	}

	for i := 0; i < len(buckets); i++ {
		r.Histogram = append(r.Histogram, Bucket{
			Bucket: buckets[i],
			Count:  counts[i],
		})
	}
}

// Prints status code distribution.
func (r *Report) printStatusCodes() {
	for code, num := range r.statusCodeDist {
		r.StatusCodes = append(r.StatusCodes, StatusCode{
			Code:  code,
			Count: num,
		})
	}
}

func (r *Report) printErrors() {
	for err, num := range r.errorDist {
		r.Errors = append(r.Errors, Error{
			Error: err,
			Count: num,
		})
	}
}
