# Genji database testing

## Test results

Following obtained by running `main.go` in this directory.

Command output:

```
2020/08/27 10:54:50 SELECT * FROM tests WHERE field1 = "test1": documents found: 29, time: 306.057µs
2020/08/27 10:54:50 SELECT * FROM tests WHERE field2 = "test1": documents found: 100, time: 618.836µs
2020/08/27 10:54:50 indexed field returned 29 records, non indexed filed returned 100 records, expected 100 records for both
2020/08/27 10:54:50 All done :-)
```

Question: why does the search for an indexed field1 only return 29 records,
where a non indexed field returns 100?
