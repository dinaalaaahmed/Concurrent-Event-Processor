```docker run --rm -it `
  --network nahdi-assesment_default `
  -p 3000:3000 `
  -v ${PWD}:/app `
  -w /app `
  -e CGO_ENABLED=1 `
  -e DB_DSN="postgres://postgres:mysecretpassword@events-processing-postgres:5432/events-processing?sslmode=disable" `
  golang:1.25 go run -race ./cmd/...```

```
for i in {1..100}; do
  curl -X POST http://localhost:3000/events \
    -H "Content-Type: application/json" \
    -d "{
        \"user_id\": \"user-$i\",
        \"event_type\": \"click\",
        \"value\": 1,
        \"timestamp\": \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"
    }" &
done
wait
```
