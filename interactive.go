package main

// interactive.go — full REPL + readline-like input with history
// Self-contained navigation stack: each "screen" owns its own read loop.
// The outer runInteractive loop ONLY sees top-level commands.

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ── Session state ─────────────────────────────────────────────────────────────

type Session struct {
	apiKey  string
	lang    string
	region  string
	history []string
	histPos int
}

func newSession(apiKey string) *Session {
	return &Session{
		apiKey:  apiKey,
		lang:    "en-US",
		region:  "US",
		histPos: -1,
	}
}

// ── Readline wrapper ──────────────────────────────────────────────────────────

func (s *Session) read(promptStr string) (string, bool) {
	line, err := readlineRaw(promptStr, &s.history, &s.histPos)
	if err != nil {
		return "", false // Ctrl+C / EOF → signal quit
	}
	line = strings.TrimSpace(line)
	if line != "" {
		if len(s.history) == 0 || s.history[len(s.history)-1] != line {
			s.history = append(s.history, line)
		}
		s.histPos = len(s.history)
	}
	return line, true
}

// isQuit returns true for universal quit signals
func isQuit(s string) bool {
	switch strings.ToLower(s) {
	case "q", "quit", "exit":
		return true
	}
	return false
}

// isBack returns true for back/exit signals from sub-screens
func isBack(s string) bool {
	switch strings.ToLower(s) {
	case "b", "back", "":
		return true
	}
	return false
}

// ── Prompt helpers ────────────────────────────────────────────────────────────

// plainPrompt reads a line using the shared scanner (safe on all platforms)
func plainPrompt(label string) string {
	fmt.Printf("\n  %s %s", c(yellow, label), c(cyan, "› "))
	sc := getScanner()
	if sc.Scan() {
		return strings.TrimSpace(sc.Text())
	}
	return ""
}

// selectPrompt shows numbered options and returns 0-based index, or -1 for back
func selectPrompt(s *Session, label string, options []string) int {
	fmt.Println()
	for i, o := range options {
		fmt.Printf("    %s  %s\n", c(cyan, fmt.Sprintf("[%d]", i+1)), o)
	}
	fmt.Printf("    %s  %s\n", c(dim, "[0]"), "Back")
	fmt.Println()
	for {
		raw, ok := s.read(fmt.Sprintf("  %s %s", c(yellow, label), c(cyan, "›")))
		if !ok || isQuit(raw) { return -1 }
		if raw == "" || raw == "0" || isBack(raw) { return -1 }
		n, err := strconv.Atoi(raw)
		if err == nil && n >= 1 && n <= len(options) { return n - 1 }
		fmt.Printf("  %s Enter 1-%d or 0 to go back\n", c(red, "✗"), len(options))
	}
}

// ── Prompt string ─────────────────────────────────────────────────────────────

func (s *Session) ps(crumbs ...string) string {
	base := c(green, "moviedb")
	if len(crumbs) > 0 {
		base += c(dim, " ["+strings.Join(crumbs, " › ")+"]")
	}
	if s.lang != "en-US" {
		base += c(magenta, " "+s.lang)
	}
	return base + " " + c(cyan, "›")
}

// ── Banner ────────────────────────────────────────────────────────────────────

func printBanner() {
	fmt.Println()
	fmt.Println(c(cyan,  "  ╔══════════════════════════════════════════════════════════╗"))
	fmt.Printf( "  %s  %s %s  %s\n",
		c(cyan, "║"), c(bold, "🎬  moviedb interactive"), c(dim, "— powered by TMDb     "), c(cyan, "║"))
	fmt.Printf( "  %s  %s  %s\n",
		c(cyan, "║"), c(dim, "Enter for menu  •  help  •  q to quit  •  ↑↓ history  "), c(cyan, "║"))
	fmt.Println(c(cyan,  "  ╚══════════════════════════════════════════════════════════╝"))
	fmt.Println()
}

// ── Main REPL ─────────────────────────────────────────────────────────────────

