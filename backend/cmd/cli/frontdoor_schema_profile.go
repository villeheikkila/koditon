package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	frontdoorclient "koditon/internal/clients/frontdoor"
)

type frontdoorSchemaCache struct {
	dir string
}

func (c frontdoorSchemaCache) readSitemapEntries() ([]frontdoorclient.SitemapEntry, error) {
	path := filepath.Join(c.dir, "sitemap", "entries.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cached frontdoor sitemap entries %s: %w", path, err)
	}
	var cached []cachedFrontdoorSitemapEntry
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("decode cached frontdoor sitemap entries: %w", err)
	}
	entries := make([]frontdoorclient.SitemapEntry, 0, len(cached))
	for _, entry := range cached {
		parsed, err := url.Parse(entry.URL)
		if err != nil {
			return nil, fmt.Errorf("parse cached frontdoor sitemap url %q: %w", entry.URL, err)
		}
		entries = append(entries, frontdoorclient.SitemapEntry{ID: entry.ID, Type: frontdoorclient.EntryType(entry.Type), URL: parsed})
	}
	return entries, nil
}

func (c frontdoorSchemaCache) writeSitemapEntries(entries []frontdoorclient.SitemapEntry) error {
	cached := make([]cachedFrontdoorSitemapEntry, 0, len(entries))
	for _, entry := range entries {
		cached = append(cached, cachedFrontdoorSitemapEntry{ID: entry.ID, Type: string(entry.Type), URL: entry.URL.String()})
	}
	return writeJSONFile(filepath.Join(c.dir, "sitemap", "entries.json"), cached)
}

func (c frontdoorSchemaCache) readOrFetchAd(ctx context.Context, client *frontdoorclient.Client, friendlyID string, fetchMissing, refresh bool) ([]byte, bool, error) {
	path := filepath.Join(c.dir, "ad-raw", safeCacheName(friendlyID)+".json")
	if !refresh {
		if data, err := os.ReadFile(path); err == nil {
			return data, false, nil
		} else if !os.IsNotExist(err) {
			return nil, false, fmt.Errorf("read cached frontdoor ad %s: %w", path, err)
		}
		legacyPath := filepath.Join(c.dir, "ad", safeCacheName(friendlyID)+".json")
		if data, err := os.ReadFile(legacyPath); err == nil {
			if writeErr := writeRawFile(path, data); writeErr != nil {
				return nil, false, writeErr
			}
			return data, false, nil
		} else if !os.IsNotExist(err) {
			return nil, false, fmt.Errorf("read cached legacy frontdoor ad %s: %w", legacyPath, err)
		}
	}
	if !fetchMissing && !refresh {
		return nil, false, fmt.Errorf("frontdoor ad fixture missing: %s", path)
	}
	data, err := client.GetAdRawByFriendlyID(ctx, friendlyID)
	if err != nil {
		return nil, false, err
	}
	if err := writeRawFile(path, data); err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func (c frontdoorSchemaCache) writeQuicktypeAd(dir, friendlyID string, data []byte) error {
	path := filepath.Join(dir, "ad-raw", safeCacheName(friendlyID)+".json")
	return writeRawFile(path, data)
}

func (c frontdoorSchemaCache) readOrFetchBuildingState(ctx context.Context, client *frontdoorclient.Client, pageURL string, fetchMissing, refresh bool) ([]byte, bool, error) {
	key := sha256Hex(pageURL)
	statePath := filepath.Join(c.dir, "building-state", key+".json")
	htmlPath := filepath.Join(c.dir, "building-page", key+".html")
	if !refresh {
		if data, err := os.ReadFile(statePath); err == nil {
			return data, false, nil
		} else if !os.IsNotExist(err) {
			return nil, false, fmt.Errorf("read cached frontdoor building state %s: %w", statePath, err)
		}
	}
	var html []byte
	fetched := false
	if !refresh {
		if data, err := os.ReadFile(htmlPath); err == nil {
			html = data
		} else if !os.IsNotExist(err) {
			return nil, false, fmt.Errorf("read cached frontdoor building page %s: %w", htmlPath, err)
		}
	}
	if len(html) == 0 {
		if !fetchMissing && !refresh {
			return nil, false, fmt.Errorf("frontdoor building fixture missing: %s", statePath)
		}
		data, err := client.GetBuildingPageRaw(ctx, pageURL)
		if err != nil {
			return nil, false, err
		}
		if err := writeRawFile(htmlPath, data); err != nil {
			return nil, false, err
		}
		html = data
		fetched = true
	}
	state, err := frontdoorclient.ExtractInitialState(html)
	if err != nil {
		return nil, fetched, err
	}
	if err := writeRawFile(statePath, state); err != nil {
		return nil, fetched, err
	}
	return state, fetched, nil
}

type cachedFrontdoorSitemapEntry struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

type frontdoorSchemaProfileResult struct {
	CacheDir         string        `json:"cacheDir"`
	AdSource         string        `json:"adSource"`
	Processed        int           `json:"processed"`
	Fetched          int           `json:"fetched"`
	ErrorCount       int           `json:"errorCount,omitempty"`
	Errors           []string      `json:"errors,omitempty"`
	CandidateIDs     int           `json:"candidateIds,omitempty"`
	QuicktypeSamples int           `json:"quicktypeSamples,omitempty"`
	AdSamples        int           `json:"adSamples"`
	BuildingSamples  int           `json:"buildingSamples"`
	Ads              schemaProfile `json:"ads"`
	BuildingStates   schemaProfile `json:"buildingStates"`
}

func newFrontdoorSchemaProfileResult(cacheDir string, includeExamples bool) *frontdoorSchemaProfileResult {
	return &frontdoorSchemaProfileResult{
		CacheDir:       cacheDir,
		AdSource:       "sitemap",
		Ads:            newSchemaProfile(includeExamples),
		BuildingStates: newSchemaProfile(includeExamples),
	}
}

func (r *frontdoorSchemaProfileResult) recordError(err error) {
	r.ErrorCount++
	if len(r.Errors) >= 10 {
		return
	}
	message := strings.TrimSpace(err.Error())
	if len(message) > 240 {
		message = message[:240]
	}
	r.Errors = append(r.Errors, message)
}

type schemaProfile struct {
	Samples         int                   `json:"samples"`
	IncludeExamples bool                  `json:"includeExamples"`
	Paths           map[string]*pathStats `json:"paths"`
}

type pathStats struct {
	Occurrences    int            `json:"occurrences"`
	Nulls          int            `json:"nulls,omitempty"`
	Types          map[string]int `json:"types"`
	Examples       []string       `json:"examples,omitempty"`
	MinArrayLength *int           `json:"minArrayLength,omitempty"`
	MaxArrayLength *int           `json:"maxArrayLength,omitempty"`
}

func newSchemaProfile(includeExamples bool) schemaProfile {
	return schemaProfile{IncludeExamples: includeExamples, Paths: make(map[string]*pathStats)}
}

func (p *schemaProfile) ProfileJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	p.Samples++
	p.profileValue("$", value)
	return nil
}

