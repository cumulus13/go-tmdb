package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	baseURL   = "https://api.themoviedb.org/3"
	imageBase = "https://image.tmdb.org/t/p"
)

// Image sizes available on TMDb
var posterSizes   = []string{"w92", "w154", "w185", "w342", "w500", "w780", "original"}
var backdropSizes = []string{"w300", "w780", "w1280", "original"}
var profileSizes  = []string{"w45", "w185", "h632", "original"}
var logoSizes     = []string{"w45", "w92", "w154", "w185", "w300", "w500", "original"}

// ── ANSI colors ──────────────────────────────────────────────────────────────

const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	cyan    = "\033[36;1m"
	yellow  = "\033[33m"
	green   = "\033[32m"
	magenta = "\033[35m"
	red     = "\033[31m"
	blue    = "\033[34;1m"
	white   = "\033[37;1m"
)

func c(code, s string) string { return code + s + reset }

// ── API structs ──────────────────────────────────────────────────────────────

type SearchResult struct {
	TotalResults int          `json:"total_results"`
	Results      []SearchItem `json:"results"`
}

type SearchItem struct {
	ID           int     `json:"id"`
	Title        string  `json:"title"`
	Name         string  `json:"name"`
	MediaType    string  `json:"media_type"`
	Overview     string  `json:"overview"`
	ReleaseDate  string  `json:"release_date"`
	FirstAirDate string  `json:"first_air_date"`
	VoteAverage  float64 `json:"vote_average"`
	VoteCount    int     `json:"vote_count"`
	Popularity   float64 `json:"popularity"`
}

type Genre   struct{ Name string `json:"name"` }
type Network struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Credits struct {
	Cast []CastMember `json:"cast"`
	Crew []CrewMember `json:"crew"`
}

type CastMember struct {
	Name      string `json:"name"`
	Character string `json:"character"`
	Order     int    `json:"order"`
	ProfilePath string `json:"profile_path"`
}

type CrewMember struct {
	Name       string `json:"name"`
	Job        string `json:"job"`
	Department string `json:"department"`
}

type ProductionCompany struct {
	Name          string `json:"name"`
	OriginCountry string `json:"origin_country"`
}

type ProductionCountry struct {
	Name string `json:"name"`
}

type SpokenLanguage struct {
	EnglishName string `json:"english_name"`
	Name        string `json:"name"`
}

type MovieDetail struct {
	ID                  int                 `json:"id"`
	Title               string              `json:"title"`
	OriginalTitle       string              `json:"original_title"`
	Tagline             string              `json:"tagline"`
	Overview            string              `json:"overview"`
	ReleaseDate         string              `json:"release_date"`
	Runtime             int                 `json:"runtime"`
	VoteAverage         float64             `json:"vote_average"`
	VoteCount           int                 `json:"vote_count"`
	Popularity          float64             `json:"popularity"`
	Status              string              `json:"status"`
	Budget              int64               `json:"budget"`
	Revenue             int64               `json:"revenue"`
	Homepage            string              `json:"homepage"`
	IMDbID              string              `json:"imdb_id"`
	OriginalLanguage    string              `json:"original_language"`
	PosterPath          string              `json:"poster_path"`
	BackdropPath        string              `json:"backdrop_path"`
	Adult               bool                `json:"adult"`
	Genres              []Genre             `json:"genres"`
	ProductionCompanies []ProductionCompany `json:"production_companies"`
	ProductionCountries []ProductionCountry `json:"production_countries"`
	SpokenLanguages     []SpokenLanguage    `json:"spoken_languages"`
	Credits             Credits             `json:"credits"`
	Images              ImageCollection     `json:"images"`
	Videos              VideoResults        `json:"videos"`
	Reviews             ReviewResults       `json:"reviews"`
	Similar             SearchResult        `json:"similar"`
	Keywords            struct {
		Keywords []Keyword `json:"keywords"`
	} `json:"keywords"`
	WatchProviders WatchProviderResponse `json:"watch/providers"`
}

type TVDetail struct {
	ID                  int                 `json:"id"`
	Name                string              `json:"name"`
	OriginalName        string              `json:"original_name"`
	Tagline             string              `json:"tagline"`
	Overview            string              `json:"overview"`
	FirstAirDate        string              `json:"first_air_date"`
	LastAirDate         string              `json:"last_air_date"`
	Status              string              `json:"status"`
	Type                string              `json:"type"`
	VoteAverage         float64             `json:"vote_average"`
	VoteCount           int                 `json:"vote_count"`
	Popularity          float64             `json:"popularity"`
	NumberOfSeasons     int                 `json:"number_of_seasons"`
	NumberOfEpisodes    int                 `json:"number_of_episodes"`
	EpisodeRunTime      []int               `json:"episode_run_time"`
	Homepage            string              `json:"homepage"`
	InProduction        bool                `json:"in_production"`
	OriginalLanguage    string              `json:"original_language"`
	PosterPath          string              `json:"poster_path"`
	BackdropPath        string              `json:"backdrop_path"`
	Genres              []Genre             `json:"genres"`
	Networks            []Network           `json:"networks"`
	ProductionCompanies []ProductionCompany `json:"production_companies"`
	Credits             Credits             `json:"credits"`
	Images              ImageCollection     `json:"images"`
	Videos              VideoResults        `json:"videos"`
	Reviews             ReviewResults       `json:"reviews"`
	Similar             SearchResult        `json:"similar"`
	Keywords            struct {
		Results []Keyword `json:"results"`
	} `json:"keywords"`
	WatchProviders WatchProviderResponse `json:"watch/providers"`
	Seasons         []SeasonSummary      `json:"seasons"`
}

type SeasonSummary struct {
	ID           int    `json:"id"`
	SeasonNumber int    `json:"season_number"`
	Name         string `json:"name"`
	AirDate      string `json:"air_date"`
	EpisodeCount int    `json:"episode_count"`
	Overview     string `json:"overview"`
	PosterPath   string `json:"poster_path"`
}

type SeasonDetail struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Overview     string    `json:"overview"`
	SeasonNumber int       `json:"season_number"`
	AirDate      string    `json:"air_date"`
	Episodes     []Episode `json:"episodes"`
	PosterPath   string    `json:"poster_path"`
}

type Episode struct {
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	Overview       string  `json:"overview"`
	EpisodeNumber  int     `json:"episode_number"`
	SeasonNumber   int     `json:"season_number"`
	AirDate        string  `json:"air_date"`
	Runtime        int     `json:"runtime"`
	VoteAverage    float64 `json:"vote_average"`
	VoteCount      int     `json:"vote_count"`
	StillPath      string  `json:"still_path"`
}

type PersonDetail struct {
	ID                 int     `json:"id"`
	Name               string  `json:"name"`
	Biography          string  `json:"biography"`
	Birthday           string  `json:"birthday"`
	Deathday           string  `json:"deathday"`
	PlaceOfBirth       string  `json:"place_of_birth"`
	KnownForDepartment string  `json:"known_for_department"`
	IMDbID             string  `json:"imdb_id"`
	Popularity         float64 `json:"popularity"`
	ProfilePath        string  `json:"profile_path"`
	Homepage           string  `json:"homepage"`
	AlsoKnownAs        []string `json:"also_known_as"`
	MovieCredits       struct {
		Cast []PersonCastCredit `json:"cast"`
		Crew []PersonCrewCredit `json:"crew"`
	} `json:"movie_credits"`
	TVCredits struct {
		Cast []PersonCastCredit `json:"cast"`
	} `json:"tv_credits"`
	Images struct {
		Profiles []ImageItem `json:"profiles"`
	} `json:"images"`
}

