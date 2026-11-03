package com.storybuilder.narrative.service;

import com.storybuilder.narrative.model.SentimentResult;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class SentimentServiceTest {

    private final SentimentService service = new SentimentService();

    @Test
    void emptyTextReturnsEmptyResult() {
        SentimentResult r = service.analyze("");
        assertEquals(0, r.getOverallScore());
    }

    @Test
    void positiveTextScoresPositively() {
        SentimentResult r = service.analyze("This is a wonderful and beautiful day. I feel great and happy.");
        assertTrue(r.getOverallScore() > 0);
    }

    @Test
    void negativeTextScoresNegatively() {
        SentimentResult r = service.analyze("This is a terrible and horrible day. I feel awful and sad.");
        assertTrue(r.getOverallScore() < 0);
    }
}
