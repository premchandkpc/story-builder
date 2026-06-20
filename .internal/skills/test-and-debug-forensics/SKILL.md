# Test & Debug Forensics

Integration tests, failure checklists, regression strategy for story-builder.

## Trigger
- "write tests for [feature]"
- "debug [failure]"
- "reproduce [bug]"
- "test plan for [area]"

## Test layers

### Graph tests (priority: highest)
- Cycle detection (valid DAG rejects, invalid accepts)
- Topological sort (ordering correct for seq/fork/join/choice)
- FindDeadEnds, FindUnreachableScenes, FindBranches

### Memory tests
- State append (CharacterStateRepository.Append)
- Embedding search (mock embedding, verify pipeline runs)

### Generation pipeline tests
- Mock each LLM service, verify pipeline orchestrates correctly
- Verify retry logic
- Verify partial success propagation

### Validation tests
- Pre-generation validation (missing fields, invalid edges)
- Post-generation validation (location jumps, knowledge contradictions)

### API integration tests
- CRUD for each entity
- Edge cases: delete story cascades, edge deletion cascades

### Agent tests (new)
- Each agent spec runs and produces expected output shape
- Orchestrator executes plan correctly
- Turn ordering respects flow type

### Debug checklist

```text
Failure: generation stuck in "running"
1. Check sync.Map genInFlight — sceneID present?
2. Check worker goroutine panic
3. Check LLM client timeout (default 5min)
4. Check context cancellation
5. Check Mongo connectivity
6. Check circuit breaker state in llm/client.go
```

## Output
- Test file with clear Arrange/Act/Assert
- Debug checklist for each failure mode
- Regression test list for each bug fix