func runInteractive(apiKey string) {
	s := newSession(apiKey)
	printBanner()

	for {
		line, ok := s.read(s.ps())
		if !ok || isQuit(line) {
			fmt.Println(c(dim, "\n  Goodbye! 👋\n"))
			return
		}

		if line == "" {
			s.screenMain()
			continue
		}

		fields := strings.Fields(line)
		cmd := strings.ToLower(fields[0])
		rest := fields[1:]

		switch cmd {
		case "help", "?", "h":
			printInteractiveHelp()

		case "clear", "cls":
			fmt.Print("\033[H\033[2J")
			printBanner()

		case "lang":
			if len(rest) > 0 {
				s.lang = rest[0]
			} else {
				v := plainPrompt("Language code (e.g. en-US, id-ID, ja-JP):")
				if v != "" { s.lang = v }
			}
			fmt.Printf("  %s Language → %s\n", c(green, "✓"), s.lang)

		case "region":
			if len(rest) > 0 {
				s.region = strings.ToUpper(rest[0])
			} else {
				v := plainPrompt("Region code (e.g. US, ID, GB):")
				if v != "" { s.region = strings.ToUpper(v) }
			}
			fmt.Printf("  %s Region → %s\n", c(green, "✓"), s.region)

		case "history", "hist":
			s.showHistory()

		case "search", "s", "find", "f":
			s.screenSearch(rest)

		case "movie", "m":
			s.screenMediaEntry("movie", rest)

		case "tv", "t", "show":
			s.screenMediaEntry("tv", rest)

		case "person", "p", "actor":
			s.screenMediaEntry("person", rest)

		case "trending", "hot":
			s.screenTrending(rest)

		case "images", "img", "i":
			s.screenImages(rest)

		case "download", "dl", "d":
			s.screenDownload(rest)

		case "videos", "v", "trailers":
			s.screenVideos(rest)

		case "reviews", "r":
			s.screenReviews(rest)

		case "season":
			s.screenSeason(rest)

		case "export", "e":
			s.screenExportWizard()

		default:
			fmt.Printf("  %s Unknown command %s — type %s for help\n",
				c(yellow, "?"), c(bold, "'"+cmd+"'"), c(cyan, "help"))
		}
	}
}

// ── Main menu screen ──────────────────────────────────────────────────────────

func (s *Session) screenMain() {
	fmt.Println()
	fmt.Println(c(bold, "  ┌─ Main Menu ──────────────────────────────────────────┐"))
	row := func(k, l, h string) {
		fmt.Printf("  │  %-6s  %-12s %s\n", c(cyan, k), l, c(dim, h))
	}
	row("s",      "search",   "Search movies, TV shows, people")
	row("hot",    "trending", "What's trending right now")
	row("m",      "movie",    "Movie details by ID or title")
	row("tv",     "tv show",  "TV show details by ID or title")
	row("p",      "person",   "Person details by ID or name")
	fmt.Println(c(bold, "  ├──────────────────────────────────────────────────────┤"))
	row("lang",   "language", fmt.Sprintf("Set language   (now: %s)", s.lang))
	row("region", "region",   fmt.Sprintf("Set region     (now: %s)", s.region))
	row("hist",   "history",  "Command history")
	row("clear",  "clear",    "Clear screen")
	row("q",      "quit",     "Exit")
	fmt.Println(c(bold, "  └──────────────────────────────────────────────────────┘"))
	fmt.Println()
}

// ── Search screen (owns its own loop) ────────────────────────────────────────

