package com.storybuilder.narrative.controller;

import com.storybuilder.narrative.model.AnalysisRequest;
import com.storybuilder.narrative.model.SceneAnalysis;
import com.storybuilder.narrative.service.AnalysisService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequestMapping("/api/v1/analysis")
public class AnalysisController {

    private final AnalysisService analysisService;

    public AnalysisController(AnalysisService analysisService) {
        this.analysisService = analysisService;
    }

    @PostMapping("/scene")
    public ResponseEntity<SceneAnalysis> analyzeScene(@RequestBody AnalysisRequest request) {
        if (request.getSceneId() == null || request.getSceneId().isBlank()) {
            return ResponseEntity.badRequest().build();
        }
        if (request.getContent() == null || request.getContent().isBlank()) {
            return ResponseEntity.badRequest().build();
        }
        SceneAnalysis result = analysisService.analyze(request);
        return ResponseEntity.ok(result);
    }

    @GetMapping("/scene/{sceneId}")
    public ResponseEntity<SceneAnalysis> getSceneAnalysis(@PathVariable String sceneId) {
        return analysisService.getBySceneId(sceneId)
            .map(ResponseEntity::ok)
            .orElse(ResponseEntity.notFound().build());
    }

    @GetMapping("/story/{storyId}")
    public ResponseEntity<List<SceneAnalysis>> getStoryAnalysis(@PathVariable String storyId) {
        List<SceneAnalysis> results = analysisService.getByStoryId(storyId);
        return ResponseEntity.ok(results);
    }

    @DeleteMapping("/scene/{sceneId}")
    public ResponseEntity<Void> deleteSceneAnalysis(@PathVariable String sceneId) {
        analysisService.delete(sceneId);
        return ResponseEntity.noContent().build();
    }
}