type PersonCastCredit struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Name        string  `json:"name"`
	Character   string  `json:"character"`
	ReleaseDate string  `json:"release_date"`
	FirstAirDate string `json:"first_air_date"`
	VoteAverage float64 `json:"vote_average"`
	MediaType   string  `json:"media_type"`
}

type PersonCrewCredit struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Job         string  `json:"job"`
	ReleaseDate string  `json:"release_date"`
	VoteAverage float64 `json:"vote_average"`
}

type ImageCollection struct {
	Posters   []ImageItem `json:"posters"`
	Backdrops []ImageItem `json:"backdrops"`
	Logos     []ImageItem `json:"logos"`
	Profiles  []ImageItem `json:"profiles"`
	Stills    []ImageItem `json:"stills"`
}

type ImageItem struct {
	FilePath    string  `json:"file_path"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	VoteAverage float64 `json:"vote_average"`
	VoteCount   int     `json:"vote_count"`
	ISO639_1    string  `json:"iso_639_1"`
	AspectRatio float64 `json:"aspect_ratio"`
}

type VideoResults struct {
	Results []Video `json:"results"`
}

type Video struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Key      string `json:"key"`
	Site     string `json:"site"`
	Size     int    `json:"size"`
	Type     string `json:"type"`
	Official bool   `json:"official"`
}

type ReviewResults struct {
	Results    []Review `json:"results"`
	TotalPages int      `json:"total_pages"`
	TotalResults int    `json:"total_results"`
}

type Review struct {
	ID          string      `json:"id"`
	Author      string      `json:"author"`
	Content     string      `json:"content"`
	CreatedAt   string      `json:"created_at"`
	AuthorDetails struct {
		Rating float64 `json:"rating"`
	} `json:"author_details"`
}

type Keyword struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type WatchProviderResponse struct {
	Results map[string]WatchRegion `json:"results"`
}

type WatchRegion struct {
	Link     string          `json:"link"`
	Flatrate []WatchProvider `json:"flatrate"`
	Rent     []WatchProvider `json:"rent"`
	Buy      []WatchProvider `json:"buy"`
	Free     []WatchProvider `json:"free"`
}

type WatchProvider struct {
	ProviderName string `json:"provider_name"`
	DisplayPriority int `json:"display_priority"`
}

// ── Export structs (clean output) ────────────────────────────────────────────

type ExportImage struct {
	Type        string `json:"type"`
	FilePath    string `json:"file_path"`
	URL         string `json:"url_original"`
	URLLarge    string `json:"url_large,omitempty"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Language    string `json:"language,omitempty"`
	VoteAverage float64 `json:"vote_average"`
	VoteCount   int    `json:"vote_count"`
}

type ExportVideo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Site     string `json:"site"`
	URL      string `json:"url"`
	Size     int    `json:"size"`
	Official bool   `json:"official"`
}

// ── HTTP helper ──────────────────────────────────────────────────────────────

func apiGet(apiKey, endpoint string, params map[string]string) ([]byte, error) {
	u, _ := url.Parse(baseURL + endpoint)
	q := u.Query()
	q.Set("api_key", apiKey)
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func imageURL(path, size string) string {
	if path == "" { return "" }
	return fmt.Sprintf("%s/%s%s", imageBase, size, path)
}

// ── Formatting helpers ───────────────────────────────────────────────────────

func printHeader(title string) {
	line := strings.Repeat("─", 64)
	fmt.Println()
	fmt.Println(c(cyan, line))
	fmt.Println(c(bold, "  "+title))
	fmt.Println(c(cyan, line))
}

func printSection(title string) {
	fmt.Printf("\n  %s\n", c(blue, "▸ "+title))
	fmt.Printf("  %s\n", c(dim, strings.Repeat("·", 48)))
}

func printField(label, value string) {
	if value == "" || value == "0" { return }
	fmt.Printf("  %s  %s\n", c(yellow, fmt.Sprintf("%-20s", label+":")), value)
}

func ratingBar(score float64) string {
	filled := int(score / 10.0 * 10)
	if filled > 10 { filled = 10 }
	bar := strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
	return fmt.Sprintf("[%s] %.1f/10", bar, score)
}

func formatMoney(n int64) string {
	if n == 0 { return "" }
	if n >= 1_000_000_000 { return fmt.Sprintf("$%.2fB", float64(n)/1e9) }
	return fmt.Sprintf("$%.2fM", float64(n)/1e6)
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max { return s }
	return s[:max-3] + "..."
}

func genreNames(genres []Genre) string {
	names := make([]string, len(genres))
	for i, g := range genres { names[i] = g.Name }
	return strings.Join(names, ", ")
}

func keywordNames(kws []Keyword) string {
	names := make([]string, len(kws))
	for i, k := range kws { names[i] = k.Name }
	return strings.Join(names, ", ")
}

func videoURL(v Video) string {
	switch v.Site {
	case "YouTube":  return "https://youtu.be/" + v.Key
	case "Vimeo":    return "https://vimeo.com/" + v.Key
	default:         return v.Key
	}
}

// ── Export / serialization ───────────────────────────────────────────────────

// Simple YAML encoder (no deps) — handles string/int/float/bool/slice/map
func toYAML(v interface{}, indent int) string {
	prefix := strings.Repeat("  ", indent)
	switch val := v.(type) {
	case nil:
		return "null"
	case bool:
		if val { return "true" }
		return "false"
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case string:
		if strings.ContainsAny(val, ":\n#{}[]|>&'\"") || strings.TrimSpace(val) != val {
			escaped := strings.ReplaceAll(val, "\\", "\\\\")
			escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
			escaped = strings.ReplaceAll(escaped, "\n", "\\n")
			return `"` + escaped + `"`
		}
		return val
	case []interface{}:
		if len(val) == 0 { return "[]" }
		var sb strings.Builder
		for _, item := range val {
			sb.WriteString("\n" + prefix + "- ")
			rendered := toYAML(item, indent+1)
			if strings.Contains(rendered, "\n") {
				sb.WriteString(rendered)
			} else {
				sb.WriteString(rendered)
			}
		}
		return sb.String()
	case map[string]interface{}:
		if len(val) == 0 { return "{}" }
		keys := make([]string, 0, len(val))
		for k := range val { keys = append(keys, k) }
		sort.Strings(keys)
		var sb strings.Builder
		for _, k := range keys {
			rendered := toYAML(val[k], indent+1)
			if strings.HasPrefix(rendered, "\n") {
				sb.WriteString("\n" + prefix + k + ":" + rendered)
			} else {
				sb.WriteString("\n" + prefix + k + ": " + rendered)
			}
		}
		return sb.String()
	}
	return fmt.Sprintf("%v", v)
}

func marshalYAML(v interface{}) ([]byte, error) {
	// round-trip through JSON to get map[string]interface{}
	b, err := json.Marshal(v)
	if err != nil { return nil, err }
	var generic interface{}
	if err := json.Unmarshal(b, &generic); err != nil { return nil, err }
	result := toYAML(generic, 0)
	if strings.HasPrefix(result, "\n") { result = result[1:] }
	return []byte(result + "\n"), nil
}

// Simple TOML encoder (no deps)
func toTOML(key string, v interface{}, sb *strings.Builder, tablePrefix string) {
	switch val := v.(type) {
	case nil:
		// skip
	case bool:
		if val { sb.WriteString(key + " = true\n") } else { sb.WriteString(key + " = false\n") }
	case float64:
		sb.WriteString(key + " = " + strconv.FormatFloat(val, 'f', -1, 64) + "\n")
	case string:
		escaped := strings.ReplaceAll(val, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		escaped = strings.ReplaceAll(escaped, "\n", "\\n")
		sb.WriteString(key + ` = "` + escaped + `"` + "\n")
	case int64:
		sb.WriteString(key + " = " + strconv.FormatInt(val, 10) + "\n")
	case []interface{}:
		// arrays of scalars inline, arrays of objects as array-of-tables
		if len(val) == 0 { sb.WriteString(key + " = []\n"); return }
		if _, isMap := val[0].(map[string]interface{}); isMap {
			for _, item := range val {
				sb.WriteString("\n[[" + tablePrefix + key + "]]\n")
				for k2, v2 := range item.(map[string]interface{}) {
					toTOML(k2, v2, sb, tablePrefix+key+".")
				}
			}
		} else {
			parts := make([]string, len(val))
			for i, item := range val {
				switch sv := item.(type) {
				case string:
					escaped := strings.ReplaceAll(sv, "\"", "\\\"")
					parts[i] = `"` + escaped + `"`
				case float64:
					parts[i] = strconv.FormatFloat(sv, 'f', -1, 64)
				default:
					parts[i] = fmt.Sprintf("%v", sv)
				}
			}
			sb.WriteString(key + " = [" + strings.Join(parts, ", ") + "]\n")
		}
	case map[string]interface{}:
		sb.WriteString("\n[" + tablePrefix + key + "]\n")
		keys := make([]string, 0, len(val))
		for k := range val { keys = append(keys, k) }
		sort.Strings(keys)
		for _, k := range keys {
			toTOML(k, val[k], sb, tablePrefix+key+".")
		}
	}
}

func marshalTOML(v interface{}) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil { return nil, err }
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil { return nil, err }

	var sb strings.Builder
	// scalars first, then tables
	keys := make([]string, 0, len(m))
	for k := range m { keys = append(keys, k) }
	sort.Strings(keys)
	for _, k := range keys {
		switch m[k].(type) {
		case map[string]interface{}, []interface{}:
			// defer
		default:
			toTOML(k, m[k], &sb, "")
		}
	}
	for _, k := range keys {
		switch m[k].(type) {
		case map[string]interface{}, []interface{}:
			toTOML(k, m[k], &sb, "")
		}
	}
	return []byte(sb.String()), nil
}

// Simple CSV for flat list of images
func marshalCSV(headers []string, rows [][]string) []byte {
	var sb strings.Builder
	sb.WriteString(strings.Join(headers, ",") + "\n")
	for _, row := range rows {
		escaped := make([]string, len(row))
		for i, f := range row {
			if strings.ContainsAny(f, ",\"\n") {
				f = `"` + strings.ReplaceAll(f, `"`, `""`) + `"`
			}
			escaped[i] = f
		}
		sb.WriteString(strings.Join(escaped, ",") + "\n")
	}
	return []byte(sb.String())
}

func exportData(data interface{}, format, filename string) error {
	var (
		out []byte
		err error
	)
	switch strings.ToLower(format) {
	case "json":
		out, err = json.MarshalIndent(data, "", "  ")
	case "yaml", "yml":
		out, err = marshalYAML(data)
	case "toml":
		out, err = marshalTOML(data)
	default:
		return fmt.Errorf("unknown format: %s (use json, yaml, toml)", format)
	}
	if err != nil { return err }
	if err := os.WriteFile(filename, out, 0644); err != nil { return err }
	fmt.Printf("\n  %s  Exported to %s\n", c(green, "✓"), c(bold, filename))
	return nil
}

// ── Parse common flags ───────────────────────────────────────────────────────

type CommonFlags struct {
	ExportFmt  string // json/yaml/toml
	ExportFile string
	Lang       string
	Region     string
}

func parseCommon(args []string) (rest []string, f CommonFlags) {
	f.Lang = "en-US"
	f.Region = "US"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--export", "-e":
			if i+1 < len(args) { i++; f.ExportFmt = args[i] }
		case "--out", "-o":
			if i+1 < len(args) { i++; f.ExportFile = args[i] }
		case "--lang":
			if i+1 < len(args) { i++; f.Lang = args[i] }
		case "--region":
			if i+1 < len(args) { i++; f.Region = strings.ToUpper(args[i]) }
		default:
			rest = append(rest, args[i])
		}
	}
	return
}

