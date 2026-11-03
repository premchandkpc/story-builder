package com.storybuilder.narrative.model;

import java.util.List;

public class AnalysisRequest {
    private String sceneId;
    private String storyId;
    private String title;
    private String content;
    private List<String> characterNames;

    public String getSceneId() { return sceneId; }
    public void setSceneId(String sceneId) { this.sceneId = sceneId; }
    public String getStoryId() { return storyId; }
    public void setStoryId(String storyId) { this.storyId = storyId; }
    public String getTitle() { return title; }
    public void setTitle(String title) { this.title = title; }
    public String getContent() { return content; }
    public void setContent(String content) { this.content = content; }
    public List<String> getCharacterNames() { return characterNames; }
    public void setCharacterNames(List<String> characterNames) { this.characterNames = characterNames; }
}