func (s *Session) screenSearch(args []string) {
	mediaType := ""
	limit := 10
	queryWords := []string{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-t", "--type":
			if i+1 < len(args) { i++; mediaType = args[i] }
		case "-l", "--limit":
			if i+1 < len(args) { i++; limit, _ = strconv.Atoi(args[i]) }
		default:
			queryWords = append(queryWords, args[i])
		}
	}

	// Prompt for missing pieces
	if len(queryWords) == 0 {
		fmt.Println()
		if mediaType == "" {
			fmt.Println(c(dim, "  Filter by type:"))
			choice := selectPrompt(s, "type", []string{
				"All  (movies + TV + people)",
				"Movies only",
				"TV shows only",
				"People only",
			})
			switch choice {
			case 1: mediaType = "movie"
			case 2: mediaType = "tv"
			case 3: mediaType = "person"
			}
			if choice == -1 { return }
		}
		q := plainPrompt("Search query:")
		if q == "" { return }
		queryWords = strings.Fields(q)
	}

	query := strings.Join(queryWords, " ")

	endpoint := "/search/multi"
	switch mediaType {
	case "movie":  endpoint = "/search/movie"
	case "tv":     endpoint = "/search/tv"
	case "person": endpoint = "/search/person"
	}

	fmt.Printf("\n  %s\n", c(dim, "Searching…"))
	body, err := apiGet(s.apiKey, endpoint, map[string]string{
		"query":    query,
		"language": s.lang,
	})
	if err != nil {
		fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
		return
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
		return
	}

	if len(result.Results) == 0 {
		fmt.Println(c(yellow, "\n  No results found.\n"))
		return
	}

	shown := result.Results
	if len(shown) > limit { shown = shown[:limit] }

	// ── inner pick loop — this is the ONLY loop that reads input here ─────────
	for {
		printHeader(fmt.Sprintf(`Search: "%s"  (%d results, showing %d)`, query, result.TotalResults, len(shown)))

		for i, item := range shown {
			title := item.Title; if title == "" { title = item.Name }
			date := item.ReleaseDate; if date == "" { date = item.FirstAirDate }
			yr := ""; if len(date) >= 4 { yr = date[:4] }
			mt := item.MediaType; if mt == "" { mt = mediaType }

			fmt.Printf("\n  %s  %s", c(cyan, fmt.Sprintf("%2d.", i+1)), c(bold, title))
			if yr != "" { fmt.Printf("  %s", c(dim, "("+yr+")")) }
			if mt != "" { fmt.Printf("  %s", c(magenta, "["+mt+"]")) }
			fmt.Println()
			if item.VoteAverage > 0 {
				fmt.Printf("      %s  %s\n",
					c(green, fmt.Sprintf("★ %.1f/10", item.VoteAverage)),
					c(dim, fmt.Sprintf("(%d votes)", item.VoteCount)))
			}
			if item.Overview != "" {
				fmt.Printf("      %s\n", c(dim, truncate(item.Overview, 110)))
			}
			fmt.Printf("      %s %d\n", c(dim, "ID:"), item.ID)
		}

		fmt.Println()
		fmt.Printf("  %s\n", c(dim, "Pick a number to open · 'back' or Enter to return"))

		raw, ok := s.read(s.ps("search:"+query))
		if !ok || isQuit(raw) { return }
		if isBack(raw) { return }

		// support commands from here too
		fields := strings.Fields(raw)
		if len(fields) > 0 {
			cmd := strings.ToLower(fields[0])
			// new search?
			if cmd == "s" || cmd == "search" {
				s.screenSearch(fields[1:])
				return
			}
		}

		n, err2 := strconv.Atoi(raw)
		if err2 != nil || n < 1 || n > len(shown) {
			fmt.Printf("  %s Enter 1–%d, or 'back'\n", c(red, "✗"), len(shown))
			continue
		}

		item := shown[n-1]
		mt := item.MediaType; if mt == "" { mt = mediaType }
		id := strconv.Itoa(item.ID)
		title := item.Title; if title == "" { title = item.Name }

		// if media type still unknown, ask once
		if mt == "" {
			fmt.Println(c(dim, "  Media type unclear — open as:"))
			choice := selectPrompt(s, "type", []string{"Movie", "TV Show", "Person"})
			switch choice {
			case 0: mt = "movie"
			case 1: mt = "tv"
			case 2: mt = "person"
			default: continue
			}
		}

		// open detail — when it returns (user went back), we loop back to list
		switch mt {
		case "movie":  s.screenMovie(id, title)
		case "tv":     s.screenTV(id, title)
		case "person": s.screenPerson(id, title)
		}
		// after returning from detail, loop shows the list again
	}
}

// ── Media entry (from top-level command) ─────────────────────────────────────

func (s *Session) screenMediaEntry(mtype string, args []string) {
	if len(args) > 0 {
		if isID(args[0]) {
			switch mtype {
			case "movie":  s.screenMovie(args[0], args[0])
			case "tv":     s.screenTV(args[0], args[0])
			case "person": s.screenPerson(args[0], args[0])
			}
			return
		}
		// treat remaining words as search query
		s.screenSearch(append([]string{"-t", mtype}, args...))
		return
	}
	// no args — prompt
	fmt.Println()
	raw := plainPrompt("Enter ID or search query:")
	if raw == "" { return }
	if isID(raw) {
		switch mtype {
		case "movie":  s.screenMovie(raw, raw)
		case "tv":     s.screenTV(raw, raw)
		case "person": s.screenPerson(raw, raw)
		}
	} else {
		s.screenSearch(append([]string{"-t", mtype}, strings.Fields(raw)...))
	}
}

