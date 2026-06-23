package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/repository"
)

type ContextBuilder struct {
	bibleRepo   repository.BibleRepository
	storyRepo   repository.StoryRepository
	charRepo    repository.CharacterRepository
	stateRepo   repository.CharacterStateRepository
	locRepo     repository.LocationRepository
	memRepo     repository.MemoryRepository
	sumRepo     repository.SummaryRepository
	tlRepo      repository.TimelineRepository
}

func NewContextBuilder(
	bibleRepo repository.BibleRepository,
	storyRepo repository.StoryRepository,
	charRepo repository.CharacterRepository,
	stateRepo repository.CharacterStateRepository,
	locRepo repository.LocationRepository,
	memRepo repository.MemoryRepository,
	sumRepo repository.SummaryRepository,
	tlRepo repository.TimelineRepository,
) *ContextBuilder {
	return &ContextBuilder{
		bibleRepo: bibleRepo,
		storyRepo: storyRepo,
		charRepo:  charRepo,
		stateRepo: stateRepo,
		locRepo:   locRepo,
		memRepo:   memRepo,
		sumRepo:   sumRepo,
		tlRepo:    tlRepo,
	}
}

type BuiltContext struct {
	Params         llm.PromptParams
	CanonXML       string
	CharStateXML   string
	BranchSummary  string
	CharacterNames []string
	CharNameToID   map[string]string
}

func (b *ContextBuilder) Build(ctx context.Context, scene *domain.Scene) (*BuiltContext, error) {
	built := &BuiltContext{
		Params: llm.PromptParams{
			BeatIntent:  scene.BeatIntent,
			POV:         scene.POV,
			Tone:        scene.Tone,
			TargetWords: scene.TargetWords,
		},
	}

	allChars, err := b.charRepo.ListByStory(ctx, scene.StoryID)
	if err != nil {
		slog.Warn("context: list characters", "storyId", scene.StoryID, "error", err)
		allChars = nil
	}

	charNameToID := b.buildCharacterCards(scene, allChars, built)
	built.CharNameToID = charNameToID
	built.CharacterNames = make([]string, 0, len(charNameToID))
	for name := range charNameToID {
		built.CharacterNames = append(built.CharacterNames, name)
	}

	b.buildCharacterStates(ctx, scene, allChars, built)

	b.buildLocationContext(ctx, scene, built)

	b.buildBibleContext(ctx, scene, built)

	b.buildMemoryContext(ctx, scene, allChars, built)

	b.buildTimelineContext(ctx, scene, built)

	b.buildSummaryContext(ctx, scene, built)

	b.buildBlueprintContext(ctx, scene, built)

	built.CanonXML = b.buildCanonXML(built)
	built.CharStateXML = b.buildCharStateXML(built)
	built.BranchSummary = b.buildBranchSummary(built)

	return built, nil
}

func (b *ContextBuilder) buildCharacterCards(scene *domain.Scene, allChars []*domain.Character, built *BuiltContext) map[string]string {
	charNameToID := make(map[string]string, len(scene.Participants))
	participantIDs := make(map[string]bool, len(scene.Participants))
	for _, pid := range scene.Participants {
		participantIDs[pid] = true
	}

	for _, c := range allChars {
		if !participantIDs[c.ID] && !participantIDs[c.CharID] {
			continue
		}
		charNameToID[c.Name] = c.CharID

		card := llm.CharacterCard{
			Name:         c.Name,
			Description:  c.Persona,
			Type:         "character",
			Traits:       c.Traits,
			VoiceSamples: c.VoiceSamples,
			Want:         c.Want,
			Need:         c.Need,
			FalseBelief:  c.FalseBelief,
			ArcType:      c.ArcType,
		}
		if c.Backstory != "" {
			card.Description = c.Persona + ". " + c.Backstory
		}
		if c.Relationships != nil {
			card.Relationships = c.Relationships
		}
		if len(c.RelData) > 0 {
			relData := make(map[string]llm.NumericRelationships, len(c.RelData))
			for _, r := range c.RelData {
				relData[r.TargetName] = llm.NumericRelationships{
					Trust:     r.Trust,
					Respect:   r.Respect,
					Fear:      r.Fear,
					Affection: r.Affection,
				}
			}
			card.RelData = relData
		}
		built.Params.CharacterCards = append(built.Params.CharacterCards, card)
	}
	return charNameToID
}

