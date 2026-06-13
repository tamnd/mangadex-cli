// Package mangadex is the library behind the mangadex command line:
// the HTTP client, request shaping, and the typed data models for MangaDex.
//
// The MangaDex API at api.mangadex.org is open and requires no authentication
// for read-only access. The client paces requests and retries 429 and 5xx.
package mangadex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ErrNotFound is returned when the API returns a missing or empty result.
var ErrNotFound = errors.New("not found")

// DefaultUserAgent identifies the client to MangaDex.
const DefaultUserAgent = "mangadex-cli/0.1 (+https://github.com/tamnd/mangadex-cli)"

// Config holds constructor parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns production-safe defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://api.mangadex.org",
		UserAgent: DefaultUserAgent,
		Rate:      300 * time.Millisecond,
		Retries:   3,
		Timeout:   15 * time.Second,
	}
}

// Client talks to the MangaDex API over HTTP.
type Client struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
	rate       time.Duration
	retries    int
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		userAgent:  cfg.UserAgent,
		rate:       cfg.Rate,
		retries:    cfg.Retries,
	}
}

// ─── HTTP internals ──────────────────────────────────────────────────────────

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, false, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

// ─── API methods ─────────────────────────────────────────────────────────────

// Search returns manga matching the query string.
func (c *Client) Search(ctx context.Context, q string, limit int) ([]Manga, error) {
	if limit <= 0 {
		limit = 20
	}
	params := url.Values{}
	params.Set("title", q)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", "0")
	params.Set("order[relevance]", "desc")
	params.Add("includes[]", "author")
	params.Add("includes[]", "cover_art")
	rawURL := c.baseURL + "/manga?" + params.Encode()

	var resp wireMangaResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, ErrNotFound
	}
	out := make([]Manga, len(resp.Data))
	for i, w := range resp.Data {
		out[i] = wireMangaObjToManga(w)
	}
	return out, nil
}

// Top returns top-rated manga.
func (c *Client) Top(ctx context.Context, limit int) ([]Manga, error) {
	if limit <= 0 {
		limit = 20
	}
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("order[rating]", "desc")
	params.Add("includes[]", "author")
	params.Add("includes[]", "cover_art")
	rawURL := c.baseURL + "/manga?" + params.Encode()

	var resp wireMangaResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, ErrNotFound
	}
	out := make([]Manga, len(resp.Data))
	for i, w := range resp.Data {
		out[i] = wireMangaObjToManga(w)
	}
	return out, nil
}

// Recent returns recently updated manga.
func (c *Client) Recent(ctx context.Context, limit int) ([]Manga, error) {
	if limit <= 0 {
		limit = 20
	}
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("order[updatedAt]", "desc")
	params.Add("includes[]", "author")
	params.Add("includes[]", "cover_art")
	rawURL := c.baseURL + "/manga?" + params.Encode()

	var resp wireMangaResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, ErrNotFound
	}
	out := make([]Manga, len(resp.Data))
	for i, w := range resp.Data {
		out[i] = wireMangaObjToManga(w)
	}
	return out, nil
}

// GetManga returns full metadata for one manga by UUID.
func (c *Client) GetManga(ctx context.Context, mangaUUID string) (Manga, error) {
	params := url.Values{}
	params.Add("includes[]", "author")
	params.Add("includes[]", "cover_art")
	rawURL := c.baseURL + "/manga/" + url.PathEscape(mangaUUID) + "?" + params.Encode()

	var resp wireMangaEntityResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return Manga{}, err
	}
	if resp.Data.ID == "" {
		return Manga{}, ErrNotFound
	}
	return wireMangaObjToManga(resp.Data), nil
}

// Chapters returns the chapter feed for a manga. lang="" fetches all languages.
func (c *Client) Chapters(ctx context.Context, mangaUUID string, lang string, limit int) ([]Chapter, error) {
	if limit <= 0 {
		limit = 50
	}
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", "0")
	params.Set("order[chapter]", "desc")
	params.Add("includes[]", "scanlation_group")
	if lang != "" {
		params.Add("translatedLanguage[]", lang)
	}
	rawURL := c.baseURL + "/manga/" + url.PathEscape(mangaUUID) + "/feed?" + params.Encode()

	var resp wireChapterResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, ErrNotFound
	}
	out := make([]Chapter, len(resp.Data))
	for i, w := range resp.Data {
		out[i] = wireChapterObjToChapter(w, mangaUUID)
	}
	return out, nil
}
