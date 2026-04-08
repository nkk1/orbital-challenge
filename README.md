# orbital-challenge
orbital challenge


# Usage

* run the service locally
```
make run 
```

# Assumptions
* Considering only characters (also \ and '), not considering letters and other characters for credit consumption
* 0.05 for each character, includes space, numbers, special-chars

# Test cases 

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
