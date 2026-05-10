package main

import "testing"

func TestLoadTeamContentSortsSeniorsByGradeDescending(t *testing.T) {
	s := newDLTestServer(t)
	raw := M{
		"seniors": []any{
			M{"id": "s23", "name": "Senior 2023", "grade": "2023级", "created_at": "2026-05-10T00:00:00Z"},
			M{"id": "s24", "name": "Senior 2024", "grade": "2024级", "created_at": "2024-01-01T00:00:00Z"},
			M{"id": "s25", "name": "Senior 2025", "grade": "2025级", "created_at": "2023-01-01T00:00:00Z"},
		},
	}
	if err := s.saveJSONToDB("team_content", raw); err != nil {
		t.Fatal(err)
	}

	items := asList(s.loadTeamContent()["seniors"])
	if len(items) != 3 {
		t.Fatalf("got %d seniors, want 3", len(items))
	}
	got := []string{}
	for _, item := range items {
		got = append(got, asString(asMap(item)["id"]))
	}
	want := []string{"s25", "s24", "s23"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("senior order=%v, want %v", got, want)
		}
	}
}