// ── Movie detail screen ───────────────────────────────────────────────────────

func (s *Session) screenMovie(id, title string) {
	fmt.Printf("\n  %s\n", c(dim, "Loading…"))
	if err := cmdMovie(s.apiKey, []string{id, "--lang", s.lang, "--region", s.region}); err != nil {
		fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
		return
	}

	crumb := title
	if crumb == id { crumb = "movie:" + id }

	for {
		fmt.Println()
		fmt.Println(c(bold, "  ┌─ Movie ──────────────────────────────────────────────┐"))
		fmt.Printf("  │  %s  images    %s\n", c(cyan, "i"), c(dim, "All posters / backdrops / logos"))
		fmt.Printf("  │  %s  download  %s\n", c(cyan, "d"), c(dim, "Download images to folder"))
		fmt.Printf("  │  %s  videos    %s\n", c(cyan, "v"), c(dim, "Trailers & clips"))
		fmt.Printf("  │  %s  reviews   %s\n", c(cyan, "r"), c(dim, "User reviews"))
		fmt.Printf("  │  %s  export    %s\n", c(cyan, "e"), c(dim, "Export full data to JSON/YAML/TOML"))
		fmt.Printf("  │  %s  back      %s\n", c(dim, "b"), c(dim, "← Return"))
		fmt.Println(c(bold, "  └──────────────────────────────────────────────────────┘"))

		raw, ok := s.read(s.ps(crumb))
		if !ok || isQuit(raw) { os.Exit(0) }
		if isBack(raw) { return }

		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "i", "images":
			s.screenImagesFor("movie", id)
		case "d", "download", "dl":
			s.screenDownloadFor("movie", id)
		case "v", "videos", "trailers":
			if err := cmdVideos(s.apiKey, []string{"movie", id}); err != nil {
				fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
			}
		case "r", "reviews":
			page := plainPrompt("Page (default 1):")
			if page == "" { page = "1" }
			if err := cmdReviews(s.apiKey, []string{"movie", id, "--page", page}); err != nil {
				fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
			}
		case "e", "export":
			s.screenExportFor("movie", id)
		default:
			// let user run other commands without leaving
			fields := strings.Fields(raw)
			if len(fields) > 0 {
				s.handleInline(fields)
			}
		}
	}
}

// ── TV detail screen ──────────────────────────────────────────────────────────

func (s *Session) screenTV(id, title string) {
	fmt.Printf("\n  %s\n", c(dim, "Loading…"))
	if err := cmdTV(s.apiKey, []string{id, "--lang", s.lang, "--region", s.region}); err != nil {
		fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
		return
	}

	crumb := title
	if crumb == id { crumb = "tv:" + id }

	for {
		fmt.Println()
		fmt.Println(c(bold, "  ┌─ TV Show ────────────────────────────────────────────┐"))
		fmt.Printf("  │  %s  season N  %s\n", c(cyan, "sN"), c(dim, "Open season  e.g. s1  s2  season 3"))
		fmt.Printf("  │  %s  images    %s\n", c(cyan, "i"),  c(dim, "All posters / backdrops / logos"))
		fmt.Printf("  │  %s  download  %s\n", c(cyan, "d"),  c(dim, "Download images to folder"))
		fmt.Printf("  │  %s  videos    %s\n", c(cyan, "v"),  c(dim, "Trailers & clips"))
		fmt.Printf("  │  %s  reviews   %s\n", c(cyan, "r"),  c(dim, "User reviews"))
		fmt.Printf("  │  %s  export    %s\n", c(cyan, "e"),  c(dim, "Export full data to JSON/YAML/TOML"))
		fmt.Printf("  │  %s  back      %s\n", c(dim, "b"),   c(dim, "← Return"))
		fmt.Println(c(bold, "  └──────────────────────────────────────────────────────┘"))

		raw, ok := s.read(s.ps(crumb))
		if !ok || isQuit(raw) { os.Exit(0) }
		if isBack(raw) { return }

		lower := strings.ToLower(strings.TrimSpace(raw))

		// season shorthand: s1 / season 1
		if sn := parseSeasonShorthand(lower); sn != "" {
			s.screenSeason([]string{id, sn})
			continue
		}

		switch lower {
		case "i", "images":
			s.screenImagesFor("tv", id)
		case "d", "download", "dl":
			s.screenDownloadFor("tv", id)
		case "v", "videos", "trailers":
			if err := cmdVideos(s.apiKey, []string{"tv", id}); err != nil {
				fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
			}
		case "r", "reviews":
			page := plainPrompt("Page (default 1):")
			if page == "" { page = "1" }
			if err := cmdReviews(s.apiKey, []string{"tv", id, "--page", page}); err != nil {
				fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
			}
		case "e", "export":
			s.screenExportFor("tv", id)
		default:
			fields := strings.Fields(raw)
			if len(fields) > 0 {
				s.handleInline(fields)
			}
		}
	}
}

