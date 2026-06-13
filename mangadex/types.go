// Package mangadex is the library behind the mangadex command line:
// the HTTP client, request shaping, and the typed data models for MangaDex.
package mangadex

import "strings"

// Manga is the record emitted for search, top, recent, and manga-detail commands.
type Manga struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	AltTitle      string   `json:"alt_title"`
	Description   string   `json:"description"`
	Status        string   `json:"status"`
	Year          int      `json:"year"`
	ContentRating string   `json:"content_rating"`
	Authors       []string `json:"authors"`
	Tags          []string `json:"tags"`
	Languages     []string `json:"languages"`
	CoverURL      string   `json:"cover_url"`
	UpdatedAt     string   `json:"updated_at"`
}

// Chapter is the record emitted for the chapters command.
type Chapter struct {
	ID          string `json:"id"`
	MangaID     string `json:"manga_id"`
	Title       string `json:"title"`
	Chapter     string `json:"chapter"`
	Volume      string `json:"volume"`
	Pages       int    `json:"pages"`
	Language    string `json:"language"`
	PublishedAt string `json:"published_at"`
	Group       string `json:"group"`
}

// ─── wire types (unexported, JSON decode only) ─────────────────────────────

type wireMangaResp struct {
	Result   string         `json:"result"`
	Response string         `json:"response"`
	Data     []wireMangaObj `json:"data"`
	Total    int            `json:"total"`
	Offset   int            `json:"offset"`
	Limit    int            `json:"limit"`
}

type wireMangaEntityResp struct {
	Result   string       `json:"result"`
	Response string       `json:"response"`
	Data     wireMangaObj `json:"data"`
}

type wireChapterResp struct {
	Result string           `json:"result"`
	Data   []wireChapterObj `json:"data"`
	Total  int              `json:"total"`
	Offset int              `json:"offset"`
	Limit  int              `json:"limit"`
}

type wireMangaObj struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	Attributes    wireMangaAttrs `json:"attributes"`
	Relationships []wireRelation `json:"relationships"`
}

type wireMangaAttrs struct {
	Title                        map[string]string   `json:"title"`
	AltTitles                    []map[string]string `json:"altTitles"`
	Description                  map[string]string   `json:"description"`
	Status                       string              `json:"status"`
	Year                         *int                `json:"year"`
	ContentRating                string              `json:"contentRating"`
	Tags                         []wireTag           `json:"tags"`
	AvailableTranslatedLanguages []*string           `json:"availableTranslatedLanguages"`
	UpdatedAt                    string              `json:"updatedAt"`
}

type wireTag struct {
	ID         string       `json:"id"`
	Type       string       `json:"type"`
	Attributes wireTagAttrs `json:"attributes"`
}

type wireTagAttrs struct {
	Name  map[string]string `json:"name"`
	Group string            `json:"group"`
}

type wireRelation struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Attributes wireRelationAttrs `json:"attributes"`
}

type wireRelationAttrs struct {
	Name     string `json:"name"`
	FileName string `json:"fileName"`
}

type wireChapterObj struct {
	ID            string           `json:"id"`
	Type          string           `json:"type"`
	Attributes    wireChapterAttrs `json:"attributes"`
	Relationships []wireRelation   `json:"relationships"`
}

type wireChapterAttrs struct {
	Title              *string `json:"title"`
	Chapter            *string `json:"chapter"`
	Volume             *string `json:"volume"`
	Pages              int     `json:"pages"`
	TranslatedLanguage string  `json:"translatedLanguage"`
	PublishAt          string  `json:"publishAt"`
	ReadableAt         string  `json:"readableAt"`
	CreatedAt          string  `json:"createdAt"`
	UpdatedAt          string  `json:"updatedAt"`
}

// ─── mapping helpers ────────────────────────────────────────────────────────

// extractTitle picks "en" from the map, falling back to the first available value.
func extractTitle(m map[string]string) string {
	if v, ok := m["en"]; ok && v != "" {
		return v
	}
	for _, v := range m {
		if v != "" {
			return v
		}
	}
	return ""
}

// extractAltTitle returns the first English alt title, or first available value.
func extractAltTitle(alts []map[string]string) string {
	for _, m := range alts {
		if v, ok := m["en"]; ok && v != "" {
			return v
		}
	}
	for _, m := range alts {
		for _, v := range m {
			if v != "" {
				return v
			}
		}
	}
	return ""
}

// buildCoverURL constructs the thumbnail cover URL from manga ID and fileName.
func buildCoverURL(mangaID, fileName string) string {
	if fileName == "" {
		return ""
	}
	return "https://uploads.mangadex.org/covers/" + mangaID + "/" + fileName + ".256.jpg"
}

func wireMangaObjToManga(w wireMangaObj) Manga {
	a := w.Attributes

	// authors and cover from relationships
	var authors []string
	var coverFileName string
	for _, rel := range w.Relationships {
		switch rel.Type {
		case "author":
			if rel.Attributes.Name != "" {
				authors = append(authors, rel.Attributes.Name)
			}
		case "cover_art":
			if coverFileName == "" {
				coverFileName = rel.Attributes.FileName
			}
		}
	}

	// tags
	var tags []string
	for _, t := range a.Tags {
		name := extractTitle(t.Attributes.Name)
		if name != "" {
			tags = append(tags, name)
		}
	}

	// languages (filter nil entries)
	var langs []string
	for _, lp := range a.AvailableTranslatedLanguages {
		if lp != nil && *lp != "" {
			langs = append(langs, *lp)
		}
	}

	year := 0
	if a.Year != nil {
		year = *a.Year
	}

	desc := extractTitle(a.Description)
	// collapse newlines in description for table display
	desc = strings.ReplaceAll(desc, "\n", " ")

	return Manga{
		ID:            w.ID,
		Title:         extractTitle(a.Title),
		AltTitle:      extractAltTitle(a.AltTitles),
		Description:   desc,
		Status:        a.Status,
		Year:          year,
		ContentRating: a.ContentRating,
		Authors:       authors,
		Tags:          tags,
		Languages:     langs,
		CoverURL:      buildCoverURL(w.ID, coverFileName),
		UpdatedAt:     a.UpdatedAt,
	}
}

func wireChapterObjToChapter(w wireChapterObj, mangaUUID string) Chapter {
	deref := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}

	var group string
	for _, rel := range w.Relationships {
		if rel.Type == "scanlation_group" && rel.Attributes.Name != "" {
			group = rel.Attributes.Name
			break
		}
	}

	return Chapter{
		ID:          w.ID,
		MangaID:     mangaUUID,
		Title:       deref(w.Attributes.Title),
		Chapter:     deref(w.Attributes.Chapter),
		Volume:      deref(w.Attributes.Volume),
		Pages:       w.Attributes.Pages,
		Language:    w.Attributes.TranslatedLanguage,
		PublishedAt: w.Attributes.PublishAt,
		Group:       group,
	}
}
