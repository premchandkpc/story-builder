package com.storybuilder.narrative.repository;

import com.storybuilder.narrative.model.SceneAnalysis;
import org.springframework.data.mongodb.repository.MongoRepository;

import java.util.List;
import java.util.Optional;

public interface SceneAnalysisRepository extends MongoRepository<SceneAnalysis, String> {
    Optional<SceneAnalysis> findBySceneId(String sceneId);
    List<SceneAnalysis> findByStoryIdOrderByCreatedAtAsc(String storyId);
    void deleteBySceneId(String sceneId);
}
