// Interactive CLI audit dashboard.
// Build: go build -o audit-bin ./audit/cli.go
// Run:   ./audit-bin
package main

import (
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
)

type Finding struct {
	ID, Title, Severity, Status, Desc, Fix, File string
}

var findings = []Finding{
	{"F-01", "genInFlight leak on panic", "critical", "fixed", "Generate() goroutine had no deferred genInFlight.Delete(). If runPipeline panicked, the sceneID was permanently locked.", "Added defer s.genInFlight.Delete(sceneID) as first line inside goroutine.", "internal/service/generation.go:87"},
	{"F-02", "CharState only holds last participant", "critical", "fixed", "params.CharState initialization was inside the character loop, resetting on each iteration.", "Moved map initialization before the character loop.", "internal/service/generation.go:128"},
	{"F-03", "AcceptGeneration race condition", "critical", "fixed", "No concurrency guard on AcceptGeneration — two concurrent calls could interleave read-modify-write cycles.", "Added acceptInFlight sync.Map guard; atomic pass marks all false then sets target true.", "internal/service/generation.go:33,180"},
	{"F-04", "GenerateStory missing character fields", "high", "fixed", "Character records created with only Name and Persona; MoralAlignment, Personality, etc. silently dropped.", "Added explicit field mapping for all StoryOutlineCharacter fields.", "internal/api/stories.go:120-154"},
	{"F-05", "GenerateStory skips Location creation", "high", "fixed", "Beat LocationName field ignored; no Location records created, scenes had no LocationRef.", "Created LocationRepository/Service/types; GenerateStory populates LocationRef.", "internal/api/stories.go"},
	{"F-06", "GenerateStory missing TimelinePosition", "high", "fixed", "Scenes created with TimelinePosition=0; topology ordering unreliable.", "Set scene.TimelinePosition = i+1 during scene creation.", "internal/api/stories.go:154"},
	{"F-07", "Entire Location system absent", "high", "fixed", "No Location domain type, repository, service, or API handlers.", "Created domain.Location, LocationRepo, LocationService, CRUD handlers, wiring.", "internal/domain/location.go"},
	{"F-08", "Topology sorts by insertion order", "high", "fixed", "V2Topology returned nodes in MongoDB insertion order, not story order.", "Sort by TimelinePosition before returning.", "internal/api/nodes.go:198"},
	{"F-09", "Dual topology endpoint confusion", "medium", "fixed", "Unused Topology() handler with different response shape from real endpoint.", "Removed dead Topology() handler.", "internal/api/scenes.go:79-96"},
	{"F-10", "ExtractStateWorker ignores location/mood", "high", "fixed", "CharacterState.Location and .Mood left empty despite delta having them.", "Set state fields directly from extraction deltas.", "internal/worker/extract.go:53-58"},
	{"F-11", "14-param constructor", "medium", "fixed", "NewGenerationService took 14 positional args; transposed args compile but corrupt data.", "Refactored to GenerationServiceConfig struct with named fields.", "internal/service/generation.go:16-58"},
	{"F-12", "Missing pipeline step observability", "medium", "fixed", "runPipeline ran 6 steps but persisted no status; operators couldn't diagnose failures.", "Added StepStatus map to Generation; pipeline updates running/done/failed per step.", "internal/domain/scene.go:46"},
	{"F-13", "go.mod minimum Go version", "low", "wontfix", "Already at go 1.26.4 (context.WithoutCancel needs 1.21+).", "No change needed.", "go.mod:3"},
	{"F-14", "docs/schema.md Props type", "low", "fixed", "Props documented as map, actually []string.", "Corrected type in docs.", "docs/schema.md:261"},
	{"F-15", "Generate called with empty context", "critical", "fixed", "runPipeline received only beat_intent/pov/tone/target_words — no character or location data.", "Added buildPromptParams() fetching character cards, states, location, summary.", "internal/service/generation.go:94-178"},
}

var sevColor = map[string]string{
	"critical": "\033[31m",
	"high":     "\033[38;5;208m",
	"medium":   "\033[33m",
	"low":      "\033[90m",
}