// ── Commands ─────────────────────────────────────────────────────────────────

func cmdSearch(apiKey string, args []string) error {
	mediaType := ""
	limit := 5
	year := ""
	query := []string{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-t", "--type":
			if i+1 < len(args) { i++; mediaType = args[i] }
		case "-l", "--limit":
			if i+1 < len(args) { i++; limit, _ = strconv.Atoi(args[i]) }
		case "-y", "--year":
			if i+1 < len(args) { i++; year = args[i] }
		default:
			query = append(query, args[i])
		}
	}

	if len(query) == 0 {
		return fmt.Errorf("usage: moviedb search [-t movie|tv|person] [-l N] [-y year] <query>")
	}

	endpoint := "/search/multi"
	switch mediaType {
	case "movie":  endpoint = "/search/movie"
	case "tv":     endpoint = "/search/tv"
	case "person": endpoint = "/search/person"
	}

	params := map[string]string{"query": strings.Join(query, " ")}
	if year != "" { params["year"] = year }

	body, err := apiGet(apiKey, endpoint, params)
	if err != nil { return err }

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil { return err }

	printHeader(fmt.Sprintf(`Search: "%s" (%d results)`, strings.Join(query, " "), result.TotalResults))

	shown := 0
	for i, item := range result.Results {
		if shown >= limit { break }
		title := item.Title; if title == "" { title = item.Name }
		date := item.ReleaseDate; if date == "" { date = item.FirstAirDate }
		yr := ""; if len(date) >= 4 { yr = date[:4] }
		mtype := item.MediaType; if mtype == "" { mtype = mediaType }

		fmt.Printf("\n  %s. %s", c(bold, strconv.Itoa(i+1)), c(bold, title))
		if yr != "" { fmt.Printf(" %s", c(dim, "("+yr+")")) }
		if mtype != "" { fmt.Printf(" %s", c(magenta, "["+mtype+"]")) }
		fmt.Println()
		if item.VoteAverage > 0 {
			fmt.Printf("     %s  %s\n",
				c(green, fmt.Sprintf("★ %.1f/10", item.VoteAverage)),
				c(dim, fmt.Sprintf("(%d votes)", item.VoteCount)))
		}
		if item.Overview != "" {
			fmt.Printf("     %s\n", c(dim, truncate(item.Overview, 120)))
		}
		fmt.Printf("     %s %d\n", c(dim, "ID:"), item.ID)
		shown++
	}
	fmt.Println()
	return nil
}

