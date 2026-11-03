package com.storybuilder.narrative.service;

import com.storybuilder.narrative.model.PacingMetrics;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class PacingServiceTest {

    private final PacingService service = new PacingService();

    @Test
    void emptyTextReturnsEmpty() {
        PacingMetrics m = service.analyze("");
        assertEquals(0, m.getParagraphCount());
    }

    @Test
    void dialogueIsDetected() {
        String text = "\"Hello there,\" she said.\n\nHe nodded silently.";
        PacingMetrics m = service.analyze(text);
        assertTrue(m.getDialogueRatio() > 0);
    }

    @Test
    void paragraphCountIsCorrect() {
        String text = "First paragraph.\n\nSecond paragraph.\n\nThird paragraph.";
        PacingMetrics m = service.analyze(text);
        assertEquals(3, m.getParagraphCount());
    }
}
