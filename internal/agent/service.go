package agent

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"
)

type agentService struct{}

func NewAgentService() AgentService {
	return &agentService{}
}

func (s *agentService) Think(agent *CharacterAgent, sceneContext map[string]any) (*AgentDecision, error) {
	intent, err := s.DecideIntent(agent, sceneContext)
	if err != nil {
		return nil, err
	}
	actionType, actionDesc, err := s.GenerateAction(agent, intent)
	if err != nil {
		return nil, err
	}
	targetID := uuid.Nil
	if intent == IntentAccuse || intent == IntentAttack || intent == IntentPersuade {
		for id := range agent.Relationships {
			targetID, _ = uuid.Parse(id)
			break
		}
	}
	var dialogue string
	if intent != IntentHide && intent != IntentAttack {
		dialogue, _ = s.GenerateDialogue(agent, intent, targetID)
	}
	return &AgentDecision{
		CharacterID: agent.ID,
		Intent:      intent,
		Action:      actionType,
		ActionDesc:  actionDesc,
		TargetID:    targetID,
		Emotion:     agent.Emotion,
		Dialogue:    dialogue,
		Confidence:  calculateConfidence(agent),
	}, nil
}

func (s *agentService) DecideIntent(agent *CharacterAgent, context map[string]any) (Intent, error) {
	scores := s.scoreIntents(agent, context)
	if len(scores) == 0 {
		return IntentQuestion, nil
	}
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Total > scores[j].Total
	})
	return scores[0].Intent, nil
}

func (s *agentService) ScoreActions(agent *CharacterAgent, context map[string]any) ([]ActionScore, error) {
	return s.scoreIntents(agent, context), nil
}

func (s *agentService) GenerateAction(agent *CharacterAgent, intent Intent) (ActionType, string, error) {
	switch intent {
	case IntentAttack:
		return ActionAttack, "attacks", nil
	case IntentHide:
		return ActionHide, "hides", nil
	case IntentThreaten, IntentAccuse:
		return ActionSpeak, "steps forward", nil
	case IntentFlirt, IntentSupport:
		return ActionInteract, "approaches", nil
	default:
		return ActionSpeak, "speaks", nil
	}
}

func (s *agentService) GenerateDialogue(agent *CharacterAgent, intent Intent, targetID uuid.UUID) (string, error) {
	switch intent {
	case IntentThreaten:
		return agent.Name + " issues a threat.", nil
	case IntentAccuse:
		return agent.Name + " makes an accusation.", nil
	case IntentLie:
		return agent.Name + " speaks deceptively.", nil
	case IntentReveal:
		return agent.Name + " reveals something important.", nil
	case IntentSupport:
		return agent.Name + " offers support.", nil
	case IntentQuestion:
		return agent.Name + " asks a question.", nil
	default:
		return agent.Name + " speaks.", nil
	}
}

func (s *agentService) scoreIntents(agent *CharacterAgent, context map[string]any) []ActionScore {
	candidates := []Intent{
		IntentQuestion, IntentReveal, IntentSupport,
		IntentAccuse, IntentThreaten, IntentLie,
		IntentPersuade, IntentHide, IntentAttack,
	}
	var scores []ActionScore
	for _, intent := range candidates {
		score := ActionScore{
			Intent:        intent,
			GoalAlignment: calcGoalAlignment(agent, intent),
			Risk:          calcRisk(agent, intent),
			Reward:        calcReward(intent),
			EmotionBias:   calcEmotionBias(agent, intent),
		}
		score.Total = score.GoalAlignment + score.Reward - score.Risk + score.EmotionBias
		if score.Total > -5 {
			scores = append(scores, score)
		}
	}
	return scores
}

func calcGoalAlignment(agent *CharacterAgent, intent Intent) float64 {
	for _, g := range agent.Goals {
		if g.Status != "active" {
			continue
		}
		switch intent {
		case IntentQuestion, IntentInvestigate:
			if contains(g.Description, "find") || contains(g.Description, "truth") {
				return g.Priority * 0.1
			}
		case IntentAttack:
			if contains(g.Description, "defeat") || contains(g.Description, "destroy") {
				return g.Priority * 0.1
			}
		case IntentSupport:
			if contains(g.Description, "protect") || contains(g.Description, "help") {
				return g.Priority * 0.1
			}
		case IntentReveal:
			if contains(g.Description, "reveal") || contains(g.Description, "expose") {
				return g.Priority * 0.1
			}
		}
	}
	return 0
}

