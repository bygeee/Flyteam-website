package main

import "testing"

func TestLoadTeamContentSortsSeniorsByGradeDescending(t *testing.T) {
	s := newDLTestServer(t)
	raw := M{
		"seniors": []any{
			M{"id": "s23", "name": "Senior 2023", "grade": "2023\u7ea7", "created_at": "2026-05-10T00:00:00Z"},
			M{"id": "s24", "name": "Senior 2024", "grade": "2024\u7ea7", "created_at": "2024-01-01T00:00:00Z"},
			M{"id": "s25", "name": "Senior 2025", "grade": "2025\u7ea7", "created_at": "2023-01-01T00:00:00Z"},
		},
	}
	if err := s.saveJSONToDB("team_content", raw); err != nil {
		t.Fatal(err)
	}

	items := asList(s.loadTeamContent()["seniors"])
	assertSeniorIDs(t, items, []string{"s25", "s24", "s23"})
}

func TestLoadTeamContentSortsLeaderThenResponsibleBeforeGrades(t *testing.T) {
	s := newDLTestServer(t)
	raw := M{
		"seniors": []any{
			M{"id": "normal25", "name": "Normal 2025", "grade": "2025\u7ea7"},
			M{"id": "resp23", "name": "Responsible 2023", "grade": "2023\u7ea7", "responsible": true},
			M{"id": "leader", "name": "Leader", "grade": "\u5e2e\u4e3b"},
			M{"id": "resp24", "name": "Responsible 2024", "grade": "2024\u7ea7", "responsible": true},
			M{"id": "normal24", "name": "Normal 2024", "grade": "2024\u7ea7"},
		},
	}
	if err := s.saveJSONToDB("team_content", raw); err != nil {
		t.Fatal(err)
	}

	items := asList(s.loadTeamContent()["seniors"])
	assertSeniorIDs(t, items, []string{"leader", "resp24", "resp23", "normal25", "normal24"})
}

func assertSeniorIDs(t *testing.T, items []any, want []string) {
	t.Helper()
	if len(items) != len(want) {
		t.Fatalf("got %d seniors, want %d", len(items), len(want))
	}
	got := []string{}
	for _, item := range items {
		got = append(got, asString(asMap(item)["id"]))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("senior order=%v, want %v", got, want)
		}
	}
}
