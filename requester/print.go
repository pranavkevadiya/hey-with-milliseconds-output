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

/*
Hey supports two output formats: summary and CSV

The summary output presents a number of statistics about the requests in a
human-readable format, including:
- general statistics: requests/second, total runtime, and average, fastest, and slowest requests.
- a response time histogram.
- a percentile latency distribution.
- statistics (average, fastest, slowest) on the stages of the requests.

The comma-separated CSV format is proceeded by a header, and consists of the following columns:
1. response-time:	Total time taken for request (in milliseconds)
2. DNS+dialup:		Time taken to establish the TCP connection (in milliseconds)
3. DNS:				Time taken to do the DNS lookup (in milliseconds)
4. Request-write:	Time taken to write full request (in milliseconds)
5. Response-delay: 	Time taken to first byte received (in milliseconds)
6. Response-read:	Time taken to read full response (in milliseconds)
7. status-code:		HTTP status code of the response (e.g. 200)
8. offset:			The time since the start of the benchmark when the request was started. (in milliseconds)
*/
package requester

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
)

func newTemplate(output string) *template.Template {
	outputTmpl := output
	switch outputTmpl {
	case "":
		outputTmpl = defaultTmpl
	case "csv":
		outputTmpl = csvTmpl
	}
	return template.Must(template.New("tmpl").Funcs(tmplFuncMap).Parse(outputTmpl))
}

var tmplFuncMap = template.FuncMap{
	"formatNumber":            formatNumber,
	"formatNumberToMillis":    formatNumberToMillis,
	"formatNumberInt":         formatNumberInt,
	"formatNumberIntToMillis": formatNumberIntToMillis,
	"histogram":               histogram,
	"jsonify":                 jsonify,
}

func jsonify(v interface{}) string {
	d, _ := json.Marshal(v)
	return string(d)
}

func formatNumber(duration float64) string {
	return fmt.Sprintf("%4.4f", duration)
}

func formatNumberToMillis(duration float64) string {
	return fmt.Sprintf("%4.0f", duration*1000)
}

func formatNumberInt(duration int) string {
	return fmt.Sprintf("%d", duration)
}

func formatNumberIntToMillis(duration int) string {
	return fmt.Sprintf("%d", duration*1000)
}

func histogram(buckets []Bucket) string {
	max := 0
	for _, b := range buckets {
		if v := b.Count; v > max {
			max = v
		}
	}
	res := new(bytes.Buffer)
	for i := 0; i < len(buckets); i++ {
		// Normalize bar lengths.
		var barLen int
		if max > 0 {
			barLen = (buckets[i].Count*40 + max/2) / max
		}
		res.WriteString(fmt.Sprintf("  %4.3f [%v]\t|%v\n", buckets[i].Mark, buckets[i].Count, strings.Repeat(barChar, barLen)))
	}
	return res.String()
}

var (
	defaultTmpl = `
Summary:
  Total:	{{ formatNumberToMillis .Total.Seconds }} millis
  Slowest:	{{ formatNumberToMillis .Slowest }} millis
  Fastest:	{{ formatNumberToMillis .Fastest }} millis
  Average:	{{ formatNumberToMillis .Average }} millis
  Requests/sec:	{{ formatNumber .Rps }}
  {{ if gt .SizeTotal 0 }}
  Total data:	{{ .SizeTotal }} bytes
  Size/request:	{{ .SizeReq }} bytes{{ end }}

Response time histogram:
{{ histogram .Histogram }}

Latency distribution:{{ range .LatencyDistribution }}
  {{ .Percentage }}%% in {{ formatNumberToMillis .Latency }} millis{{ end }}

Details (average, fastest, slowest):
  DNS+dialup:	{{ formatNumberToMillis .AvgConn }} millis, {{ formatNumberToMillis .ConnMax }} millis, {{ formatNumberToMillis .ConnMin }} millis
  DNS-lookup:	{{ formatNumberToMillis .AvgDNS }} millis, {{ formatNumberToMillis .DnsMax }} millis, {{ formatNumberToMillis .DnsMin }} millis
  req write:	{{ formatNumberToMillis .AvgReq }} millis, {{ formatNumberToMillis .ReqMax }} millis, {{ formatNumberToMillis .ReqMin }} millis
  resp wait:	{{ formatNumberToMillis .AvgDelay }} millis, {{ formatNumberToMillis .DelayMax }} millis, {{ formatNumberToMillis .DelayMin }} millis
  resp read:	{{ formatNumberToMillis .AvgRes }} millis, {{ formatNumberToMillis .ResMax }} millis, {{ formatNumberToMillis .ResMin }} millis

Status code distribution:{{ range $code, $num := .StatusCodeDist }}
  [{{ $code }}]	{{ $num }} responses{{ end }}

{{ if gt (len .ErrorDist) 0 }}Error distribution:{{ range $err, $num := .ErrorDist }}
  [{{ $num }}]	{{ $err }}{{ end }}{{ end }}
`
	csvTmpl = `{{ $connLats := .ConnLats }}{{ $dnsLats := .DnsLats }}{{ $dnsLats := .DnsLats }}{{ $reqLats := .ReqLats }}{{ $delayLats := .DelayLats }}{{ $resLats := .ResLats }}{{ $statusCodeLats := .StatusCodes }}{{ $offsets := .Offsets}}response-time,DNS+dialup,DNS,Request-write,Response-delay,Response-read,status-code,offset{{ range $i, $v := .Lats }}
{{ formatNumberToMillis $v }},{{ formatNumberToMillis (index $connLats $i) }},{{ formatNumberToMillis (index $dnsLats $i) }},{{ formatNumberToMillis (index $reqLats $i) }},{{ formatNumberToMillis (index $delayLats $i) }},{{ formatNumberToMillis (index $resLats $i) }},{{ formatNumberInt (index $statusCodeLats $i) }},{{ formatNumberToMillis (index $offsets $i) }}{{ end }}`
)
