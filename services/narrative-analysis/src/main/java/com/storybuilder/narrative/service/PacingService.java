package com.storybuilder.narrative.service;

import com.storybuilder.narrative.model.PacingMetrics;
import org.springframework.stereotype.Service;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.regex.Pattern;

@Service
public class PacingService {

    private static final Pattern PARAGRAPH_SPLIT = Pattern.compile("\\n\\s*\\n");
    private static final Pattern SENTENCE_SPLIT = Pattern.compile("[.!?]+\\s*");
    private static final Pattern DIALOGUE_LINE = Pattern.compile(".*\"[^\"]*\".*");

    public PacingMetrics analyze(String text) {
        if (text == null || text.isBlank()) {
            return new PacingMetrics();
        }

        String[] paragraphs = PARAGRAPH_SPLIT.split(text.trim());
        List<Integer> paragraphLengths = new ArrayList<>();
        List<Integer> sentenceLengths = new ArrayList<>();
        int dialogueWords = 0, narrativeWords = 0;

        for (String para : paragraphs) {
            if (para.isBlank()) continue;
            String[] words = para.split("\\s+");
            paragraphLengths.add(words.length);

            if (DIALOGUE_LINE.matcher(para).matches()) {
                dialogueWords += words.length;
            } else {
                narrativeWords += words.length;
            }

            String[] sentences = SENTENCE_SPLIT.split(para);
            for (String s : sentences) {
                if (!s.isBlank()) {
                    sentenceLengths.add(s.split("\\s+").length);
                }
            }
        }

        int totalWords = dialogueWords + narrativeWords;
        double dialogueRatio = totalWords > 0 ? (double) dialogueWords / totalWords : 0;
        double narrativeRatio = totalWords > 0 ? (double) narrativeWords / totalWords : 0;

        int pCount = paragraphLengths.size();
        double avgParaLen = pCount > 0 ?
            paragraphLengths.stream().mapToInt(Integer::intValue).average().orElse(0) : 0;

        double sentenceVariety = computeSentenceVariety(sentenceLengths);

        PacingMetrics m = new PacingMetrics();
        m.setDialogueRatio(round(dialogueRatio));
        m.setNarrativeRatio(round(narrativeRatio));
        m.setParagraphCount(pCount);
        m.setAverageParagraphLength(round(avgParaLen));
        m.setSentenceVariety(round(sentenceVariety));
        m.setParagraphLengths(paragraphLengths);
        m.setSentenceLengths(sentenceLengths);
        return m;
    }

    private double computeSentenceVariety(List<Integer> lengths) {
        if (lengths.size() < 2) return 0;
        double mean = lengths.stream().mapToInt(Integer::intValue).average().orElse(0);
        double variance = lengths.stream()
            .mapToDouble(l -> Math.pow(l - mean, 2))
            .average()
            .orElse(0);
        return Math.sqrt(variance) / mean;
    }

    private double round(double v) {
        return Math.round(v * 100.0) / 100.0;
    }
}