func cmdMovie(apiKey string, args []string) error {
	rest, flags := parseCommon(args)
	if len(rest) == 0 { return fmt.Errorf("usage: moviedb movie <id> [--export json|yaml|toml] [--lang xx-XX]") }
	id := rest[0]

	appendItems := "credits,images,videos,reviews,similar,keywords,watch/providers"
	body, err := apiGet(apiKey, "/movie/"+id, map[string]string{
		"append_to_response": appendItems,
		"language":           flags.Lang,
		"include_image_language": flags.Lang[:2] + ",en,null",
	})
	if err != nil { return err }

	var m MovieDetail
	if err := json.Unmarshal(body, &m); err != nil { return err }

	printHeader(m.Title)
	if m.OriginalTitle != "" && m.OriginalTitle != m.Title {
		fmt.Printf("  %s  %s\n", c(dim, "Original Title:    "), m.OriginalTitle)
	}
	if m.Tagline != "" { fmt.Printf("  %s\n\n", c(magenta, `"`+m.Tagline+`"`)) }

	printField("Release Date",     m.ReleaseDate)
	if m.Runtime > 0 { printField("Runtime", strconv.Itoa(m.Runtime)+" min") }
	printField("Status",           m.Status)
	printField("Original Language",m.OriginalLanguage)
	printField("Genres",           genreNames(m.Genres))
	printField("IMDb ID",          m.IMDbID)
	fmt.Printf("  %s  %s  %s\n",
		c(yellow, fmt.Sprintf("%-20s", "Rating:")),
		c(green, ratingBar(m.VoteAverage)),
		c(dim, fmt.Sprintf("(%d votes)  pop: %.1f", m.VoteCount, m.Popularity)))
	if m.Budget > 0  { printField("Budget",  formatMoney(m.Budget)) }
	if m.Revenue > 0 { printField("Revenue", formatMoney(m.Revenue)) }
	if m.Homepage != "" { printField("Homepage", m.Homepage) }

	// Keywords
	if len(m.Keywords.Keywords) > 0 {
		printField("Keywords", truncate(keywordNames(m.Keywords.Keywords), 100))
	}

	// Production
	if len(m.ProductionCompanies) > 0 {
		names := make([]string, len(m.ProductionCompanies))
		for i, p := range m.ProductionCompanies { names[i] = p.Name }
		printField("Production",  strings.Join(names, ", "))
	}

	// Overview
	if m.Overview != "" {
		printSection("Overview")
		fmt.Printf("  %s\n", truncate(m.Overview, 600))
	}

	// Director + Writers
	var directors, writers []string
	for _, crew := range m.Credits.Crew {
		if crew.Job == "Director" { directors = append(directors, crew.Name) }
		if crew.Department == "Writing" { writers = append(writers, crew.Name) }
	}
	if len(directors) > 0 { printField("Director",  strings.Join(directors, ", ")) }
	if len(writers) > 0   { printField("Writers",   strings.Join(writers[:min(3, len(writers))], ", ")) }

	// Cast
	if len(m.Credits.Cast) > 0 {
		printSection("Cast")
		max := 10; if len(m.Credits.Cast) < max { max = len(m.Credits.Cast) }
		for _, cast := range m.Credits.Cast[:max] {
			fmt.Printf("    • %-26s  %s\n", cast.Name, c(dim, "as "+cast.Character))
		}
	}

	// Videos
	if len(m.Videos.Results) > 0 {
		printSection(fmt.Sprintf("Videos (%d)", len(m.Videos.Results)))
		for _, v := range m.Videos.Results {
			official := ""; if v.Official { official = c(green, " ✓official") }
			fmt.Printf("    • %-12s  %-30s  %s%s\n",
				c(magenta, v.Type), v.Name, c(dim, videoURL(v)), official)
		}
	}

	// Images summary
	imgs := m.Images
	total := len(imgs.Posters) + len(imgs.Backdrops) + len(imgs.Logos)
	if total > 0 {
		printSection(fmt.Sprintf("Images  (posters: %d  backdrops: %d  logos: %d)",
			len(imgs.Posters), len(imgs.Backdrops), len(imgs.Logos)))
		fmt.Printf("  %s  Use: moviedb images movie %s  to see all URLs\n",
			c(dim, "Tip:"), id)
		// Show top poster
		if len(imgs.Posters) > 0 {
			fmt.Printf("    Best poster:   %s\n", imageURL(imgs.Posters[0].FilePath, "w500"))
		}
		if len(imgs.Backdrops) > 0 {
			fmt.Printf("    Best backdrop: %s\n", imageURL(imgs.Backdrops[0].FilePath, "w1280"))
		}
	}

	// Watch providers
	if region, ok := m.WatchProviders.Results[flags.Region]; ok {
		printSection("Where to Watch (" + flags.Region + ")")
		printProviders("Stream", region.Flatrate)
		printProviders("Rent",   region.Rent)
		printProviders("Buy",    region.Buy)
		printProviders("Free",   region.Free)
		if region.Link != "" {
			fmt.Printf("    %s %s\n", c(dim, "More:"), region.Link)
		}
	}

	// Reviews summary
	if m.Reviews.TotalResults > 0 {
		printSection(fmt.Sprintf("Reviews (%d total)", m.Reviews.TotalResults))
		max := 2; if len(m.Reviews.Results) < max { max = len(m.Reviews.Results) }
		for _, rev := range m.Reviews.Results[:max] {
			rating := ""
			if rev.AuthorDetails.Rating > 0 {
				rating = fmt.Sprintf(" [%.0f/10]", rev.AuthorDetails.Rating)
			}
			fmt.Printf("    %s%s\n", c(bold, rev.Author), c(green, rating))
			fmt.Printf("    %s\n\n", c(dim, truncate(rev.Content, 200)))
		}
	}

	// Similar
	if len(m.Similar.Results) > 0 {
		printSection("Similar Movies")
		max := 5; if len(m.Similar.Results) < max { max = len(m.Similar.Results) }
		for _, s := range m.Similar.Results[:max] {
			yr := ""; if len(s.ReleaseDate) >= 4 { yr = s.ReleaseDate[:4] }
			fmt.Printf("    • %-32s %s  %s  %s\n",
				s.Title, c(dim, "("+yr+")"),
				c(green, fmt.Sprintf("★ %.1f", s.VoteAverage)),
				c(dim, fmt.Sprintf("ID:%d", s.ID)))
		}
	}

	fmt.Println()

	// Export
	if flags.ExportFmt != "" {
		fname := flags.ExportFile
		if fname == "" { fname = fmt.Sprintf("movie_%s.%s", id, flags.ExportFmt) }
		return exportData(m, flags.ExportFmt, fname)
	}
	return nil
}

