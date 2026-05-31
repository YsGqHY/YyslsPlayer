package midi

import "testing"

func TestTempoMapDefaultTempo(t *testing.T) {
	m, err := NewTempoMap(480, nil)
	if err != nil {
		t.Fatalf("NewTempoMap failed: %v", err)
	}

	assertTick(t, m, 0, 0, 0)
	assertTick(t, m, 240, 250000, 250)
	assertTick(t, m, 480, 500000, 500)
	assertTick(t, m, 960, 1000000, 1000)
	assertTick(t, m, -120, 0, 0)

	segments := m.Segments()
	if len(segments) != 1 {
		t.Fatalf("segments = %d, want 1", len(segments))
	}
	if segments[0].Tick != 0 || segments[0].MicrosecondsPerQuarter != defaultTempoMicrosecondsPerQuarter {
		t.Fatalf("default segment = %+v", segments[0])
	}
}

func TestTempoMapWithTempoChanges(t *testing.T) {
	m, err := NewTempoMap(480, []TempoChange{
		{Tick: 480, MicrosecondsPerQuarter: 1000000},
		{Tick: 960, MicrosecondsPerQuarter: 250000},
	})
	if err != nil {
		t.Fatalf("NewTempoMap failed: %v", err)
	}

	assertTick(t, m, 480, 500000, 500)
	assertTick(t, m, 720, 1000000, 1000)
	assertTick(t, m, 960, 1500000, 1500)
	assertTick(t, m, 1440, 1750000, 1750)

	segments := m.Segments()
	if len(segments) != 3 {
		t.Fatalf("segments = %d, want 3", len(segments))
	}
	if segments[0].Tick != 0 || segments[0].TimeMicroseconds != 0 || segments[0].MicrosecondsPerQuarter != 500000 {
		t.Fatalf("segment 0 = %+v", segments[0])
	}
	if segments[1].Tick != 480 || segments[1].TimeMicroseconds != 500000 || segments[1].MicrosecondsPerQuarter != 1000000 {
		t.Fatalf("segment 1 = %+v", segments[1])
	}
	if segments[2].Tick != 960 || segments[2].TimeMicroseconds != 1500000 || segments[2].MicrosecondsPerQuarter != 250000 {
		t.Fatalf("segment 2 = %+v", segments[2])
	}
}

func TestTempoMapNormalizesUnorderedDuplicateAndInvalidChanges(t *testing.T) {
	m, err := NewTempoMap(480, []TempoChange{
		{Tick: 960, MicrosecondsPerQuarter: 250000},
		{Tick: -1, MicrosecondsPerQuarter: 100000},
		{Tick: 0, MicrosecondsPerQuarter: 600000},
		{Tick: 480, MicrosecondsPerQuarter: 0},
		{Tick: 480, MicrosecondsPerQuarter: 1000000},
		{Tick: 480, MicrosecondsPerQuarter: 750000},
	})
	if err != nil {
		t.Fatalf("NewTempoMap failed: %v", err)
	}

	assertTick(t, m, 480, 600000, 600)
	assertTick(t, m, 960, 1350000, 1350)
	assertTick(t, m, 1440, 1600000, 1600)

	segments := m.Segments()
	if len(segments) != 3 {
		t.Fatalf("segments = %d, want 3", len(segments))
	}
	if segments[0].Tick != 0 || segments[0].MicrosecondsPerQuarter != 600000 {
		t.Fatalf("segment 0 = %+v", segments[0])
	}
	if segments[1].Tick != 480 || segments[1].MicrosecondsPerQuarter != 750000 {
		t.Fatalf("segment 1 = %+v", segments[1])
	}
	if segments[2].Tick != 960 || segments[2].MicrosecondsPerQuarter != 250000 {
		t.Fatalf("segment 2 = %+v", segments[2])
	}
}

func TestTempoMapRejectsInvalidPPQ(t *testing.T) {
	if _, err := NewTempoMap(0, nil); err == nil {
		t.Fatalf("expected invalid PPQ error")
	}
	if _, err := NewTempoMap(-480, nil); err == nil {
		t.Fatalf("expected invalid PPQ error")
	}
}

func TestTempoMapSegmentsReturnsCopy(t *testing.T) {
	m, err := NewTempoMap(480, []TempoChange{{Tick: 480, MicrosecondsPerQuarter: 1000000}})
	if err != nil {
		t.Fatalf("NewTempoMap failed: %v", err)
	}

	segments := m.Segments()
	segments[0].MicrosecondsPerQuarter = 1
	assertTick(t, m, 480, 500000, 500)
}

func assertTick(t *testing.T, m TempoMap, tick, wantUS, wantMS int64) {
	t.Helper()

	if got := m.TickToMicroseconds(tick); got != wantUS {
		t.Fatalf("TickToMicroseconds(%d) = %d, want %d", tick, got, wantUS)
	}
	if got := m.TickToMilliseconds(tick); got != wantMS {
		t.Fatalf("TickToMilliseconds(%d) = %d, want %d", tick, got, wantMS)
	}
}