func calcRisk(agent *CharacterAgent, intent Intent) float64 {
	switch intent {
	case IntentAttack, IntentAccuse:
		return 3.0 - agent.Energy*0.02
	case IntentReveal:
		return 2.0 - agent.Confidence()
	case IntentLie:
		return 2.0
	default:
		return 0.5
	}
}

func calcReward(intent Intent) float64 {
	switch intent {
	case IntentReveal:
		return 4.0
	case IntentAttack:
		return 3.0
	case IntentSupport:
		return 2.0
	case IntentQuestion:
		return 1.0
	default:
		return 1.5
	}
}

func calcEmotionBias(agent *CharacterAgent, intent Intent) float64 {
	bias := 0.0
	switch agent.Emotion {
	case "anger":
		if intent == IntentAttack || intent == IntentAccuse || intent == IntentThreaten {
			bias = agent.Intensity * 0.05
		}
	case "fear":
		if intent == IntentHide || intent == IntentDefend {
			bias = agent.Intensity * 0.05
		}
	case "joy":
		if intent == IntentSupport || intent == IntentFlirt {
			bias = agent.Intensity * 0.05
		}
	case "sadness":
		if intent == IntentHide || intent == IntentSupport {
			bias = agent.Intensity * 0.03
		}
	case "surprise":
		if intent == IntentQuestion || intent == IntentInvestigate {
			bias = agent.Intensity * 0.04
		}
	}
	if agent.Stress > 70 {
		bias += 0.5
	}
	if agent.Energy < 20 {
		bias -= 1.0
	}
	return bias
}

func calculateConfidence(agent *CharacterAgent) float64 {
	base := 50.0
	base += agent.Intensity * 0.2
	base -= agent.Stress * 0.1
	if base < 0 {
		base = 0
	}
	if base > 100 {
		base = 100
	}
	return base + (rand.Float64()-0.5)*10
}

func (a *CharacterAgent) Confidence() float64 {
	return 50.0 + a.Intensity*0.2 - a.Stress*0.1
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSub(s, substr)
}

func containsSub(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

type directorService struct{}

func NewDirectorService() DirectorService {
	return &directorService{}
}

func (d *directorService) Direct(sceneID uuid.UUID, characters []CharacterAgent, context map[string]any) (*DirectorDecision, error) {
	intensity := 0.0
	for _, c := range characters {
		intensity += c.Intensity
	}
	avgIntensity := intensity / math.Max(1, float64(len(characters)))

	tension := avgIntensity * 0.01
	if tension > 1.0 {
		tension = 1.0
	}
	pacing := d.AdjustPacing("normal", tension)

	var intervention string
	if tension > 0.7 {
		intervention = "increase tension with a reveal"
	} else if tension < 0.3 {
		intervention = "introduce a new clue or complication"
	} else {
		intervention = "let the scene develop naturally"
	}

	return &DirectorDecision{
		SceneID: sceneID,
		Note: DirectorNote{
			Intervention:      intervention,
			TensionAdjustment: (tension - 0.5) * 0.2,
			Pacing:            pacing,
		},
		CreatedAt: time.Now(),
	}, nil
}

//lint:ignore U1000 used via AdjustPacing
func (d *directorService) adjustPacingStr(currentPacing string, tension float64) string {
	return d.AdjustPacing(currentPacing, tension)
}

func (d *directorService) AdjustPacing(currentPacing string, tension float64) string {
	if tension > 0.8 {
		return "fast"
	} else if tension > 0.5 {
		return "moderate"
	} else if tension < 0.3 {
		return "slow"
	}
	return currentPacing
}

func (d *directorService) SuggestIntervention(characters []CharacterAgent, context map[string]any) string {
	if len(characters) == 0 {
		return ""
	}
	avgStress := 0.0
	for _, c := range characters {
		avgStress += c.Stress
	}
	avgStress /= float64(len(characters))
	if avgStress > 70 {
		return "a character lashes out unexpectedly"
	}
	return ""
}