func (b *ContextBuilder) buildCharacterStates(ctx context.Context, scene *domain.Scene, allChars []*domain.Character, built *BuiltContext) {
	built.Params.CharState = make(map[string]interface{})

	nameByCharID := make(map[string]string, len(allChars))
	for _, c := range allChars {
		nameByCharID[c.CharID] = c.Name
		nameByCharID[c.ID] = c.Name
	}

	for _, c := range allChars {
		charID := c.CharID
		if charID == "" {
			charID = c.ID
		}

		states, err := b.stateRepo.ListByCharacter(ctx, charID)
		if err != nil || len(states) == 0 {
			continue
		}

		latest := states[len(states)-1]
		cs := llm.CharacterState{
			StoryID:       latest.StoryID,
			CharacterID:   latest.CharacterID,
			AsOfScene:     latest.SceneID,
			Location:      latest.Location,
			Mood:          latest.Mood,
			Knows:         latest.Knowledge,
			DoesNotKnow:   latest.DoesNotKnow,
			Items:         latest.Inventory,
		}
		if latest.Relationships != nil {
			cs.Relationships = latest.Relationships
		}
		if m, ok := latest.Changes["learned"].([]string); ok {
			cs.Knows = append(cs.Knows, m...)
		}
		if m, ok := latest.Changes["does_not_know"].([]string); ok {
			cs.DoesNotKnow = append(cs.DoesNotKnow, m...)
		}

		name, ok := nameByCharID[charID]
		if !ok {
			name = c.Name
		}
		built.Params.CharState[name] = cs
	}
}

func (b *ContextBuilder) buildLocationContext(ctx context.Context, scene *domain.Scene, built *BuiltContext) {
	if scene.LocationRef == "" {
		return
	}

	locs, err := b.locRepo.ListByStory(ctx, scene.StoryID)
	if err != nil || len(locs) == 0 {
		return
	}

	locMap := make(map[string]*domain.Location, len(locs))
	for _, loc := range locs {
		locMap[loc.ID] = loc
		locMap[loc.Name] = loc
	}

	current := locMap[scene.LocationRef]
	if current == nil {
		return
	}

	var hierarchyParts []string
	var ancestors []string

	walk := current
	for walk != nil {
		hierarchyParts = append([]string{fmt.Sprintf("%s (%s)", walk.Name, walk.LocType)}, hierarchyParts...)
		if walk.ParentID != "" {
			parent := locMap[walk.ParentID]
			if parent != nil {
				ancestors = append([]string{fmt.Sprintf("%s: %s", parent.Name, parent.Description)}, ancestors...)
			}
		}
		if walk.ParentID == "" || locMap[walk.ParentID] == nil {
			break
		}
		walk = locMap[walk.ParentID]
	}

	lore := built.Params.Lore
	if current.Description != "" {
		lore = append(lore, fmt.Sprintf("Location (%s): %s", current.Name, current.Description))
	}
	if len(current.Props) > 0 {
		lore = append(lore, fmt.Sprintf("Props in %s: %s", current.Name, strings.Join(current.Props, ", ")))
	}
	if current.Atmosphere != "" {
		lore = append(lore, fmt.Sprintf("Atmosphere in %s: %s", current.Name, current.Atmosphere))
	}
	if len(ancestors) > 0 {
		lore = append(lore, "Location context (outer→inner): "+strings.Join(ancestors, " → "))
	}
	built.Params.Lore = lore

	built.Params.LocationCard = &llm.CharacterCard{
		Name:        current.Name,
		Description: current.Description,
		Type:        "location",
		Props:       current.Props,
	}
}

