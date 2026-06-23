package com.storybuilder.narrative.model;

public class ReadabilityMetrics {
    private double fleschReadingEase;
    private double fleschKincaidGrade;
    private double colemanLiauIndex;
    private double averageSentenceLength;
    private double averageWordLength;
    private double lexicalDiversity;
    private int wordCount;
    private int sentenceCount;
    private int syllableCount;
    private int characterCount;

    public double getFleschReadingEase() { return fleschReadingEase; }
    public void setFleschReadingEase(double v) { this.fleschReadingEase = v; }
    public double getFleschKincaidGrade() { return fleschKincaidGrade; }
    public void setFleschKincaidGrade(double v) { this.fleschKincaidGrade = v; }
    public double getColemanLiauIndex() { return colemanLiauIndex; }
    public void setColemanLiauIndex(double v) { this.colemanLiauIndex = v; }
    public double getAverageSentenceLength() { return averageSentenceLength; }
    public void setAverageSentenceLength(double v) { this.averageSentenceLength = v; }
    public double getAverageWordLength() { return averageWordLength; }
    public void setAverageWordLength(double v) { this.averageWordLength = v; }
    public double getLexicalDiversity() { return lexicalDiversity; }
    public void setLexicalDiversity(double v) { this.lexicalDiversity = v; }
    public int getWordCount() { return wordCount; }
    public void setWordCount(int v) { this.wordCount = v; }
    public int getSentenceCount() { return sentenceCount; }
    public void setSentenceCount(int v) { this.sentenceCount = v; }
    public int getSyllableCount() { return syllableCount; }
    public void setSyllableCount(int v) { this.syllableCount = v; }
    public int getCharacterCount() { return characterCount; }
    public void setCharacterCount(int v) { this.characterCount = v; }
}
