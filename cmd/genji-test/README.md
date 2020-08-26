# Genji database testing

## Test results

Following obtained by running `main.go` in this directory.

Command output:

```
2020/08/26 16:12:06 SELECT * FROM users WHERE email = "joe@admin.com": documents found: 1, time: 113.778µs
2020/08/26 16:12:06 SELECT * FROM users WHERE firstname = "Joe": documents found: 1, time: 76.951541ms
2020/08/26 16:12:06 SELECT * FROM users WHERE email = "fred@admin.com": documents found: 29, time: 519.651µs
2020/08/26 16:12:06 SELECT * FROM users WHERE firstname = "Fred": documents found: 100000, time: 591.640536ms
2020/08/26 16:12:07 SELECT * FROM users: documents found: 100001, time: 505.113878ms
2020/08/26 16:12:07 All done :-)

```

Notes:

- User insert time (100 users)
  - older workstation with spinning HD: 57.9ms
  - newer workstation with SSD: 41.49ms
- DB size
  - 1000 records: 524288 (524 bytes/record)
  - 100,000 records: 41951232 (419 bytes/record)
- does inserting a lot of records with the same indexed value cause insert times
  to grow?
  - insert time for 100,000 records averages 33.73ms, so index does not appear
    to affect write time with large record sets.
- does index automatically improve query?
  - in our test, we index email, but not firstname
  - search on value in a table of 100,000 entries
    - search by email (indexed) takes: 116us
    - search by firstname (not indexed) takes: 90ms
    - indexed search is roughly an order of magnitude faster, but even a raw
      search is still blazing fast for 100,000 records -- seems this database
      would scale quite well.
