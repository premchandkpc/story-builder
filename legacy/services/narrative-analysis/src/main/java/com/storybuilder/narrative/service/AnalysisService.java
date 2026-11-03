package com.storybuilder.narrative.service;

import com.storybuilder.narrative.model.*;
import com.storybuilder.narrative.repository.SceneAnalysisRepository;
import org.springframework.stereotype.Service;

import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.Optional;
import java.util.regex.Pattern;

@Service
public class AnalysisService {

    private static final Pattern WORD_SPLIT = Pattern.compile("\\s+");
    private static final Pattern PUNCTUATION = Pattern.compile("[^a-zA-Z0-9']");

    private final ReadabilityService readabilityService;
    private final SentimentService sentimentService;
    private final PacingService pacingService;
    private final SceneAnalysisRepository repository;

    public AnalysisService(ReadabilityService readabilityService,
                           SentimentService sentimentService,
                           PacingService pacingService,
                           SceneAnalysisRepository repository) {
        this.readabilityService = readabilityService;
        this.sentimentService = sentimentService;
        this.pacingService = pacingService;
        this.repository = repository;
    }

    public SceneAnalysis analyze(AnalysisRequest request) {
        String text = request.getContent();

        SceneAnalysis analysis = new SceneAnalysis();
        analysis.setSceneId(request.getSceneId());
        analysis.setStoryId(request.getStoryId());
        analysis.setTitle(request.getTitle());
        analysis.setReadability(readabilityService.analyze(text));
        analysis.setSentiment(sentimentService.analyze(text));
        analysis.setPacing(pacingService.analyze(text));
        analysis.setCharacterMentions(findCharacterMentions(text, request.getCharacterNames()));
        analysis.setCreatedAt(Instant.now());
        analysis.setUpdatedAt(Instant.now());

        return repository.save(analysis);
    }

    public Optional<SceneAnalysis> getBySceneId(String sceneId) {
        return repository.findBySceneId(sceneId);
    }

    public List<SceneAnalysis> getByStoryId(String storyId) {
        return repository.findByStoryIdOrderByCreatedAtAsc(storyId);
    }

    public void delete(String sceneId) {
        repository.deleteBySceneId(sceneId);
    }

    private List<CharacterMention> findCharacterMentions(String text, List<String> characterNames) {
        List<CharacterMention> mentions = new ArrayList<>();
        if (characterNames == null || characterNames.isEmpty()) return mentions;

        String[] words = WORD_SPLIT.split(PUNCTUATION.matcher(text).replaceAll(" "));
        int totalWords = words.length;

        for (String name : characterNames) {
            if (name == null || name.isBlank()) continue;
            int count = 0;
            String lowerName = name.toLowerCase();
            for (String w : words) {
                if (w.toLowerCase().equals(lowerName)) count++;
            }
            if (count > 0) {
                mentions.add(new CharacterMention(
                    name, count, totalWords > 0 ? round((double) count / totalWords) : 0
                ));
            }
        }

        mentions.sort((a, b) -> Integer.compare(b.getMentionCount(), a.getMentionCount()));
        return mentions;
    }

    private double round(double v) {
        return Math.round(v * 100.0) / 100.0;
    }
}
