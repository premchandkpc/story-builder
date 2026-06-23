package com.storybuilder.narrative.model;

import java.util.List;

public class SentimentResult {
    private double overallScore;
    private String overallLabel;
    private List<Double> sentenceScores;
    private List<String> sentenceLabels;
    private double positiveRatio;
    private double negativeRatio;
    private double neutralRatio;

    public double getOverallScore() { return overallScore; }
    public void setOverallScore(double v) { this.overallScore = v; }
    public String getOverallLabel() { return overallLabel; }
    public void setOverallLabel(String v) { this.overallLabel = v; }
    public List<Double> getSentenceScores() { return sentenceScores; }
    public void setSentenceScores(List<Double> v) { this.sentenceScores = v; }
    public List<String> getSentenceLabels() { return sentenceLabels; }
    public void setSentenceLabels(List<String> v) { this.sentenceLabels = v; }
    public double getPositiveRatio() { return positiveRatio; }
    public void setPositiveRatio(double v) { this.positiveRatio = v; }
    public double getNegativeRatio() { return negativeRatio; }
    public void setNegativeRatio(double v) { this.negativeRatio = v; }
    public double getNeutralRatio() { return neutralRatio; }
    public void setNeutralRatio(double v) { this.neutralRatio = v; }
}