func cmdTV(apiKey string, args []string) error {
	rest, flags := parseCommon(args)
	if len(rest) == 0 { return fmt.Errorf("usage: moviedb tv <id> [--export json|yaml|toml]") }
	id := rest[0]

	appendItems := "credits,images,videos,reviews,similar,keywords,watch/providers"
	body, err := apiGet(apiKey, "/tv/"+id, map[string]string{
		"append_to_response":      appendItems,
		"language":                flags.Lang,
		"include_image_language":  flags.Lang[:2] + ",en,null",
	})
	if err != nil { return err }

	var t TVDetail
	if err := json.Unmarshal(body, &t); err != nil { return err }

	printHeader(t.Name)
	if t.OriginalName != "" && t.OriginalName != t.Name {
		fmt.Printf("  %s  %s\n", c(dim, "Original Name:     "), t.OriginalName)
	}
	if t.Tagline != "" { fmt.Printf("  %s\n\n", c(magenta, `"`+t.Tagline+`"`)) }

	printField("First Air Date",   t.FirstAirDate)
	printField("Last Air Date",    t.LastAirDate)
	printField("Status",           t.Status)
	printField("Type",             t.Type)
	printField("Genres",           genreNames(t.Genres))
	printField("Seasons",          strconv.Itoa(t.NumberOfSeasons))
	printField("Episodes",         strconv.Itoa(t.NumberOfEpisodes))
	if len(t.EpisodeRunTime) > 0 {
		printField("Episode Length", strconv.Itoa(t.EpisodeRunTime[0])+" min")
	}
	nets := make([]string, len(t.Networks))
	for i, n := range t.Networks { nets[i] = n.Name }
	printField("Networks", strings.Join(nets, ", "))
	printField("In Production",    fmt.Sprintf("%v", t.InProduction))
	fmt.Printf("  %s  %s  %s\n",
		c(yellow, fmt.Sprintf("%-20s", "Rating:")),
		c(green, ratingBar(t.VoteAverage)),
		c(dim, fmt.Sprintf("(%d votes)  pop: %.1f", t.VoteCount, t.Popularity)))

	if len(t.Keywords.Results) > 0 {
		printField("Keywords", truncate(keywordNames(t.Keywords.Results), 100))
	}

	if t.Overview != "" {
		printSection("Overview")
		fmt.Printf("  %s\n", truncate(t.Overview, 600))
	}

	if len(t.Credits.Cast) > 0 {
		printSection("Cast")
		max := 10; if len(t.Credits.Cast) < max { max = len(t.Credits.Cast) }
		for _, cast := range t.Credits.Cast[:max] {
			fmt.Printf("    • %-26s  %s\n", cast.Name, c(dim, "as "+cast.Character))
		}
	}

	// Seasons list
	if len(t.Seasons) > 0 {
		printSection(fmt.Sprintf("Seasons (%d)", len(t.Seasons)))
		for _, s := range t.Seasons {
			yr := ""; if len(s.AirDate) >= 4 { yr = s.AirDate[:4] }
			fmt.Printf("    • S%-2d  %-28s %s  %s\n",
				s.SeasonNumber, s.Name,
				c(dim, "("+yr+")"),
				c(dim, fmt.Sprintf("%d eps", s.EpisodeCount)))
		}
		fmt.Printf("  %s  Use: moviedb season %s <N>  for episode list\n", c(dim, "Tip:"), id)
	}

	// Videos
	if len(t.Videos.Results) > 0 {
		printSection(fmt.Sprintf("Videos (%d)", len(t.Videos.Results)))
		for _, v := range t.Videos.Results {
			official := ""; if v.Official { official = c(green, " ✓official") }
			fmt.Printf("    • %-12s  %-30s  %s%s\n",
				c(magenta, v.Type), v.Name, c(dim, videoURL(v)), official)
		}
	}

	// Images
	imgs := t.Images
	total := len(imgs.Posters) + len(imgs.Backdrops) + len(imgs.Logos)
	if total > 0 {
		printSection(fmt.Sprintf("Images  (posters: %d  backdrops: %d  logos: %d)",
			len(imgs.Posters), len(imgs.Backdrops), len(imgs.Logos)))
		if len(imgs.Posters)   > 0 { fmt.Printf("    Best poster:   %s\n", imageURL(imgs.Posters[0].FilePath, "w500")) }
		if len(imgs.Backdrops) > 0 { fmt.Printf("    Best backdrop: %s\n", imageURL(imgs.Backdrops[0].FilePath, "w1280")) }
		fmt.Printf("  %s  Use: moviedb images tv %s  to see all URLs\n", c(dim, "Tip:"), id)
	}

	// Watch providers
	if region, ok := t.WatchProviders.Results[flags.Region]; ok {
		printSection("Where to Watch (" + flags.Region + ")")
		printProviders("Stream", region.Flatrate)
		printProviders("Rent",   region.Rent)
		printProviders("Buy",    region.Buy)
		printProviders("Free",   region.Free)
	}

	// Similar
	if len(t.Similar.Results) > 0 {
		printSection("Similar Shows")
		max := 5; if len(t.Similar.Results) < max { max = len(t.Similar.Results) }
		for _, s := range t.Similar.Results[:max] {
			yr := ""; if len(s.FirstAirDate) >= 4 { yr = s.FirstAirDate[:4] }
			fmt.Printf("    • %-32s %s  %s  %s\n",
				s.Name, c(dim, "("+yr+")"),
				c(green, fmt.Sprintf("★ %.1f", s.VoteAverage)),
				c(dim, fmt.Sprintf("ID:%d", s.ID)))
		}
	}

	fmt.Println()
	if flags.ExportFmt != "" {
		fname := flags.ExportFile
		if fname == "" { fname = fmt.Sprintf("tv_%s.%s", id, flags.ExportFmt) }
		return exportData(t, flags.ExportFmt, fname)
	}
	return nil
}

func cmdPerson(apiKey string, args []string) error {
	rest, flags := parseCommon(args)
	if len(rest) == 0 { return fmt.Errorf("usage: moviedb person <id> [--export json|yaml|toml]") }
	id := rest[0]

	body, err := apiGet(apiKey, "/person/"+id, map[string]string{
		"append_to_response": "movie_credits,tv_credits,images",
		"language":           flags.Lang,
	})
	if err != nil { return err }

	var p PersonDetail
	if err := json.Unmarshal(body, &p); err != nil { return err }

	printHeader(p.Name)
	printField("Known For",    p.KnownForDepartment)
	printField("Born",         p.Birthday)
	if p.Deathday != "" { printField("Died", p.Deathday) }
	printField("Birthplace",   p.PlaceOfBirth)
	printField("IMDb ID",      p.IMDbID)
	if len(p.AlsoKnownAs) > 0 {
		printField("Also Known As", strings.Join(p.AlsoKnownAs[:min(3, len(p.AlsoKnownAs))], " / "))
	}
	if p.ProfilePath != "" {
		printField("Photo", imageURL(p.ProfilePath, "w185"))
	}

	if p.Biography != "" {
		printSection("Biography")
		fmt.Printf("  %s\n", truncate(p.Biography, 700))
	}

	if len(p.MovieCredits.Cast) > 0 {
		credits := p.MovieCredits.Cast
		sort.Slice(credits, func(i, j int) bool {
			return credits[i].VoteAverage > credits[j].VoteAverage
		})
		printSection(fmt.Sprintf("Top Movies (%d total)", len(credits)))
		max := 10; if len(credits) < max { max = len(credits) }
		for _, cr := range credits[:max] {
			yr := ""; if len(cr.ReleaseDate) >= 4 { yr = cr.ReleaseDate[:4] }
			fmt.Printf("    • %-32s %s  %s  %s\n",
				cr.Title, c(dim, "("+yr+")"),
				c(green, fmt.Sprintf("★ %.1f", cr.VoteAverage)),
				c(dim, "as "+truncate(cr.Character, 28)))
		}
	}

	if len(p.TVCredits.Cast) > 0 {
		credits := p.TVCredits.Cast
		sort.Slice(credits, func(i, j int) bool {
			return credits[i].VoteAverage > credits[j].VoteAverage
		})
		printSection(fmt.Sprintf("Top TV Shows (%d total)", len(credits)))
		max := 6; if len(credits) < max { max = len(credits) }
		for _, cr := range credits[:max] {
			yr := ""; if len(cr.FirstAirDate) >= 4 { yr = cr.FirstAirDate[:4] }
			fmt.Printf("    • %-32s %s  %s\n",
				cr.Name, c(dim, "("+yr+")"),
				c(green, fmt.Sprintf("★ %.1f", cr.VoteAverage)))
		}
	}

	if len(p.Images.Profiles) > 0 {
		printSection(fmt.Sprintf("Profile Images (%d)", len(p.Images.Profiles)))
		for i, img := range p.Images.Profiles {
			fmt.Printf("    %d. %s  %s\n",
				i+1,
				imageURL(img.FilePath, "w185"),
				c(dim, fmt.Sprintf("(%dx%d)", img.Width, img.Height)))
		}
	}

	fmt.Println()
	if flags.ExportFmt != "" {
		fname := flags.ExportFile
		if fname == "" { fname = fmt.Sprintf("person_%s.%s", id, flags.ExportFmt) }
		return exportData(p, flags.ExportFmt, fname)
	}
	return nil
}