func (b *ContextBuilder) buildBibleContext(ctx context.Context, scene *domain.Scene, built *BuiltContext) {
	bible, err := b.bibleRepo.GetByStory(ctx, scene.StoryID)
	if err != nil || bible == nil {
		return
	}

	lore := built.Params.Lore

	if bible.World != "" {
		lore = append(lore, fmt.Sprintf("World setting: %s", bible.World))
	}
	if len(bible.WorldRules) > 0 {
		for _, r := range bible.WorldRules {
			lore = append(lore, fmt.Sprintf("World rule (%s, %s): %s", r.Category, r.Strictness, r.Description))
		}
	}
	if len(bible.MagicSystems) > 0 {
		for _, m := range bible.MagicSystems {
			lore = append(lore, fmt.Sprintf("Magic: %s — source: %s, cost: %s", m.Name, m.Source, m.Cost))
			if len(m.Limitations) > 0 {
				lore = append(lore, fmt.Sprintf("Magic limits: %s", strings.Join(m.Limitations, "; ")))
			}
		}
	}
	if len(bible.Factions) > 0 {
		for _, f := range bible.Factions {
			lore = append(lore, fmt.Sprintf("Faction: %s — %s", f.Name, f.Goal))
			if f.Relations != "" {
				lore = append(lore, fmt.Sprintf("Faction relations: %s", f.Relations))
			}
		}
	}
	if len(bible.Cultures) > 0 {
		for _, c := range bible.Cultures {
			lore = append(lore, fmt.Sprintf("Culture: %s — values: %s, technology: %s", c.Name, strings.Join(c.Values, ", "), c.Technology))
		}
	}
	if bible.CentralTheme != "" {
		lore = append(lore, fmt.Sprintf("Theme: %s", bible.CentralTheme))
	}
	if bible.Tone != "" {
		lore = append(lore, fmt.Sprintf("Global tone: %s", bible.Tone))
	}
	if bible.NarrativeVoice != "" {
		lore = append(lore, fmt.Sprintf("Narrative voice: %s", bible.NarrativeVoice))
	}

	built.Params.Lore = lore
}

func (b *ContextBuilder) buildMemoryContext(ctx context.Context, scene *domain.Scene, allChars []*domain.Character, built *BuiltContext) {
	participantIDs := make(map[string]bool, len(scene.Participants))
	for _, pid := range scene.Participants {
		participantIDs[pid] = true
	}

	nameByID := make(map[string]string, len(allChars))
	for _, c := range allChars {
		nameByID[c.ID] = c.Name
		nameByID[c.CharID] = c.Name
	}

	memories := make(map[string][]string)
	for charID := range participantIDs {
		mems, err := b.memRepo.ListByCharacter(ctx, charID)
		if err != nil || len(mems) == 0 {
			continue
		}
		charName, ok := nameByID[charID]
		if !ok {
			continue
		}

		sort.Slice(mems, func(i, j int) bool {
			return mems[i].Importance > mems[j].Importance
		})
		limit := 10
		if len(mems) < limit {
			limit = len(mems)
		}
		snippets := make([]string, 0, limit)
		for i := 0; i < limit; i++ {
			snippets = append(snippets, truncate(mems[i].Content, 200))
		}
		memories[charName] = snippets
	}
	if len(memories) > 0 {
		built.Params.Memories = memories
	}
}

func (b *ContextBuilder) buildTimelineContext(ctx context.Context, scene *domain.Scene, built *BuiltContext) {
	events, err := b.tlRepo.ListByStory(ctx, scene.StoryID)
	if err != nil || len(events) == 0 {
		return
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Order < events[j].Order
	})

	limit := 20
	if len(events) > limit {
		events = events[len(events)-limit:]
	}

	var timelineParts []string
	for _, e := range events {
		timelineParts = append(timelineParts, fmt.Sprintf("%d. %s (%s)", e.Order, e.Title, e.EventType))
		if e.Description != "" {
			timelineParts = append(timelineParts, fmt.Sprintf("   -> %s", e.Description))
		}
	}

	lore := built.Params.Lore
	lore = append(lore, "=== Story Timeline ===")
	lore = append(lore, timelineParts...)
	built.Params.Lore = lore
}

func (b *ContextBuilder) buildSummaryContext(ctx context.Context, scene *domain.Scene, built *BuiltContext) {
	if existing, _ := b.sumRepo.GetByLevel(ctx, scene.StoryID, "story"); existing != nil {
		built.Params.BranchSummary = existing.Content
		built.Params.Lore = append(built.Params.Lore, "Story summary: "+existing.Content)
	}

	if existing, _ := b.sumRepo.GetByLevel(ctx, scene.StoryID, "scene"); existing != nil {
		lore := built.Params.Lore
		lore = append(lore, "Previous scene summary: "+existing.Content)
		built.Params.Lore = lore
	}
}

