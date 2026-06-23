package com.storybuilder.narrative.service;

import com.storybuilder.narrative.model.ReadabilityMetrics;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class ReadabilityServiceTest {

    private final ReadabilityService service = new ReadabilityService();

    @Test
    void emptyTextReturnsEmptyMetrics() {
        ReadabilityMetrics m = service.analyze("");
        assertEquals(0, m.getWordCount());
    }

    @Test
    void nullTextReturnsEmptyMetrics() {
        ReadabilityMetrics m = service.analyze(null);
        assertEquals(0, m.getWordCount());
    }

    @Test
    void simpleSentenceHasExpectedCounts() {
        ReadabilityMetrics m = service.analyze("The cat sat on the mat.");
        assertEquals(6, m.getWordCount());
        assertEquals(1, m.getSentenceCount());
    }

    @Test
    void fleschScoresAreInReasonableRange() {
        String text = "The cat sat on the mat. It was a sunny day. The dog ran fast.";
        ReadabilityMetrics m = service.analyze(text);
        assertTrue(m.getFleschReadingEase() > 0);
        assertTrue(m.getFleschKincaidGrade() > 0);
    }

    @Test
    void complexTextHasHigherGradeLevel() {
        String simple = "The cat is happy.";
        String complex = "Notwithstanding the aforementioned circumstances, the committee's determination regarding the fundamental feasibility of this proposition remains, at best, highly questionable.";

        ReadabilityMetrics simpleM = service.analyze(simple);
        ReadabilityMetrics complexM = service.analyze(complex);

        assertTrue(complexM.getFleschKincaidGrade() > simpleM.getFleschKincaidGrade());
    }
}
