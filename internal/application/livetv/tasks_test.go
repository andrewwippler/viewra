package livetv

import "testing"

func TestEPGRefreshTask_Definition(t *testing.T) {
	task := &epgRefreshTask{}
	def := task.Definition()
	if def.ID != "internal:livetv:epg-refresh" {
		t.Errorf("ID = %q", def.ID)
	}
	if def.Schedule != "0 4 * * *" {
		t.Errorf("Schedule = %q", def.Schedule)
	}
	if def.Group != "livetv" {
		t.Errorf("Group = %q", def.Group)
	}
	if def.TimeoutSeconds != 600 {
		t.Errorf("TimeoutSeconds = %d", def.TimeoutSeconds)
	}
}

func TestSATIPPIDSyncTask_Definition(t *testing.T) {
	task := &satipPIDSyncTask{}
	def := task.Definition()
	if def.ID != "internal:livetv:satip-pid-sync" {
		t.Errorf("ID = %q", def.ID)
	}
	if def.Schedule != "0 3 1 * *" {
		t.Errorf("Schedule = %q", def.Schedule)
	}
	if def.Group != "livetv" {
		t.Errorf("Group = %q", def.Group)
	}
	if def.TimeoutSeconds != 600 {
		t.Errorf("TimeoutSeconds = %d", def.TimeoutSeconds)
	}
}

func TestTasksExported(t *testing.T) {
	if len(Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(Tasks))
	}
	foundEPG := false
	foundSatip := false
	for _, tb := range Tasks {
		def := tb.Definition()
		if def.ID == "internal:livetv:epg-refresh" {
			foundEPG = true
		}
		if def.ID == "internal:livetv:satip-pid-sync" {
			foundSatip = true
		}
	}
	if !foundEPG {
		t.Error("missing epg-refresh task")
	}
	if !foundSatip {
		t.Error("missing satip-pid-sync task")
	}
}
