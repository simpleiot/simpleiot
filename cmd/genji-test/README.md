# Genji database testing

## Test results

Following obtained by running `main.go` in this directory.

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
