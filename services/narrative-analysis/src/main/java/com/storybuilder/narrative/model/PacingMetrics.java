package com.storybuilder.narrative.model;

import java.util.List;

public class PacingMetrics {
    private double dialogueRatio;
    private double narrativeRatio;
    private int paragraphCount;
    private double averageParagraphLength;
    private double sentenceVariety;
    private List<Integer> paragraphLengths;
    private List<Integer> sentenceLengths;

    public double getDialogueRatio() { return dialogueRatio; }
    public void setDialogueRatio(double v) { this.dialogueRatio = v; }
    public double getNarrativeRatio() { return narrativeRatio; }
    public void setNarrativeRatio(double v) { this.narrativeRatio = v; }
    public int getParagraphCount() { return paragraphCount; }
    public void setParagraphCount(int v) { this.paragraphCount = v; }
    public double getAverageParagraphLength() { return averageParagraphLength; }
    public void setAverageParagraphLength(double v) { this.averageParagraphLength = v; }
    public double getSentenceVariety() { return sentenceVariety; }
    public void setSentenceVariety(double v) { this.sentenceVariety = v; }
    public List<Integer> getParagraphLengths() { return paragraphLengths; }
    public void setParagraphLengths(List<Integer> v) { this.paragraphLengths = v; }
    public List<Integer> getSentenceLengths() { return sentenceLengths; }
    public void setSentenceLengths(List<Integer> v) { this.sentenceLengths = v; }
}