func cmdImages(apiKey string, args []string) error {
	rest, flags := parseCommon(args)
	if len(rest) < 2 { return fmt.Errorf("usage: moviedb images <movie|tv|person> <id> [--size w500] [--type poster|backdrop|logo|profile] [--export json|yaml|toml|csv]") }

	mediaType := rest[0]
	id := rest[1]
	sizeFilter := "original"
	typeFilter := ""

	for i := 2; i < len(rest); i++ {
		switch rest[i] {
		case "--size", "-s":
			if i+1 < len(rest) { i++; sizeFilter = rest[i] }
		case "--type", "-t":
			if i+1 < len(rest) { i++; typeFilter = rest[i] }
		}
	}

	var endpoint string
	switch mediaType {
	case "movie":  endpoint = "/movie/" + id + "/images"
	case "tv":     endpoint = "/tv/" + id + "/images"
	case "person": endpoint = "/person/" + id + "/images"
	default:
		return fmt.Errorf("media type must be movie, tv, or person")
	}

	body, err := apiGet(apiKey, endpoint, map[string]string{
		"include_image_language": "en,null",
	})
	if err != nil { return err }

	var imgs ImageCollection
	if err := json.Unmarshal(body, &imgs); err != nil { return err }

	// Build sized URLs and determine appropriate sizes per type
	type imageGroup struct {
		typeName string
		items    []ImageItem
		sizes    []string
	}

	groups := []imageGroup{
		{"poster",   imgs.Posters,   posterSizes},
		{"backdrop", imgs.Backdrops, backdropSizes},
		{"logo",     imgs.Logos,     logoSizes},
		{"profile",  imgs.Profiles,  profileSizes},
		{"still",    imgs.Stills,    posterSizes},
	}

	var exportImages []ExportImage
	var csvRows [][]string

	printHeader(fmt.Sprintf("Images: %s %s", strings.ToUpper(mediaType), id))

	for _, grp := range groups {
		if len(grp.items) == 0 { continue }
		if typeFilter != "" && grp.typeName != typeFilter { continue }

		printSection(fmt.Sprintf("%s (%d)", strings.ToUpper(grp.typeName), len(grp.items)))

		// pick display size
		displaySize := sizeFilter
		if sizeFilter == "original" {
			switch grp.typeName {
			case "backdrop": displaySize = "w1280"
			case "poster", "profile": displaySize = "w500"
			case "logo": displaySize = "w300"
			default: displaySize = "w500"
			}
		}

		for i, img := range grp.items {
			lang := img.ISO639_1; if lang == "" { lang = "—" }
			fmt.Printf("\n  %s %d  %s\n",
				c(bold, fmt.Sprintf("[%s]", strings.ToUpper(grp.typeName))), i+1,
				c(dim, fmt.Sprintf("%dx%d  lang:%s  ★%.1f (%dv)",
					img.Width, img.Height, lang, img.VoteAverage, img.VoteCount)))
			fmt.Printf("    %s  %s\n", c(yellow, displaySize+":"), imageURL(img.FilePath, displaySize))
			fmt.Printf("    %s  %s\n", c(dim, "original:"), imageURL(img.FilePath, "original"))

			// All size variants (compact)
			fmt.Printf("    %s  ", c(dim, "all sizes:"))
			var sizeURLs []string
			for _, sz := range grp.sizes {
				sizeURLs = append(sizeURLs, c(cyan, sz)+":"+imageURL(img.FilePath, sz))
			}
			fmt.Println(strings.Join(sizeURLs[:min(4, len(sizeURLs))], "  "))

			ei := ExportImage{
				Type:        grp.typeName,
				FilePath:    img.FilePath,
				URL:         imageURL(img.FilePath, "original"),
				Width:       img.Width,
				Height:      img.Height,
				Language:    img.ISO639_1,
				VoteAverage: img.VoteAverage,
				VoteCount:   img.VoteCount,
			}
			if displaySize != "original" { ei.URLLarge = imageURL(img.FilePath, displaySize) }
			exportImages = append(exportImages, ei)
			csvRows = append(csvRows, []string{
				grp.typeName, img.FilePath, imageURL(img.FilePath, "original"),
				imageURL(img.FilePath, displaySize),
				strconv.Itoa(img.Width), strconv.Itoa(img.Height),
				img.ISO639_1,
				strconv.FormatFloat(img.VoteAverage, 'f', 2, 64),
				strconv.Itoa(img.VoteCount),
			})
		}
	}
	fmt.Println()

	if flags.ExportFmt != "" {
		fname := flags.ExportFile
		if fname == "" { fname = fmt.Sprintf("images_%s_%s.%s", mediaType, id, flags.ExportFmt) }
		if strings.ToLower(flags.ExportFmt) == "csv" {
			headers := []string{"type","file_path","url_original","url_large","width","height","language","vote_average","vote_count"}
			out := marshalCSV(headers, csvRows)
			if err := os.WriteFile(fname, out, 0644); err != nil { return err }
			fmt.Printf("\n  %s  Exported to %s\n", c(green, "✓"), c(bold, fname))
		} else {
			return exportData(exportImages, flags.ExportFmt, fname)
		}
	}
	return nil
}

func cmdDownload(apiKey string, args []string) error {
	rest, flags := parseCommon(args)
	if len(rest) < 2 {
		return fmt.Errorf("usage: moviedb download <movie|tv|person> <id> [--type poster|backdrop|logo|all] [--size w500] [--dir ./images] [--limit N]")
	}

	mediaType := rest[0]
	id := rest[1]
	imgType := "all"
	size := "original"
	outDir := fmt.Sprintf("./%s_%s_images", mediaType, id)
	dlLimit := 0

	for i := 2; i < len(rest); i++ {
		switch rest[i] {
		case "--type", "-t":
			if i+1 < len(rest) { i++; imgType = rest[i] }
		case "--size", "-s":
			if i+1 < len(rest) { i++; size = rest[i] }
		case "--dir", "-d":
			if i+1 < len(rest) { i++; outDir = rest[i] }
		case "--limit", "-l":
			if i+1 < len(rest) { i++; dlLimit, _ = strconv.Atoi(rest[i]) }
		}
	}
	_ = flags

	endpoint := fmt.Sprintf("/%s/%s/images", mediaType, id)
	body, err := apiGet(apiKey, endpoint, map[string]string{"include_image_language": "en,null"})
	if err != nil { return err }

	var imgs ImageCollection
	if err := json.Unmarshal(body, &imgs); err != nil { return err }

	type dlItem struct { url, filename string }
	var items []dlItem

	addItems := func(typeName string, list []ImageItem, sizes []string) {
		if imgType != "all" && typeName != imgType { return }
		sz := size
		if sz == "original" {
			switch typeName {
			case "backdrop": sz = "w1280"
			case "logo":     sz = "w300"
			default:         sz = "w500"
			}
		}
		for i, img := range list {
			if dlLimit > 0 && len(items) >= dlLimit { break }
			ext := filepath.Ext(img.FilePath)
			if ext == "" { ext = ".jpg" }
			fname := fmt.Sprintf("%s_%03d_%s%s", typeName, i+1, sz, ext)
			items = append(items, dlItem{imageURL(img.FilePath, sz), fname})
		}
	}

	addItems("poster",   imgs.Posters,   posterSizes)
	addItems("backdrop", imgs.Backdrops, backdropSizes)
	addItems("logo",     imgs.Logos,     logoSizes)
	addItems("profile",  imgs.Profiles,  profileSizes)

	if len(items) == 0 {
		fmt.Println(c(yellow, "\n  No images found for the specified filters.\n"))
		return nil
	}

	if err := os.MkdirAll(outDir, 0755); err != nil { return err }

	printHeader(fmt.Sprintf("Downloading %d images → %s", len(items), outDir))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // 5 concurrent downloads
	client := &http.Client{Timeout: 30 * time.Second}
	var mu sync.Mutex
	downloaded := 0
	failed := 0

	for _, item := range items {
		wg.Add(1)
		go func(it dlItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			resp, err := client.Get(it.url)
			if err != nil {
				mu.Lock(); failed++; mu.Unlock()
				fmt.Printf("  %s  %s\n", c(red, "✗"), it.filename)
				return
			}
			defer resp.Body.Close()
			data, _ := io.ReadAll(resp.Body)
			outPath := filepath.Join(outDir, it.filename)
			if err := os.WriteFile(outPath, data, 0644); err != nil {
				mu.Lock(); failed++; mu.Unlock()
				return
			}
			mu.Lock()
			downloaded++
			fmt.Printf("  %s  %s  %s\n",
				c(green, "✓"),
				it.filename,
				c(dim, fmt.Sprintf("(%d KB)", len(data)/1024)))
			mu.Unlock()
		}(item)
	}
	wg.Wait()

	fmt.Printf("\n  Done: %s downloaded, %s failed\n\n",
		c(green, strconv.Itoa(downloaded)),
		c(red, strconv.Itoa(failed)))
	return nil
}

