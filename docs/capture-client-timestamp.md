# Ticket: Capture Client Timestamp

## Product AC

Capture the client event timestamp (occurred_at)

1. The API must accept an optional occurred_at field on incoming events, giving the time the event happened on the client.
2. The pipeline must save occurred_at to the database with the event.
3. If occurred_at is missing, the event must still be accepted, and the time the server received it is saved instead.
4. If occurred_at is not a valid timestamp or is in the future, the API must reject the event with a clear error message and must not save it.

---

## Technical AC

### Unit Test

Function to test: [pipeline_handler.go->ProcessAndStoreEvents](internal/adapter/handler/pipeline_handler.go)
Test file: [pipeline_handler_test.go](internal/adapter/handler/pipeline_handler_test.go)

1. Techical AC 1 (Happy path): Given a valid `occured_at`, ensure the response is 202 and the `event` passed to the service has `OccuredAt` set to UTC time. (Product AC 1)
2. Technical AC 2 (Missing field) Given a request without `occured_at`, ensure the response is 202 and the `event` passed to the service injected with server time. Use fixed clock in for deterministic test. (Product AC 3)
3. Technical AC 3 (Invalid format): Given invalid value `"01-01-2026"`, `"invalid-date"`, ensure the response is 400 and service is not called. (Product AC 4)
4. Technical AC 4 (Future time): Given the time is out of range (beyond allowed clock drift), ensure the response is 400 and service is not called. (Product AC 4)


### Integration Test

Function to test [postgres_event_repository.go->CreateEvents](internal/adapter/repository/postgres_event_repository.go)
Test file: [postgres_event_repository_test.go](internal/adapter/repository/postgres_event_repository_test.go)

1. Technical AC 5 (Persist): Given a batch of events with `OccuredAt` set, ensure each row's `occured_at` column in Postgres matches.
