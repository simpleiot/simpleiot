package isdb

import (
	"fmt"
	"time"

	"github.com/simpleiot/simpleiot/data"
	"github.com/simpleiot/simpleiot/farmation/isdata"
)

// DbDumpSamples dumps all samples from DB
func DbDumpSamples(dataDir string) error {
	config := isdata.Config{}
	state := isdata.State{}

	_, _, dbData, err := DbInit(dataDir, &config, &state)

	if err != nil {
		return err
	}

	count := 0

	err = dbData.store.ForEach(nil, func(samp *data.Sample) error {
		fmt.Printf("%+v\n", samp)
		count++
		return nil
	})

	fmt.Printf("\n\nSample count: %v\n", count)

	if err != nil {
		return err
	}

	return nil
}

// PopDbTestData sets the system timezone from the config setting
func PopDbTestData(dataDir string) error {
	config := isdata.Config{}
	state := isdata.State{}

	_, _, dbData, err := DbInit(dataDir, &config, &state)

	if err != nil {
		return err
	}

	// data in samples bucket:
	//&{Type:amount ID: Value:5.499867899603685 Min:0 Max:0 Time:2020-04-23 10:32:46.674557782 -0400 EDT Duration:0s Tags:map[] Attributes:map[]}

	//&{Type:flowWindowAvg ID: Value:32.999203323694665 Min:32.84477521008442 Max:33.00651340376113 Time:2020-04-23 10:32:50.675466547 -0400 EDT Duration:0s Tags:map[] Attributes:map[]}

	//37:&{Type:faultFlowOffTarget ID: Value:2.000010202716653 Min:0 Max:0 Time:2020-04-23 11:05:49.039262969 -0400 EDT Duration:0s Tags:map[] Attributes:map[inputInjector:2 inputIrrigator:2 inputWaterOn:2 shutdownThresHigh:7.291944775264274 shutdownThresLow:5.389698312151855]}

	ts := time.Now().AddDate(0, -2, 0)

	count := 0
	faultCount := 0
	printCount := 0
	timeLastPrint := time.Now()

	for {
		if ts.After(time.Now()) {
			break
		}

		s := data.Sample{
			Time:  ts,
			Type:  isdata.SampleTypeAmount,
			Value: 10.32,
		}

		dbData.WriteSample(s)
		count++

		s = data.Sample{
			Time:  ts.Add(time.Second),
			Type:  isdata.SampleTypeFlowWindowAvg,
			Value: 32.9999,
			Min:   32.8423,
			Max:   34.232,
		}

		dbData.WriteSample(s)
		count++

		if faultCount > 100 {
			s = data.Sample{
				Time:  ts.Add(time.Second * 2),
				Type:  isdata.SampleTypeFaultFlowOff,
				Value: 2.00023,
				Attributes: map[string]float64{
					"inputInjector":     1,
					"inputWaterOn":      0,
					"inputIrrigator":    1,
					"shutdownThresHigh": 0,
					"shutdownThresLow":  1,
				},
			}
			dbData.WriteSample(s)
			count++
			faultCount = 0
		}

		ts = ts.Add(time.Minute * 10)

		faultCount++
		printCount++
		if printCount > 10 {
			timeLeft := time.Now().Sub(ts)
			fmt.Printf("count: %v, timeleft: %v, time per sample: %v\n", count, timeLeft, time.Since(timeLastPrint)/time.Duration(printCount))
			printCount = 0
			timeLastPrint = time.Now()
		}
	}

	fmt.Printf("Inserted %v entries into DB\n", count)

	return nil
}
