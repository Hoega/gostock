package quote

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// InflationYear holds inflation data for a single year.
type InflationYear struct {
	Year           int
	Rate           float64 // Annual inflation rate (%)
	CumulativeRate float64 // Cumulative rate since first year (multiplier, e.g. 1.05)
}

var inflationCache struct {
	mu        sync.Mutex
	data      []InflationYear
	fetchedAt time.Time
}

const inflationCacheTTL = 24 * time.Hour

// StructureSpecificData XML types for INSEE BDM SDMX response.
type sdmxStructure struct {
	XMLName xml.Name    `xml:"StructureSpecificData"`
	DataSet sdmxDataSet `xml:"DataSet"`
}

type sdmxDataSet struct {
	Series sdmxSeries `xml:"Series"`
}

type sdmxSeries struct {
	Obs []sdmxObs `xml:"Obs"`
}

type sdmxObs struct {
	TimePeriod string `xml:"TIME_PERIOD,attr"`
	ObsValue   string `xml:"OBS_VALUE,attr"`
}

// FetchInflationData fetches French CPI data from INSEE and computes annual inflation rates.
// Returns data for years [startYear, endYear].
func FetchInflationData(startYear, endYear int) ([]InflationYear, error) {
	inflationCache.mu.Lock()
	if inflationCache.data != nil && time.Since(inflationCache.fetchedAt) < inflationCacheTTL {
		cached := filterInflationYears(inflationCache.data, startYear, endYear)
		inflationCache.mu.Unlock()
		if len(cached) > 0 {
			return cached, nil
		}
	}
	inflationCache.mu.Unlock()

	// Fetch one extra year before startYear to compute the rate for startYear
	fetchStart := startYear - 1
	url := fmt.Sprintf(
		"https://api.insee.fr/series/BDM/V1/data/SERIES_BDM/001759970?startPeriod=%d-01&endPeriod=%d-12",
		fetchStart, endYear,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("insee: create request: %w", err)
	}
	req.Header.Set("Accept", "application/xml")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("insee: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("insee: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("insee: read body: %w", err)
	}

	var data sdmxStructure
	if err := xml.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("insee: parse XML: %w", err)
	}

	// Extract December IPC values by year
	decemberIPC := make(map[int]float64)
	for _, obs := range data.DataSet.Series.Obs {
		period := obs.TimePeriod // format: "2021-12"
		if len(period) < 7 {
			continue
		}
		year, err := strconv.Atoi(period[:4])
		if err != nil {
			continue
		}
		month := period[5:]
		if month != "12" {
			continue
		}
		val, err := strconv.ParseFloat(obs.ObsValue, 64)
		if err != nil {
			continue
		}
		decemberIPC[year] = val
	}

	// Compute annual rates and cumulative multiplier
	var results []InflationYear
	baseIPC := decemberIPC[startYear-1]
	if baseIPC == 0 {
		// If we can't get the base year, use the earliest available December value
		for y := fetchStart; y <= endYear; y++ {
			if v, ok := decemberIPC[y]; ok {
				baseIPC = v
				break
			}
		}
	}

	firstIPC := baseIPC // IPC of the year before startYear, for cumulative calculation
	for y := startYear; y <= endYear; y++ {
		ipcCurrent, ok := decemberIPC[y]
		if !ok {
			continue
		}
		ipcPrev, ok := decemberIPC[y-1]
		if !ok {
			continue
		}

		annualRate := (ipcCurrent/ipcPrev - 1) * 100
		cumulativeRate := 1.0
		if firstIPC > 0 {
			cumulativeRate = ipcCurrent / firstIPC
		}

		results = append(results, InflationYear{
			Year:           y,
			Rate:           math.Round(annualRate*100) / 100,
			CumulativeRate: cumulativeRate,
		})
	}

	// Update cache
	inflationCache.mu.Lock()
	inflationCache.data = results
	inflationCache.fetchedAt = time.Now()
	inflationCache.mu.Unlock()

	return filterInflationYears(results, startYear, endYear), nil
}

func filterInflationYears(data []InflationYear, startYear, endYear int) []InflationYear {
	var filtered []InflationYear
	for _, d := range data {
		if d.Year >= startYear && d.Year <= endYear {
			filtered = append(filtered, d)
		}
	}
	return filtered
}