var sevOrder = map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; fmt.Println("\033[?25h"); os.Exit(0) }()

	filter := "all"
	selected := 0

	items := func() []Finding { return applyFilter(filter) }

	for {
		fmt.Print("\033[H\033[J\033[?25l")
		render(filter, selected, items())
		b := readKey()
		switch b {
		case 'q', 27:
			fmt.Print("\033[?25h")
			return
		case 'j', 66:
			if selected < len(items())-1 {
				selected++
			}
		case 'k', 65:
			if selected > 0 {
				selected--
			}
		case '1':
			filter = "all"; selected = 0
		case '2':
			filter = "critical"; selected = 0
		case '3':
			filter = "high"; selected = 0
		case '4':
			filter = "medium"; selected = 0
		case '5':
			filter = "low"; selected = 0
		case '6':
			filter = "fixed"; selected = 0
		case '7':
			filter = "wontfix"; selected = 0
		case 10:
			if len(items()) > 0 {
				detailView(items()[selected])
			}
		}
	}
}

func applyFilter(f string) []Finding {
	var out []Finding
	for _, f2 := range findings {
		if f == "all" || f2.Severity == f || f2.Status == f {
			out = append(out, f2)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return sevOrder[out[i].Severity] < sevOrder[out[j].Severity]
	})
	return out
}

func render(filter string, selected int, items []Finding) {
	reset := "\033[0m"
	bold := "\033[1m"
	dim := "\033[2m"

	fmt.Printf("\033[33m%s\u00a0Code Audit\033[0m  %s%d findings%s\n", bold, dim, len(findings), reset)
	fmt.Printf("\033[90mStory Builder — internal code review\033[0m\n\n")

	counts := func(sev string) int {
		n := 0
		for _, f := range findings {
			match := sev == "all" || f.Severity == sev || f.Status == sev
			if match {
				n++
			}
		}
		return n
	}

	fmt.Printf("\033[90m[1]\033[0m All:%d  \033[31m[2]\033[0m Crit:%d  \033[38;5;208m[3]\033[0m High:%d  \033[33m[4]\033[0m Med:%d  \033[90m[5]\033[0m Low:%d  ", counts("all"), counts("critical"), counts("high"), counts("medium"), counts("low"))
	fmt.Printf("\033[32m[6]\033[0m Fixed:%d  \033[90m[7]\033[0m Wontfix:%d\n\n", counts("fixed"), counts("wontfix"))

	fmt.Printf("  %s%-5s %-45s %-10s %s%s\033[0m\n", bold, "ID", "Title", "Severity", "Status", reset)
	fmt.Printf("  %s\n", strings.Repeat("─", 72))

	for i, f := range items {
		cursor := " "
		style := dim
		if i == selected {
			cursor = "▸"
			style = bold + "\033[37m"
		}
		sc := sevColor[f.Severity]
		stat := "✓"
		sc2 := "\033[32m"
		if f.Status == "wontfix" {
			stat = "–"
			sc2 = "\033[90m"
		}
		title := f.Title
		if len(title) > 42 {
			title = title[:42] + "…"
		}
		fmt.Printf("%s %s %s%-5s%s %-45s %s%-10s%s %s%s%s\033[0m\n",
			cursor, style, sc, f.ID, reset+style, title,
			sc, f.Severity, reset+style,
			sc2, stat, reset+style)
	}

	fmt.Printf("\n%s\033[90m↑↓/jk navigate · ↩ detail · 1-7 filter · q quit\033[0m\n", reset)
}

func detailView(f Finding) {
	fmt.Print("\033[H\033[J")
	reset := "\033[0m"
	bold := "\033[1m"
	sc := sevColor[f.Severity]

	fmt.Printf("%s%s %s%s%s\n\n", bold, f.ID, sc, f.Severity, reset)
	fmt.Printf("%s%s%s\n\n", bold, f.Title, reset)
	fmt.Printf("  %sDescription:%s %s\n\n", bold, reset, f.Desc)
	fmt.Printf("  %sFix:%s %s\n\n", bold, reset, f.Fix)
	fmt.Printf("  %sFile:%s %s\n", bold, reset, f.File)
	fmt.Printf("  %sStatus:%s %s\n", bold, reset, f.Status)

	fmt.Printf("\n\n\033[90mPress ↩ to go back, q to quit\033[0m")
	for {
		b := readKey()
		switch b {
		case 'q', 27:
			fmt.Print("\033[?25h")
			os.Exit(0)
		case 10:
			return
		}
	}
}

func readKey() byte {
	buf := make([]byte, 3)
	n, _ := os.Stdin.Read(buf)
	if n == 0 {
		return 0
	}
	if buf[0] == 27 && n > 1 {
		return buf[2] // arrow keys send ESC [ A/B/C/D
	}
	return buf[0]
}
