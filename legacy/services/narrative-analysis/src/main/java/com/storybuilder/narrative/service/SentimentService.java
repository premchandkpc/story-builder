package com.storybuilder.narrative.service;

import com.storybuilder.narrative.model.SentimentResult;
import org.springframework.stereotype.Service;

import java.util.ArrayList;
import java.util.List;
import java.util.regex.Pattern;

@Service
public class SentimentService {

    private static final Pattern SENTENCE_SPLIT = Pattern.compile("[.!?]+\\s*");

    private static final String[] POSITIVE_WORDS = {
        "good", "great", "excellent", "amazing", "beautiful", "wonderful", "happy", "joy",
        "love", "hope", "peace", "kind", "gentle", "brave", "courageous", "triumph",
        "victory", "success", "celebrate", "glorious", "magnificent", "delight", "ecstasy",
        "bliss", "grace", "elegant", "radiant", "bright", "warm", "generous", "compassion",
        "grateful", "bless", "thrilled", "elated", "optimistic", "passion", "serene",
        "splendid", "sublime", "tender", "vibrant", "wholesome", "zeal", "caring"
    };

    private static final String[] NEGATIVE_WORDS = {
        "bad", "terrible", "awful", "horrible", "hate", "anger", "rage", "sad", "grief",
        "despair", "agony", "pain", "suffer", "cruel", "wicked", "evil", "vile", "harsh",
        "brutal", "vicious", "fear", "terror", "dread", "horror", "panic", "misery",
        "sorrow", "weep", "gloom", "desolate", "tragic", "dismal", "dire", "mournful",
        "wretched", "agonize", "betray", "curse", "damn", "venom", "malice", "spite",
        "ruthless", "sinister", "toxic", "ugly", "dreadful", "appalling", "devastate"
    };

    public SentimentResult analyze(String text) {
        if (text == null || text.isBlank()) {
            return emptyResult();
        }

        String[] sentences = SENTENCE_SPLIT.split(text.trim());
        List<Double> scores = new ArrayList<>();
        List<String> labels = new ArrayList<>();
        int positiveCount = 0, negativeCount = 0, neutralCount = 0;

        for (String sentence : sentences) {
            if (sentence.isBlank()) continue;
            double score = scoreSentence(sentence);
            scores.add(score);
            String label = labelFor(score);
            labels.add(label);
            if ("positive".equals(label)) positiveCount++;
            else if ("negative".equals(label)) negativeCount++;
            else neutralCount++;
        }

        double overall = scores.isEmpty() ? 0 : scores.stream().mapToDouble(Double::doubleValue).average().orElse(0);
        int total = positiveCount + negativeCount + neutralCount;

        SentimentResult r = new SentimentResult();
        r.setOverallScore(round(overall));
        r.setOverallLabel(labelFor(overall));
        r.setSentenceScores(scores.stream().map(this::round).toList());
        r.setSentenceLabels(labels);
        r.setPositiveRatio(total > 0 ? round((double) positiveCount / total) : 0);
        r.setNegativeRatio(total > 0 ? round((double) negativeCount / total) : 0);
        r.setNeutralRatio(total > 0 ? round((double) neutralCount / total) : 0);
        return r;
    }

    private double scoreSentence(String sentence) {
        String[] words = sentence.toLowerCase().replaceAll("[^a-zA-Z\\s]", "").split("\\s+");
        int positive = 0, negative = 0;
        for (String w : words) {
            if (w.isEmpty()) continue;
            for (String p : POSITIVE_WORDS) { if (w.equals(p)) { positive++; break; } }
            for (String n : NEGATIVE_WORDS) { if (w.equals(n)) { negative++; break; } }
        }
        int total = positive + negative;
        if (total == 0) return 0;
        return (double) (positive - negative) / total;
    }

    private String labelFor(double score) {
        if (score > 0.25) return "positive";
        if (score < -0.25) return "negative";
        return "neutral";
    }

    private SentimentResult emptyResult() {
        return new SentimentResult();
    }

    private double round(double v) {
        return Math.round(v * 100.0) / 100.0;
    }
}