// ── Person detail screen ──────────────────────────────────────────────────────

func (s *Session) screenPerson(id, title string) {
	fmt.Printf("\n  %s\n", c(dim, "Loading…"))
	if err := cmdPerson(s.apiKey, []string{id, "--lang", s.lang}); err != nil {
		fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
		return
	}

	crumb := title
	if crumb == id { crumb = "person:" + id }

	for {
		fmt.Println()
		fmt.Println(c(bold, "  ┌─ Person ─────────────────────────────────────────────┐"))
		fmt.Printf("  │  %s  images    %s\n", c(cyan, "i"), c(dim, "Profile photos"))
		fmt.Printf("  │  %s  download  %s\n", c(cyan, "d"), c(dim, "Download profile photos"))
		fmt.Printf("  │  %s  export    %s\n", c(cyan, "e"), c(dim, "Export full data"))
		fmt.Printf("  │  %s  back      %s\n", c(dim, "b"), c(dim, "← Return"))
		fmt.Println(c(bold, "  └──────────────────────────────────────────────────────┘"))

		raw, ok := s.read(s.ps(crumb))
		if !ok || isQuit(raw) { os.Exit(0) }
		if isBack(raw) { return }

		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "i", "images":
			if err := cmdImages(s.apiKey, []string{"person", id}); err != nil {
				fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
			}
		case "d", "download", "dl":
			s.screenDownloadFor("person", id)
		case "e", "export":
			s.screenExportFor("person", id)
		default:
			fields := strings.Fields(raw)
			if len(fields) > 0 {
				s.handleInline(fields)
			}
		}
	}
}

// ── Trending screen ───────────────────────────────────────────────────────────

func (s *Session) screenTrending(args []string) {
	mediaType := ""
	window := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-t", "--type":   if i+1 < len(args) { i++; mediaType = args[i] }
		case "-w", "--window": if i+1 < len(args) { i++; window = args[i] }
		}
	}

	if mediaType == "" {
		fmt.Println()
		choice := selectPrompt(s, "type", []string{"Movies", "TV Shows", "All"})
		switch choice {
		case 0: mediaType = "movie"
		case 1: mediaType = "tv"
		case 2: mediaType = "all"
		default: return
		}
	}
	if window == "" {
		choice := selectPrompt(s, "window", []string{"This week", "Today"})
		switch choice {
		case 0: window = "week"
		case 1: window = "day"
		default: return
		}
	}

	if err := cmdTrending(s.apiKey, []string{"-t", mediaType, "-w", window}); err != nil {
		fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
		return
	}

	// Fetch again for pick list
	body, err := apiGet(s.apiKey, fmt.Sprintf("/trending/%s/%s", mediaType, window), nil)
	if err != nil { return }
	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil { return }

	// re-use search pick loop by calling screenSearch with pre-filled results
	shown := result.Results
	if len(shown) > 20 { shown = shown[:20] }

	for {
		fmt.Printf("\n  %s\n", c(dim, fmt.Sprintf("Pick a number to open · 'back' to return · %d results", len(shown))))
		raw, ok := s.read(s.ps("trending"))
		if !ok || isQuit(raw) { return }
		if isBack(raw) { return }

		n, err2 := strconv.Atoi(raw)
		if err2 != nil || n < 1 || n > len(shown) {
			fmt.Printf("  %s Enter 1–%d or 'back'\n", c(red, "✗"), len(shown))
			continue
		}
		item := shown[n-1]
		mt := item.MediaType; if mt == "" { mt = mediaType }
		id := strconv.Itoa(item.ID)
		title := item.Title; if title == "" { title = item.Name }

		switch mt {
		case "movie": s.screenMovie(id, title)
		case "tv":    s.screenTV(id, title)
		}
	}
}

// ── Season screen ─────────────────────────────────────────────────────────────

