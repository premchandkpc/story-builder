package com.storybuilder.narrative.model;

public class CharacterMention {
    private String characterName;
    private int mentionCount;
    private double mentionDensity;

    public CharacterMention() {}
    public CharacterMention(String characterName, int mentionCount, double mentionDensity) {
        this.characterName = characterName;
        this.mentionCount = mentionCount;
        this.mentionDensity = mentionDensity;
    }

    public String getCharacterName() { return characterName; }
    public void setCharacterName(String v) { this.characterName = v; }
    public int getMentionCount() { return mentionCount; }
    public void setMentionCount(int v) { this.mentionCount = v; }
    public double getMentionDensity() { return mentionDensity; }
    public void setMentionDensity(double v) { this.mentionDensity = v; }
}
