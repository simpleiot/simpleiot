from(bucket:"bec")
        |> range(start: -1h)
        |> filter(fn: (r) => r._measurement == "points" and
		r.nodeID == "64015fec-2786-47c1-9b4b-a6c92ebc0052" and
		r.type == "value" and
		r._field == "value")