func cmdVideos(apiKey string, args []string) error {
	rest, flags := parseCommon(args)
	if len(rest) < 2 { return fmt.Errorf("usage: moviedb videos <movie|tv> <id> [--export json|yaml|toml]") }

	mediaType, id := rest[0], rest[1]
	body, err := apiGet(apiKey, fmt.Sprintf("/%s/%s/videos", mediaType, id),
		map[string]string{"language": flags.Lang})
	if err != nil { return err }

	var result VideoResults
	if err := json.Unmarshal(body, &result); err != nil { return err }

	printHeader(fmt.Sprintf("Videos: %s %s (%d)", strings.ToUpper(mediaType), id, len(result.Results)))

	// Group by type
	byType := map[string][]Video{}
	for _, v := range result.Results {
		byType[v.Type] = append(byType[v.Type], v)
	}
	order := []string{"Trailer", "Teaser", "Clip", "Featurette", "Behind the Scenes", "Bloopers"}
	for _, t := range order {
		if vids, ok := byType[t]; ok {
			printSection(t + "s")
			for _, v := range vids {
				official := ""; if v.Official { official = c(green, " [official]") }
				fmt.Printf("    • %s%s\n      %s\n",
					c(bold, v.Name), official,
					c(cyan, videoURL(v)))
			}
		}
	}
	fmt.Println()

	if flags.ExportFmt != "" {
		var exportVids []ExportVideo
		for _, v := range result.Results {
			exportVids = append(exportVids, ExportVideo{
				Name: v.Name, Type: v.Type, Site: v.Site,
				URL: videoURL(v), Size: v.Size, Official: v.Official,
			})
		}
		fname := flags.ExportFile
		if fname == "" { fname = fmt.Sprintf("videos_%s_%s.%s", mediaType, id, flags.ExportFmt) }
		return exportData(exportVids, flags.ExportFmt, fname)
	}
	return nil
}

func cmdReviews(apiKey string, args []string) error {
	rest, flags := parseCommon(args)
	if len(rest) < 2 { return fmt.Errorf("usage: moviedb reviews <movie|tv> <id> [--page N]") }

	mediaType, id := rest[0], rest[1]
	page := "1"
	for i := 2; i < len(rest); i++ {
		if (rest[i] == "--page" || rest[i] == "-p") && i+1 < len(rest) {
			i++; page = rest[i]
		}
	}

	body, err := apiGet(apiKey, fmt.Sprintf("/%s/%s/reviews", mediaType, id),
		map[string]string{"page": page, "language": flags.Lang})
	if err != nil { return err }

	var result ReviewResults
	if err := json.Unmarshal(body, &result); err != nil { return err }

	printHeader(fmt.Sprintf("Reviews: %s %s  (%d total, page %s/%d)",
		strings.ToUpper(mediaType), id, result.TotalResults, page, result.TotalPages))

	for _, rev := range result.Results {
		rating := c(dim, "no rating")
		if rev.AuthorDetails.Rating > 0 {
			rating = c(green, fmt.Sprintf("★ %.0f/10", rev.AuthorDetails.Rating))
		}
		date := ""
		if len(rev.CreatedAt) >= 10 { date = rev.CreatedAt[:10] }
		fmt.Printf("\n  %s  %s  %s\n",
			c(bold, rev.Author), rating, c(dim, date))
		fmt.Printf("  %s\n", strings.Repeat("·", 50))
		// word-wrap at ~80 chars
		words := strings.Fields(rev.Content)
		line := "  "
		for _, w := range words {
			if len(line)+len(w)+1 > 80 {
				fmt.Println(line)
				line = "  " + w + " "
			} else {
				line += w + " "
			}
		}
		if line != "  " { fmt.Println(line) }
	}
	fmt.Println()
	return nil
}

func cmdSeason(apiKey string, args []string) error {
	rest, flags := parseCommon(args)
	if len(rest) < 2 { return fmt.Errorf("usage: moviedb season <tv_id> <season_number> [--export json|yaml|toml]") }

	tvID, seasonNum := rest[0], rest[1]
	body, err := apiGet(apiKey, fmt.Sprintf("/tv/%s/season/%s", tvID, seasonNum),
		map[string]string{"language": flags.Lang})
	if err != nil { return err }

	var s SeasonDetail
	if err := json.Unmarshal(body, &s); err != nil { return err }

	printHeader(fmt.Sprintf("Season %d: %s", s.SeasonNumber, s.Name))
	printField("Air Date",  s.AirDate)
	printField("Episodes",  strconv.Itoa(len(s.Episodes)))
	if s.Overview != "" {
		printSection("Overview")
		fmt.Printf("  %s\n", truncate(s.Overview, 400))
	}

	printSection(fmt.Sprintf("Episodes (%d)", len(s.Episodes)))
	for _, ep := range s.Episodes {
		runtime := ""
		if ep.Runtime > 0 { runtime = fmt.Sprintf("  %dmin", ep.Runtime) }
		rating := ""
		if ep.VoteAverage > 0 { rating = c(green, fmt.Sprintf("  ★%.1f", ep.VoteAverage)) }
		fmt.Printf("\n  %s  %s%s%s\n",
			c(bold, fmt.Sprintf("E%02d", ep.EpisodeNumber)),
			c(white, ep.Name),
			c(dim, runtime),
			rating)
		fmt.Printf("       %s\n", c(dim, "Air: "+ep.AirDate))
		if ep.Overview != "" {
			fmt.Printf("       %s\n", c(dim, truncate(ep.Overview, 120)))
		}
	}
	fmt.Println()

	if flags.ExportFmt != "" {
		fname := flags.ExportFile
		if fname == "" { fname = fmt.Sprintf("season_%s_%s.%s", tvID, seasonNum, flags.ExportFmt) }
		return exportData(s, flags.ExportFmt, fname)
	}
	return nil
}

func cmdTrending(apiKey string, args []string) error {
	rest, flags := parseCommon(args)
	mediaType := "movie"
	window := "week"

	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "-t", "--type":
			if i+1 < len(rest) { i++; mediaType = rest[i] }
		case "-w", "--window":
			if i+1 < len(rest) { i++; window = rest[i] }
		}
	}

	body, err := apiGet(apiKey, fmt.Sprintf("/trending/%s/%s", mediaType, window), nil)
	if err != nil { return err }

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil { return err }

	printHeader(fmt.Sprintf("Trending %s (%s)", strings.ToUpper(mediaType), window))
	for i, item := range result.Results {
		title := item.Title; if title == "" { title = item.Name }
		date := item.ReleaseDate; if date == "" { date = item.FirstAirDate }
		yr := ""; if len(date) >= 4 { yr = date[:4] }

		fmt.Printf("\n  %s. %s", c(bold, strconv.Itoa(i+1)), c(bold, title))
		if yr != "" { fmt.Printf(" %s", c(dim, "("+yr+")")) }
		fmt.Println()
		fmt.Printf("     %s  %s  %s  %s\n",
			c(green, fmt.Sprintf("★ %.1f/10", item.VoteAverage)),
			c(dim, fmt.Sprintf("(%d votes)", item.VoteCount)),
			c(magenta, fmt.Sprintf("pop: %.0f", item.Popularity)),
			c(dim, fmt.Sprintf("ID:%d", item.ID)))
		if item.Overview != "" {
			fmt.Printf("     %s\n", c(dim, truncate(item.Overview, 110)))
		}
	}
	fmt.Println()

	if flags.ExportFmt != "" {
		fname := flags.ExportFile
		if fname == "" { fname = fmt.Sprintf("trending_%s_%s.%s", mediaType, window, flags.ExportFmt) }
		return exportData(result, flags.ExportFmt, fname)
	}
	return nil
}