func (s *Session) screenSeason(args []string) {
	tvID, seasonNum := "", ""

	if len(args) >= 1 && isID(args[0]) { tvID = args[0] }
	if len(args) >= 2 { seasonNum = args[1] }

	if tvID == "" {
		tvID = plainPrompt("TV Show ID:")
		if tvID == "" { return }
	}
	if seasonNum == "" {
		seasonNum = plainPrompt("Season number:")
		if seasonNum == "" { return }
	}

	if err := cmdSeason(s.apiKey, []string{tvID, seasonNum, "--lang", s.lang}); err != nil {
		fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
		return
	}

	for {
		fmt.Printf("\n  %s\n", c(dim, "Season "+seasonNum+" — [e]xport  [b]ack"))
		raw, ok := s.read(s.ps("season " + seasonNum))
		if !ok || isQuit(raw) { return }
		if isBack(raw) { return }
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "e", "export":
			s.screenExportForFn("season_"+tvID+"_"+seasonNum, func(fmt2, fname string) error {
				return cmdSeason(s.apiKey, []string{tvID, seasonNum, "--export", fmt2, "--out", fname, "--lang", s.lang})
			})
		}
	}
}

// ── Images screens ────────────────────────────────────────────────────────────

func (s *Session) screenImages(args []string) {
	if len(args) >= 2 && isID(args[1]) {
		if err := cmdImages(s.apiKey, args); err != nil {
			fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
		}
		return
	}
	fmt.Println()
	mChoice := selectPrompt(s, "media type", []string{"Movie", "TV Show", "Person"})
	if mChoice < 0 { return }
	mtypes := []string{"movie", "tv", "person"}
	mtype := mtypes[mChoice]
	id := plainPrompt("ID:")
	if id == "" { return }
	s.screenImagesFor(mtype, id)
}

func (s *Session) screenImagesFor(mtype, id string) {
	fmt.Println()
	tChoice := selectPrompt(s, "image type", []string{
		"All types",
		"Posters only",
		"Backdrops only",
		"Logos only",
		"Profiles only",
	})
	if tChoice < 0 { return }
	typeMap := map[int]string{0:"", 1:"poster", 2:"backdrop", 3:"logo", 4:"profile"}
	imgType := typeMap[tChoice]

	sChoice := selectPrompt(s, "size", []string{
		"original  (best quality)",
		"w1280     (large)",
		"w780      (medium-large)",
		"w500      (medium)",
		"w185      (thumbnail)",
	})
	sizes := []string{"original", "w1280", "w780", "w500", "w185"}
	sz := "original"
	if sChoice >= 0 { sz = sizes[sChoice] }

	args := []string{mtype, id, "--size", sz}
	if imgType != "" { args = append(args, "--type", imgType) }

	eChoice := selectPrompt(s, "also export?", []string{
		"Just display",
		"Export as JSON",
		"Export as YAML",
		"Export as CSV",
		"Export as TOML",
	})
	fmts := []string{"", "json", "yaml", "csv", "toml"}
	if eChoice > 0 {
		dflt := fmt.Sprintf("images_%s_%s.%s", mtype, id, fmts[eChoice])
		fname := plainPrompt(fmt.Sprintf("Filename (default: %s):", dflt))
		if fname == "" { fname = dflt }
		args = append(args, "--export", fmts[eChoice], "--out", fname)
	}

	if err := cmdImages(s.apiKey, args); err != nil {
		fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
	}
}

// ── Download screens ──────────────────────────────────────────────────────────

func (s *Session) screenDownload(args []string) {
	if len(args) >= 2 && isID(args[1]) {
		if err := cmdDownload(s.apiKey, args); err != nil {
			fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
		}
		return
	}
	fmt.Println()
	mChoice := selectPrompt(s, "media type", []string{"Movie", "TV Show", "Person"})
	if mChoice < 0 { return }
	mtypes := []string{"movie", "tv", "person"}
	mtype := mtypes[mChoice]
	id := plainPrompt("ID:")
	if id == "" { return }
	s.screenDownloadFor(mtype, id)
}

