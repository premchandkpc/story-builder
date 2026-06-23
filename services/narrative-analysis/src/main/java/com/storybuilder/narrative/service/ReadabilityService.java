package com.storybuilder.narrative.service;

import com.storybuilder.narrative.model.ReadabilityMetrics;
import org.springframework.stereotype.Service;

import java.util.Arrays;
import java.util.HashSet;
import java.util.Set;
import java.util.regex.Pattern;

@Service
public class ReadabilityService {

    private static final Pattern SENTENCE_SPLIT = Pattern.compile("[.!?]+\\s*");
    private static final Pattern WORD_SPLIT = Pattern.compile("\\s+");
    private static final Pattern PUNCTUATION = Pattern.compile("[^a-zA-Z0-9']");
    private static final Set<String> STOP_WORDS = new HashSet<>(Arrays.asList(
        "the", "a", "an", "and", "or", "but", "in", "on", "at", "to", "for",
        "of", "with", "by", "from", "as", "is", "was", "are", "were", "be",
        "been", "being", "have", "has", "had", "do", "does", "did", "will",
        "would", "could", "should", "may", "might", "shall", "can", "need",
        "it", "its", "this", "that", "these", "those", "i", "you", "he",
        "she", "we", "they", "me", "him", "her", "us", "them", "my", "your",
        "his", "its", "our", "their", "not", "no", "nor", "so", "if", "then",
        "than", "too", "very", "just", "about", "up", "out", "also", "well"
    ));

    public ReadabilityMetrics analyze(String text) {
        if (text == null || text.isBlank()) {
            return emptyMetrics();
        }

        String clean = text.trim();
        String[] sentences = SENTENCE_SPLIT.split(clean);
        String[] words = WORD_SPLIT.split(PUNCTUATION.matcher(clean).replaceAll(" "));

        int sentenceCount = sentences.length;
        int wordCount = words.length;
        int charCount = clean.length();
        int syllableCount = countSyllables(words);
        int[] wordLengths = Arrays.stream(words).mapToInt(String::length).toArray();

        double avgSentenceLength = sentenceCount > 0 ? (double) wordCount / sentenceCount : 0;
        double avgWordLength = wordCount > 0 ? (double) charCount / wordCount : 0;

        double fleschReadingEase = computeFleschReadingEase(avgSentenceLength, syllableCount, wordCount);
        double fleschKincaidGrade = computeFleschKincaidGrade(avgSentenceLength, syllableCount, wordCount);
        double colemanLiau = computeColemanLiau(charCount, wordCount, sentenceCount);
        double lexicalDiversity = computeLexicalDiversity(words);

        ReadabilityMetrics m = new ReadabilityMetrics();
        m.setFleschReadingEase(round(fleschReadingEase));
        m.setFleschKincaidGrade(round(fleschKincaidGrade));
        m.setColemanLiauIndex(round(colemanLiau));
        m.setAverageSentenceLength(round(avgSentenceLength));
        m.setAverageWordLength(round(avgWordLength));
        m.setLexicalDiversity(round(lexicalDiversity));
        m.setWordCount(wordCount);
        m.setSentenceCount(sentenceCount);
        m.setSyllableCount(syllableCount);
        m.setCharacterCount(charCount);
        return m;
    }

    private double computeFleschReadingEase(double avgSentenceLen, int syllableCount, int wordCount) {
        if (wordCount == 0) return 0;
        double syllPerWord = (double) syllableCount / wordCount;
        return 206.835 - (1.015 * avgSentenceLen) - (84.6 * syllPerWord);
    }

    private double computeFleschKincaidGrade(double avgSentenceLen, int syllableCount, int wordCount) {
        if (wordCount == 0) return 0;
        double syllPerWord = (double) syllableCount / wordCount;
        return (0.39 * avgSentenceLen) + (11.8 * syllPerWord) - 15.59;
    }

    private double computeColemanLiau(int charCount, int wordCount, int sentenceCount) {
        if (wordCount == 0 || sentenceCount == 0) return 0;
        double l = (double) charCount / wordCount * 100;
        double s = (double) sentenceCount / wordCount * 100;
        return (0.0588 * l) - (0.296 * s) - 15.8;
    }

    private double computeLexicalDiversity(String[] words) {
        if (words.length == 0) return 0;
        long unique = Arrays.stream(words)
            .map(String::toLowerCase)
            .filter(w -> !STOP_WORDS.contains(w))
            .distinct()
            .count();
        return (double) unique / words.length;
    }

    private int countSyllables(String[] words) {
        int count = 0;
        for (String w : words) {
            if (w.isEmpty()) continue;
            String lower = w.toLowerCase();
            int s = 0;
            boolean prevVowel = false;
            for (int i = 0; i < lower.length(); i++) {
                boolean isVowel = isVowel(lower.charAt(i));
                if (isVowel && !prevVowel) s++;
                prevVowel = isVowel;
            }
            if (lower.endsWith("e") && s > 1) s--;
            if (s == 0) s = 1;
            count += s;
        }
        return count;
    }

    private boolean isVowel(char c) {
        return "aeiouy".indexOf(c) != -1;
    }

    private ReadabilityMetrics emptyMetrics() {
        return new ReadabilityMetrics();
    }

    private double round(double v) {
        return Math.round(v * 100.0) / 100.0;
    }
}
