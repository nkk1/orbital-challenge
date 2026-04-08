# orbital-challenge
orbital challenge

## Assumptions
* Considering only characters (also \ and '), not considering letters and other characters for credit consumption
* 0.05 for each character, includes space, numbers, special-chars

## Usage

* run the service locally

```
make run
```
* run smoke tests (not mocked but actual data, so may fail if data changes)
  * NOTE: It will start the service and run the tests

```
make smoke
```

## Manual Test

* get the full response
```
curl -s http://localhost:8080/usage | jq .
```

### verify result
* message with report
  * find the raw msg
  ```
  $ curl -s 'https://owpublic.blob.core.windows.net/tech-task/messages/current-period' | jq '.messages[] | select(.id==1009)'
  {
    "text": "Produce a comprehensive Maintenance Responsibilities Report for the tenants.",
    "timestamp": "2024-04-29T16:02:31.649Z",
    "report_id": 8806,
    "id": 1009
  }
  ```

  * get its corresponding report
  ```
  $ curl -s 'https://owpublic.blob.core.windows.net/tech-task/reports/8806' | jq
  {
    "id": 8806,
    "name": "Maintenance Responsibilities Report",
    "credit_cost": 94
  }
  ```

  * verify the usage credit=94 and message_id=1009
  ```
  $ curl -s http://localhost:8080/usage | jq '.usage[] | select(.message_id == 1009)'
  {
    "credits": 94,
    "message_id": 1009,
    "report_name": "Maintenance Responsibilities Report",
    "timestamp": "2024-04-29T16:02:31.649Z"
  }
  ```

* message without report
  * find the raw msg
  ```
  $ curl -s 'https://owpublic.blob.core.windows.net/tech-task/messages/current-period' | jq '.messages[] | select(.id==1104)'
  {
    "text": "orbital latibro",
    "timestamp": "2024-05-04T14:18:20.137Z",
    "id": 1104
  }
  ```

  * Calculation
  ```
  15 characters x 0.05        = 1.75
  orbital len 4-7             = 0.2
  latibro len 4-7             = 0.2
  vowels 'a','i','o' ie 3×0.3 = 0.9
  2 unique                    =-2
  =======================================
  sum                         = 1.05

  palindrome         x2       = 2.10
  ```

  * verify usage
  ```
  $ curl -s http://localhost:8080/usage | jq '.usage[] | select(.message_id == 1104)'
  {
    "credits": 2.0999999999999988,
    "message_id": 1104,
    "timestamp": "2024-05-04T14:18:20.137Z"
  }
  ```

### audit

* total credits across the period
```
$ curl -s http://localhost:8080/usage | jq '[.usage[].credits] | add'
2264.3999999999996
```

* Only items that came from a report (have report_name):

```
$ curl -s http://localhost:8080/usage | jq '[.usage[] | select(.report_name) | .credits] | add'
1650
```

* Count of report-backed vs text-calculated messages:
```
$ curl -s http://localhost:8080/usage | jq '
  {
    total: (.usage | length),
    with_report: ([.usage[] | select(.report_name)] | length),
    text_only:   ([.usage[] | select(.report_name | not)] | length)
  }'
{
  "total": 110,
  "with_report": 25,
  "text_only": 85
}
```

* Invalid url
```
$ curl -I 'http://localhost:8080/nope'
HTTP/1.1 404 Not Found
Content-Type: text/plain; charset=utf-8
X-Content-Type-Options: nosniff
Date: Wed, 08 Apr 2026 11:07:57 GMT
Content-Length: 19
```


## TODO

* Run in a Kind environment
* Add tracing
