package data

import (
	"testing"
	"time"
)

func TestSampleAverager(t *testing.T) {
	// Round 1
	min := 50.01
	max := 5000.90
	avg := 2000.00

	sampleAverager := NewSampleAverager("testSample")
	avgSample := sampleAverager.GetAverage()
	if avgSample.Value != 0 {
		t.Error("sample avg with 0 samples is not correct: ", avgSample.Value)
	}

	// Round 1.5
	feedSamples(sampleAverager, avg, min, max)

	avgSample = sampleAverager.GetAverage()
	if avgSample.Value != avg {
		t.Error("sample avg is not correct")
	}
	if avgSample.Min != min {
		t.Error("sample min is not correct")
	}
	if avgSample.Max != max {
		t.Error("sample max is not correct")
	}

	sampleAverager.ResetAverage()

	// Round 2
	min = 1
	max = 3
	avg = 2

	sampleAverager = NewSampleAverager("testSample")
	feedSamples(sampleAverager, avg, min, max)

	avgSample = sampleAverager.GetAverage()
	if avgSample.Value != avg {
		t.Error("sample avg is not correct")
	}
	if avgSample.Min != min {
		t.Error("sample min is not correct")
	}
	if avgSample.Max != max {
		t.Error("sample max is not correct")
	}

	sampleAverager.ResetAverage()

	// Round 3
	min = 5.01
	max = 500.90
	avg = 200.00

	sampleAverager = NewSampleAverager("testSample")
	feedSamples(sampleAverager, avg, min, max)

	avgSample = sampleAverager.GetAverage()
	if avgSample.Value != avg {
		t.Error("sample avg is not correct")
	}
	if avgSample.Min != min {
		t.Error("sample min is not correct")
	}
	if avgSample.Max != max {
		t.Error("sample max is not correct")
	}
}

func feedSamples(sampleAverager *SampleAverager, avg, min, max float64) {
	sample := Sample{
		Time:  time.Now(),
		Value: avg - 100,
		Min:   min + 100,
		Max:   max - 100,
	}
	sampleAverager.AddSample(sample)
	sampleAverager.AddSample(sample)

	sample = Sample{
		Time:  time.Now(),
		Value: avg,
		Min:   min,
		Max:   max,
	}
	sampleAverager.AddSample(sample)
	sampleAverager.AddSample(sample)

	sample = Sample{
		Time:  time.Now(),
		Value: avg + 100,
		Min:   min + 100,
		Max:   max - 100,
	}
	sampleAverager.AddSample(sample)
	sampleAverager.AddSample(sample)
}
