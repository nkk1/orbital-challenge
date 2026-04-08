# orbital-challenge
orbital challenge

# Assumptions
* Considering only characters (also \ and '), not considering letters and other characters for credit consumption
* 0.05 for each character, includes space, numbers, special-chars

# Usage

* run the service locally

```
make run 
```
* run smoke tests (not mocked but actual data, so may fail if data changes)

```
go test -tags smoke -v ./server/api/...
```

# Test

* test 
```
curl -s http://localhost:8080/usage | jq .
```

* total credits across the period
```
curl -s http://localhost:8080/usage | jq '[.usage[].credits] | add'
```

* Only items that came from a report (have report_name):

```
curl -s http://localhost:8080/usage | jq '.usage[] | select(.report_name)'
```

*  Count of report-backed vs text-calculated messages:

```
curl -s http://localhost:8080/usage | jq '
  {
    total: (.usage | length),
    with_report: ([.usage[] | select(.report_name)] | length),
    text_only:   ([.usage[] | select(.report_name | not)] | length)
  }'
```

# TODO

* Run in a Kind environment 
* Add tracing