func (s *Session) screenDownloadFor(mtype, id string) {
	fmt.Println()
	tChoice := selectPrompt(s, "image type", []string{
		"All types",
		"Posters only",
		"Backdrops only",
		"Logos only",
	})
	if tChoice < 0 { return }
	typeMap := map[int]string{0:"all", 1:"poster", 2:"backdrop", 3:"logo"}
	imgType := typeMap[tChoice]

	sChoice := selectPrompt(s, "size", []string{
		"original  (best quality)",
		"w1280     (large)",
		"w500      (medium)",
		"w342      (small)",
	})
	sizes := []string{"original", "w1280", "w500", "w342"}
	sz := "original"
	if sChoice >= 0 { sz = sizes[sChoice] }

	dfltDir := fmt.Sprintf("./%s_%s_images", mtype, id)
	dir := plainPrompt(fmt.Sprintf("Download folder (default: %s):", dfltDir))
	if dir == "" { dir = dfltDir }

	limitStr := plainPrompt("Max images (default: all, e.g. 10):")
	args := []string{mtype, id, "--type", imgType, "--size", sz, "--dir", dir}
	if limitStr != "" {
		if _, err := strconv.Atoi(limitStr); err == nil {
			args = append(args, "--limit", limitStr)
		}
	}

	fmt.Printf("\n  %s Downloading to %s …\n", c(cyan, "→"), dir)
	if err := cmdDownload(s.apiKey, args); err != nil {
		fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
	}
}

// ── Videos screen ─────────────────────────────────────────────────────────────

func (s *Session) screenVideos(args []string) {
	if len(args) >= 2 {
		if err := cmdVideos(s.apiKey, args); err != nil {
			fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
		}
		return
	}
	fmt.Println()
	mChoice := selectPrompt(s, "media type", []string{"Movie", "TV Show"})
	if mChoice < 0 { return }
	mtypes := []string{"movie", "tv"}
	id := plainPrompt("ID:")
	if id == "" { return }
	if err := cmdVideos(s.apiKey, []string{mtypes[mChoice], id}); err != nil {
		fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
	}
}

// ── Reviews screen ────────────────────────────────────────────────────────────

func (s *Session) screenReviews(args []string) {
	if len(args) >= 2 {
		if err := cmdReviews(s.apiKey, args); err != nil {
			fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
		}
		return
	}
	fmt.Println()
	mChoice := selectPrompt(s, "media type", []string{"Movie", "TV Show"})
	if mChoice < 0 { return }
	mtypes := []string{"movie", "tv"}
	id := plainPrompt("ID:")
	if id == "" { return }
	page := plainPrompt("Page (default 1):")
	if page == "" { page = "1" }
	if err := cmdReviews(s.apiKey, []string{mtypes[mChoice], id, "--page", page}); err != nil {
		fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
	}
}

// ── Export screens ────────────────────────────────────────────────────────────

func (s *Session) screenExportWizard() {
	fmt.Println()
	choice := selectPrompt(s, "export what", []string{
		"Movie",
		"TV Show",
		"Person",
		"Season",
	})
	if choice < 0 { return }

	switch choice {
	case 0:
		id := plainPrompt("Movie ID:")
		if id != "" { s.screenExportFor("movie", id) }
	case 1:
		id := plainPrompt("TV Show ID:")
		if id != "" { s.screenExportFor("tv", id) }
	case 2:
		id := plainPrompt("Person ID:")
		if id != "" { s.screenExportFor("person", id) }
	case 3:
		tvID := plainPrompt("TV Show ID:")
		sn := plainPrompt("Season number:")
		if tvID != "" && sn != "" {
			s.screenExportForFn("season_"+tvID+"_"+sn, func(fmt2, fname string) error {
				return cmdSeason(s.apiKey, []string{tvID, sn, "--export", fmt2, "--out", fname, "--lang", s.lang})
			})
		}
	}
}

// screenExportFor handles the standard movie/tv/person export flow
func (s *Session) screenExportFor(mtype, id string) {
	s.screenExportForFn(mtype+"_"+id, func(fmt2, fname string) error {
		flags := []string{id, "--export", fmt2, "--out", fname, "--lang", s.lang, "--region", s.region}
		switch mtype {
		case "movie":  return cmdMovie(s.apiKey, flags)
		case "tv":     return cmdTV(s.apiKey, flags)
		case "person": return cmdPerson(s.apiKey, flags)
		}
		return fmt.Errorf("unknown type: %s", mtype)
	})
}