func (p *schemaProfile) Finalize() {
	for _, stats := range p.Paths {
		if stats.Types == nil {
			stats.Types = map[string]int{}
		}
	}
}

func (p *schemaProfile) profileValue(path string, value any) {
	stats := p.stats(path)
	stats.Occurrences++
	switch typed := value.(type) {
	case nil:
		stats.Nulls++
		stats.addType("null")
	case bool:
		stats.addType("bool")
		p.addExample(stats, strconv.FormatBool(typed))
	case string:
		stats.addType("string")
		p.addExample(stats, typed)
	case json.Number:
		stats.addType(numberKind(typed))
		p.addExample(stats, typed.String())
	case []any:
		stats.addType("array")
		stats.observeArrayLength(len(typed))
		for _, item := range typed {
			p.profileValue(path+"[]", item)
		}
	case map[string]any:
		stats.addType("object")
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			p.profileValue(path+"."+schemaPathKey(key), typed[key])
		}
	}
}

func (p *schemaProfile) stats(path string) *pathStats {
	stats := p.Paths[path]
	if stats == nil {
		stats = &pathStats{Types: make(map[string]int)}
		p.Paths[path] = stats
	}
	return stats
}

func (p *schemaProfile) addExample(stats *pathStats, value string) {
	if p.IncludeExamples {
		stats.addExample(value)
	}
}

func (s *pathStats) addType(kind string) {
	s.Types[kind]++
}

func (s *pathStats) addExample(value string) {
	value = strings.TrimSpace(value)
	if value == "" || len(s.Examples) >= 3 {
		return
	}
	if len(value) > 120 {
		value = value[:120]
	}
	for _, existing := range s.Examples {
		if existing == value {
			return
		}
	}
	s.Examples = append(s.Examples, value)
}

func (s *pathStats) observeArrayLength(length int) {
	if s.MinArrayLength == nil || length < *s.MinArrayLength {
		value := length
		s.MinArrayLength = &value
	}
	if s.MaxArrayLength == nil || length > *s.MaxArrayLength {
		value := length
		s.MaxArrayLength = &value
	}
}

func numberKind(value json.Number) string {
	if _, err := value.Int64(); err == nil {
		return "int"
	}
	return "float"
}

func schemaPathKey(key string) string {
	if key == "" {
		return "{}"
	}
	for _, r := range key {
		if r < '0' || r > '9' {
			return key
		}
	}
	return "{numberKey}"
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	data = append(data, '\n')
	return writeRawFile(path, data)
}

func writeRawFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create cache dir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write cache file %s: %w", path, err)
	}
	return nil
}

func safeCacheName(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "?", "_", "#", "_", "&", "_", "=", "_")
	return replacer.Replace(value)
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func sleepContext(ctx context.Context, delay time.Duration) {
	if delay <= 0 {
		return
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