func (b *ContextBuilder) buildBlueprintContext(ctx context.Context, scene *domain.Scene, built *BuiltContext) {
	story, err := b.storyRepo.Get(ctx, scene.StoryID)
	if err != nil || story == nil || story.Blueprint == nil {
		return
	}

	bp := story.Blueprint
	lore := built.Params.Lore

	if bp.Premise != "" {
		lore = append(lore, fmt.Sprintf("Story premise: %s", bp.Premise))
	}
	if bp.Theme != "" {
		lore = append(lore, fmt.Sprintf("Theme: %s", bp.Theme))
	}
	if bp.MainConflict != "" {
		lore = append(lore, fmt.Sprintf("Main conflict: %s", bp.MainConflict))
	}
	if len(bp.Acts) > 0 {
		var actParts []string
		for _, a := range bp.Acts {
			actParts = append(actParts, fmt.Sprintf("Act %d: %s", a.Number, a.Summary))
		}
		lore = append(lore, "Acts: "+strings.Join(actParts, " | "))
	}
	if len(bp.PlotThreads) > 0 {
		for _, pt := range bp.PlotThreads {
			lore = append(lore, fmt.Sprintf("Plot thread: %s [%s]", pt.Description, pt.Status))
		}
	}

	built.Params.Lore = lore
}

func (b *ContextBuilder) buildCanonXML(built *BuiltContext) string {
	canon := ""
	for _, card := range built.Params.CharacterCards {
		canon += fmt.Sprintf("<character name=\"%s\">\n", escXML(card.Name))
		canon += fmt.Sprintf("Traits: %v\n", card.Traits)
		canon += fmt.Sprintf("Relationships: %v\n", card.Relationships)
		if len(card.RelData) > 0 {
			canon += "Relationship scores (0-100):\n"
			for target, rel := range card.RelData {
				canon += fmt.Sprintf("  %s → trust:%v respect:%v fear:%v affection:%v\n", escXML(target), rel.Trust, rel.Respect, rel.Fear, rel.Affection)
			}
		}
		if card.Want != "" {
			canon += fmt.Sprintf("Wants: %s\n", escXML(card.Want))
		}
		if card.Need != "" {
			canon += fmt.Sprintf("Needs: %s\n", escXML(card.Need))
		}
		if len(card.VoiceSamples) > 0 {
			canon += "Voice samples:\n"
			for _, v := range card.VoiceSamples {
				canon += fmt.Sprintf("- \"%s\"\n", escXML(v))
			}
		}
		canon += "</character>\n"
	}
	if built.Params.LocationCard != nil {
		canon += fmt.Sprintf("<location name=\"%s\">%s\n", escXML(built.Params.LocationCard.Name), escXML(built.Params.LocationCard.Description))
		canon += fmt.Sprintf("Props available: %v</location>\n", built.Params.LocationCard.Props)
	}
	if len(built.Params.Lore) > 0 {
		canon += "<world_rules>\n"
		for _, l := range built.Params.Lore {
			canon += fmt.Sprintf("- %s\n", escXML(l))
		}
		canon += "</world_rules>"
	}
	return canon
}

func (b *ContextBuilder) buildCharStateXML(built *BuiltContext) string {
	stateBlock := ""
	for char, st := range built.Params.CharState {
		cs, ok := st.(llm.CharacterState)
		if !ok {
			continue
		}
		stateBlock += fmt.Sprintf("%s: at %s, mood %s,\n", escXML(char), escXML(cs.Location), escXML(cs.Mood))
		stateBlock += fmt.Sprintf("knows: %v,\n", cs.Knows)
		if cs.DoesNotKnow != nil {
			stateBlock += fmt.Sprintf("does NOT know: %v\n", cs.DoesNotKnow)
		} else {
			stateBlock += "does NOT know: []\n"
		}
		if len(cs.Items) > 0 {
			stateBlock += fmt.Sprintf("inventory: %v\n", cs.Items)
		}
		if mems, ok := built.Params.Memories[char]; ok && len(mems) > 0 {
			stateBlock += "recent memories:\n"
			for _, m := range mems {
				stateBlock += fmt.Sprintf("- %s\n", escXML(m))
			}
		}
	}
	return stateBlock
}

func (b *ContextBuilder) buildBranchSummary(built *BuiltContext) string {
	return built.Params.BranchSummary
}

func escXML(s string) string {
	return strings.NewReplacer("<", "＜", ">", "＞").Replace(s)
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