func (s *Session) screenExportForFn(base string, run func(string, string) error) {
	fmt.Println()
	fmtChoice := selectPrompt(s, "format", []string{"JSON", "YAML", "TOML"})
	if fmtChoice < 0 { return }
	fmts := []string{"json", "yaml", "toml"}
	fmt2 := fmts[fmtChoice]
	dflt := base + "." + fmt2
	fname := plainPrompt(fmt.Sprintf("Filename (default: %s):", dflt))
	if fname == "" { fname = dflt }
	fmt.Printf("  %s Exporting…\n", c(dim, "→"))
	if err := run(fmt2, fname); err != nil {
		fmt.Printf("  %s %s\n", c(red, "✗"), err.Error())
	}
}

// ── Inline command handling (from inside a detail screen) ─────────────────────

// handleInline lets users run any top-level command while inside a detail view
// without going back to the outer loop.
func (s *Session) handleInline(fields []string) {
	cmd := strings.ToLower(fields[0])
	rest := fields[1:]
	switch cmd {
	case "search", "s", "find":
		s.screenSearch(rest)
	case "movie", "m":
		s.screenMediaEntry("movie", rest)
	case "tv", "t", "show":
		s.screenMediaEntry("tv", rest)
	case "person", "p":
		s.screenMediaEntry("person", rest)
	case "trending", "hot":
		s.screenTrending(rest)
	case "images", "img", "i":
		s.screenImages(rest)
	case "download", "dl", "d":
		s.screenDownload(rest)
	case "videos", "v":
		s.screenVideos(rest)
	case "reviews", "r":
		s.screenReviews(rest)
	case "season":
		s.screenSeason(rest)
	case "lang":
		if len(rest) > 0 { s.lang = rest[0]; fmt.Printf("  %s Language → %s\n", c(green, "✓"), s.lang) }
	case "region":
		if len(rest) > 0 { s.region = strings.ToUpper(rest[0]); fmt.Printf("  %s Region → %s\n", c(green, "✓"), s.region) }
	case "help", "?":
		printInteractiveHelp()
	default:
		fmt.Printf("  %s Unknown: %s\n", c(yellow, "?"), c(bold, cmd))
	}
}

// ── History ───────────────────────────────────────────────────────────────────

func (s *Session) showHistory() {
	if len(s.history) == 0 {
		fmt.Println(c(dim, "\n  (no history yet)\n"))
		return
	}
	printHeader("Command History")
	for i, h := range s.history {
		fmt.Printf("  %s  %s\n", c(dim, fmt.Sprintf("%3d", i+1)), h)
	}
	fmt.Println()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func isID(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func parseSeasonShorthand(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "season") {
		rest := strings.TrimSpace(strings.TrimPrefix(s, "season"))
		if _, err := strconv.Atoi(rest); err == nil { return rest }
	}
	if strings.HasPrefix(s, "s") {
		rest := strings.TrimSpace(strings.TrimPrefix(s, "s"))
		if _, err := strconv.Atoi(rest); err == nil { return rest }
	}
	return ""
}

func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil { return false }
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// ── Interactive help ──────────────────────────────────────────────────────────

func printInteractiveHelp() {
	fmt.Println(c(bold, "\n  ── Commands ──────────────────────────────────────────────────"))
	rows := [][]string{
		{"s [query]",     "Search  (prompts for query if omitted)"},
		{"m [id|query]",  "Movie details  (ID or search query)"},
		{"tv [id|query]", "TV show details"},
		{"p [id|query]",  "Person details"},
		{"hot",           "Trending now"},
		{"i [type] [id]", "Images"},
		{"d [type] [id]", "Download images"},
		{"v [type] [id]", "Videos / trailers"},
		{"r [type] [id]", "Reviews"},
		{"season [id] N", "TV season + episodes"},
		{"e",             "Export wizard"},
		{"lang [code]",   "Set language   e.g. id-ID  ja-JP  fr-FR"},
		{"region [code]", "Set region     e.g. ID  GB  JP"},
		{"hist",          "Command history  (↑↓ to scroll)"},
		{"clear",         "Clear screen"},
		{"q",             "Quit"},
	}
	for _, row := range rows {
		fmt.Printf("  %-20s %s\n", c(cyan, row[0]), c(dim, row[1]))
	}
	fmt.Println()
	fmt.Println(c(dim, "  Inside any detail view you can still type commands like  s Alien  or  m 123"))
	fmt.Println(c(dim, "  TV show screens: type s1 / s2 / season 3 to open that season directly"))
	fmt.Println()
}