func printProviders(label string, providers []WatchProvider) {
	if len(providers) == 0 { return }
	names := make([]string, len(providers))
	for i, p := range providers { names[i] = p.ProviderName }
	fmt.Printf("    %-8s  %s\n", c(yellow, label+":"), strings.Join(names, "  •  "))
}

func min(a, b int) int {
	if a < b { return a }
	return b
}

// ── Help ─────────────────────────────────────────────────────────────────────

func printHelp() {
	fmt.Println(c(cyan, `
  🎬 moviedb — Movie & TV CLI powered by TMDb`))
	fmt.Println(c(dim, `  Data: The Movie Database (themoviedb.org)  |  Zero external dependencies`))
	fmt.Printf(`
%s
  Windows:  set TMDB_API_KEY=your_key_here
  Linux:    export TMDB_API_KEY=your_key_here
  Get a free key at: https://www.themoviedb.org/settings/api

%s
  moviedb                   ← no args: launches interactive mode (REPL)
  moviedb interactive       ← same, explicit
  moviedb search    <query> [-t movie|tv|person] [-l N] [-y year]
  moviedb movie     <id>    [--export json|yaml|toml] [--lang xx-XX] [--region XX]
  moviedb tv        <id>    [--export json|yaml|toml] [--lang xx-XX] [--region XX]
  moviedb season    <tv_id> <season_num> [--export json|yaml|toml]
  moviedb person    <id>    [--export json|yaml|toml]
  moviedb images    <movie|tv|person> <id>  [--type poster|backdrop|logo|profile]
                            [--size w500]   [--export json|yaml|toml|csv]
  moviedb download  <movie|tv|person> <id>  [--type poster|backdrop|logo|all]
                            [--size w500]   [--dir ./my_images] [--limit N]
  moviedb videos    <movie|tv> <id>  [--export json|yaml|toml]
  moviedb reviews   <movie|tv> <id>  [--page N]
  moviedb trending  [-t movie|tv|all] [-w day|week] [--export json|yaml|toml]
  moviedb export                      ← interactive export wizard

  %s  Running any command without required args on a TTY will prompt for them.

%s
  moviedb search "Dune" -t movie -l 10
  moviedb movie 693134                        (Dune Part Two)
  moviedb movie 693134 --export json
  moviedb movie 693134 --export yaml --out dune.yaml
  moviedb movie 693134 --lang id-ID           (Indonesian)
  moviedb movie 693134 --region ID            (watch providers for Indonesia)
  moviedb tv 1396 --export toml              (Breaking Bad)
  moviedb season 1396 1 --export json
  moviedb person 6193 --export yaml          (Leonardo DiCaprio)
  moviedb images movie 693134 --type poster
  moviedb images movie 693134 --export csv   (all image URLs as CSV)
  moviedb images tv 1396 --type backdrop --size w1280
  moviedb download movie 693134 --type poster --size w500 --dir ./dune_images
  moviedb download tv 1396 --type all --limit 20
  moviedb videos movie 27205
  moviedb reviews movie 27205 --page 2
  moviedb trending -t tv -w day

%s
  --export   Output format: json, yaml, toml  (csv also for images command)
  --out      Output filename  (default: auto-generated)
  --lang     Language code, e.g. en-US, id-ID, ja-JP, fr-FR  (default: en-US)
  --region   Country code for watch providers, e.g. US, ID, GB  (default: US)
`,
		c(bold, "SETUP:"),
		c(bold, "COMMANDS:"),
		c(dim, "Note:"),
		c(bold, "EXAMPLES:"),
		c(bold, "GLOBAL FLAGS:"))
}

// ── Main ─────────────────────────────────────────────────────────────────────

func main() {
	apiKey := os.Getenv("TMDB_API_KEY")
	args := os.Args[1:]

	// No args and stdin is a TTY → enter interactive mode
	if len(args) == 0 {
		if isTTY() {
			requireAPIKey(apiKey)
			runInteractive(apiKey)
			return
		}
		printHelp()
		return
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "help", "--help", "-h":
		printHelp()
		return
	case "interactive", "i", "repl":
		requireAPIKey(apiKey)
		runInteractive(apiKey)
		return
	}

	requireAPIKey(apiKey)

	// For commands that take a required ID/query, prompt if missing and on a TTY
	var err error
	switch cmd {
	case "search", "s", "find":
		if len(rest) == 0 && isTTY() {
			s := newSession(apiKey)
			s.screenSearch(nil)
		} else {
			err = cmdSearch(apiKey, rest)
		}

	case "movie", "m":
		if len(rest) == 0 && isTTY() {
			s := newSession(apiKey)
			s.screenMediaEntry("movie", nil)
		} else {
			err = cmdMovie(apiKey, rest)
		}

	case "tv", "show":
		if len(rest) == 0 && isTTY() {
			s := newSession(apiKey)
			s.screenMediaEntry("tv", nil)
		} else {
			err = cmdTV(apiKey, rest)
		}

	case "season":
		if len(rest) < 2 && isTTY() {
			s := newSession(apiKey)
			s.screenSeason(rest)
		} else {
			err = cmdSeason(apiKey, rest)
		}

	case "person", "p":
		if len(rest) == 0 && isTTY() {
			s := newSession(apiKey)
			s.screenMediaEntry("person", nil)
		} else {
			err = cmdPerson(apiKey, rest)
		}

	case "images", "img":
		if len(rest) < 2 && isTTY() {
			s := newSession(apiKey)
			s.screenImages(rest)
		} else {
			err = cmdImages(apiKey, rest)
		}

	case "download", "dl":
		if len(rest) < 2 && isTTY() {
			s := newSession(apiKey)
			s.screenDownload(rest)
		} else {
			err = cmdDownload(apiKey, rest)
		}

	case "videos", "v":
		if len(rest) < 2 && isTTY() {
			s := newSession(apiKey)
			s.screenVideos(rest)
		} else {
			err = cmdVideos(apiKey, rest)
		}

	case "reviews", "r":
		if len(rest) < 2 && isTTY() {
			s := newSession(apiKey)
			s.screenReviews(rest)
		} else {
			err = cmdReviews(apiKey, rest)
		}

	case "trending", "hot":
		if len(rest) == 0 && isTTY() {
			s := newSession(apiKey)
			s.screenTrending(nil)
		} else {
			err = cmdTrending(apiKey, rest)
		}

	case "export", "e":
		if isTTY() {
			s := newSession(apiKey)
			s.screenExportWizard()
		} else {
			fmt.Println(c(red, "\n  ✗ Export wizard requires an interactive terminal.\n"))
			os.Exit(1)
		}

	default:
		fmt.Printf(c(red, "\n  ✗ Unknown command: %s\n\n"), cmd)
		printHelp()
		os.Exit(1)
	}

	if err != nil {
		fmt.Println(c(red, "\n  ✗ Error: "+err.Error()+"\n"))
		os.Exit(1)
	}
}

func requireAPIKey(apiKey string) {
	if apiKey == "" {
		fmt.Println(c(red, "\n  ✗ TMDB_API_KEY not set."))
		fmt.Println("  Windows: set TMDB_API_KEY=your_key_here")
		fmt.Println("  Linux:   export TMDB_API_KEY=your_key_here")
		fmt.Println("  Get key: https://www.themoviedb.org/settings/api\n")
		os.Exit(1)
	}
}
