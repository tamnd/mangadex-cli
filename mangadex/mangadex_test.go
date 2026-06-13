package mangadex_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tamnd/mangadex-cli/mangadex"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*mangadex.Client, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	cfg := mangadex.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 1
	cfg.Timeout = 5 * time.Second
	return mangadex.NewClient(cfg), ts
}

const mangaCollectionJSON = `{
  "result": "ok",
  "response": "collection",
  "data": [
    {
      "id": "manga-uuid-1",
      "type": "manga",
      "attributes": {
        "title": {"en": "One Piece"},
        "altTitles": [{"ja": "ワンピース"}],
        "description": {"en": "A pirate adventure."},
        "status": "ongoing",
        "year": 1997,
        "contentRating": "safe",
        "tags": [
          {"id": "tag-1","type":"tag","attributes":{"name":{"en":"Action"},"group":"genre"}}
        ],
        "availableTranslatedLanguages": ["en", "ja"],
        "updatedAt": "2024-01-15T10:00:00Z"
      },
      "relationships": [
        {"id": "author-1", "type": "author", "attributes": {"name": "Oda Eiichiro", "fileName": ""}},
        {"id": "cover-1", "type": "cover_art", "attributes": {"name": "", "fileName": "cover.jpg"}}
      ]
    }
  ],
  "total": 1,
  "offset": 0,
  "limit": 20
}`

const mangaEntityJSON = `{
  "result": "ok",
  "response": "entity",
  "data": {
    "id": "manga-uuid-1",
    "type": "manga",
    "attributes": {
      "title": {"en": "One Piece"},
      "altTitles": [],
      "description": {"en": "A pirate adventure."},
      "status": "ongoing",
      "year": 1997,
      "contentRating": "safe",
      "tags": [],
      "availableTranslatedLanguages": ["en"],
      "updatedAt": "2024-01-15T10:00:00Z"
    },
    "relationships": [
      {"id": "author-1", "type": "author", "attributes": {"name": "Oda Eiichiro", "fileName": ""}},
      {"id": "cover-1", "type": "cover_art", "attributes": {"name": "", "fileName": "cover.jpg"}}
    ]
  }
}`

const chapterFeedJSON = `{
  "result": "ok",
  "data": [
    {
      "id": "chap-uuid-1",
      "type": "chapter",
      "attributes": {
        "title": "Romance Dawn",
        "chapter": "1",
        "volume": "1",
        "pages": 53,
        "translatedLanguage": "en",
        "publishAt": "2020-06-01T00:00:00Z",
        "readableAt": "2020-06-01T00:00:00Z",
        "createdAt": "2020-06-01T00:00:00Z",
        "updatedAt": "2020-06-01T00:00:00Z"
      },
      "relationships": [
        {"id": "group-1", "type": "scanlation_group", "attributes": {"name": "MangaPlus", "fileName": ""}}
      ]
    }
  ],
  "total": 1,
  "offset": 0,
  "limit": 50
}`

func TestSearch(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/manga" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mangaCollectionJSON))
	})

	results, err := c.Search(context.Background(), "one piece", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	m := results[0]
	if m.ID != "manga-uuid-1" {
		t.Errorf("ID = %q, want manga-uuid-1", m.ID)
	}
	if m.Title != "One Piece" {
		t.Errorf("Title = %q, want One Piece", m.Title)
	}
	if m.Status != "ongoing" {
		t.Errorf("Status = %q, want ongoing", m.Status)
	}
	if m.Year != 1997 {
		t.Errorf("Year = %d, want 1997", m.Year)
	}
	if len(m.Authors) == 0 || m.Authors[0] != "Oda Eiichiro" {
		t.Errorf("Authors = %v, want [Oda Eiichiro]", m.Authors)
	}
	wantCover := "https://uploads.mangadex.org/covers/manga-uuid-1/cover.jpg.256.jpg"
	if m.CoverURL != wantCover {
		t.Errorf("CoverURL = %q, want %q", m.CoverURL, wantCover)
	}
	if len(m.Tags) == 0 || m.Tags[0] != "Action" {
		t.Errorf("Tags = %v, want [Action]", m.Tags)
	}
}

func TestTop(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("order[rating]") != "desc" {
			t.Errorf("expected order[rating]=desc, got %q", r.URL.Query().Get("order[rating]"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mangaCollectionJSON))
	})

	results, err := c.Top(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Title != "One Piece" {
		t.Errorf("Title = %q, want One Piece", results[0].Title)
	}
}

func TestRecent(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("order[updatedAt]") != "desc" {
			t.Errorf("expected order[updatedAt]=desc, got %q", r.URL.Query().Get("order[updatedAt]"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mangaCollectionJSON))
	})

	results, err := c.Recent(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

func TestGetManga(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/manga/manga-uuid-1" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mangaEntityJSON))
	})

	m, err := c.GetManga(context.Background(), "manga-uuid-1")
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "manga-uuid-1" {
		t.Errorf("ID = %q, want manga-uuid-1", m.ID)
	}
	if m.Title != "One Piece" {
		t.Errorf("Title = %q, want One Piece", m.Title)
	}
	wantCover := "https://uploads.mangadex.org/covers/manga-uuid-1/cover.jpg.256.jpg"
	if m.CoverURL != wantCover {
		t.Errorf("CoverURL = %q, want %q", m.CoverURL, wantCover)
	}
}

func TestChapters(t *testing.T) {
	const mangaID = "manga-uuid-1"
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/manga/"+mangaID+"/feed" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(chapterFeedJSON))
	})

	chapters, err := c.Chapters(context.Background(), mangaID, "en", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(chapters) != 1 {
		t.Fatalf("got %d chapters, want 1", len(chapters))
	}
	ch := chapters[0]
	if ch.ID != "chap-uuid-1" {
		t.Errorf("ID = %q, want chap-uuid-1", ch.ID)
	}
	if ch.Chapter != "1" {
		t.Errorf("Chapter = %q, want 1", ch.Chapter)
	}
	if ch.MangaID != mangaID {
		t.Errorf("MangaID = %q, want %s", ch.MangaID, mangaID)
	}
	if ch.Group != "MangaPlus" {
		t.Errorf("Group = %q, want MangaPlus", ch.Group)
	}
	if ch.Pages != 53 {
		t.Errorf("Pages = %d, want 53", ch.Pages)
	}
}

func TestNotFound(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := c.GetManga(context.Background(), "nonexistent-uuid")
	if !errors.Is(err, mangadex.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestRetry(t *testing.T) {
	var hits int
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mangaCollectionJSON))
	})
	// Override retries to allow second attempt
	cfg := mangadex.DefaultConfig()
	cfg.Retries = 3
	cfg.Rate = 0

	results, err := c.Search(context.Background(), "one piece", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results after retry, want 1", len(results))
	}
}
