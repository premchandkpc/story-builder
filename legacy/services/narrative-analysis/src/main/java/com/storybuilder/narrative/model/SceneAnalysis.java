package com.storybuilder.narrative.model;

import org.springframework.data.annotation.Id;
import org.springframework.data.mongodb.core.mapping.Document;

import java.time.Instant;
import java.util.List;

@Document(collection = "scene_analysis")
public class SceneAnalysis {
    @Id
    private String id;
    private String sceneId;
    private String storyId;
    private String title;
    private ReadabilityMetrics readability;
    private SentimentResult sentiment;
    private PacingMetrics pacing;
    private List<CharacterMention> characterMentions;
    private Instant createdAt;
    private Instant updatedAt;

    public String getId() { return id; }
    public void setId(String v) { this.id = v; }
    public String getSceneId() { return sceneId; }
    public void setSceneId(String v) { this.sceneId = v; }
    public String getStoryId() { return storyId; }
    public void setStoryId(String v) { this.storyId = v; }
    public String getTitle() { return title; }
    public void setTitle(String v) { this.title = v; }
    public ReadabilityMetrics getReadability() { return readability; }
    public void setReadability(ReadabilityMetrics v) { this.readability = v; }
    public SentimentResult getSentiment() { return sentiment; }
    public void setSentiment(SentimentResult v) { this.sentiment = v; }
    public PacingMetrics getPacing() { return pacing; }
    public void setPacing(PacingMetrics v) { this.pacing = v; }
    public List<CharacterMention> getCharacterMentions() { return characterMentions; }
    public void setCharacterMentions(List<CharacterMention> v) { this.characterMentions = v; }
    public Instant getCreatedAt() { return createdAt; }
    public void setCreatedAt(Instant v) { this.createdAt = v; }
    public Instant getUpdatedAt() { return updatedAt; }
    public void setUpdatedAt(Instant v) { this.updatedAt = v; }
}
