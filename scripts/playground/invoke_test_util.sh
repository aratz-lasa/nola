curl -w -X POST "http://localhost:9090/api/v1/invoke-actor" -H 'Content-Type: application/json' -d '{"namespace":"playground", "operation":"inc", "actor_id":"test_utils_actor_1"}'
