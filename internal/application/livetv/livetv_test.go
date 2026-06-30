package livetv

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mantonx/viewra/internal/domain/library"
	"github.com/mantonx/viewra/internal/domain/livetv"
)

func TestToChannelResponse(t *testing.T) {
	ch := &livetv.Channel{
		ID:            1,
		LibraryID:     42,
		ChannelNumber: 5,
		Name:          "Test",
		StreamURL:     "rtsp://example.com/stream",
		LogoURL:       "http://example.com/logo.png",
		Group:         "News",
		EPGChannelID:  "test.channel",
		Enabled:       true,
	}
	resp := toChannelResponse(ch)
	if resp.ID != 1 {
		t.Errorf("ID = %d", resp.ID)
	}
	if resp.LibraryID != 42 {
		t.Errorf("LibraryID = %d", resp.LibraryID)
	}
	if resp.ChannelNumber != 5 {
		t.Errorf("ChannelNumber = %d", resp.ChannelNumber)
	}
	if resp.Name != "Test" {
		t.Errorf("Name = %q", resp.Name)
	}
	if resp.StreamURL != "rtsp://example.com/stream" {
		t.Errorf("StreamURL = %q", resp.StreamURL)
	}
	if resp.LogoURL != "http://example.com/logo.png" {
		t.Errorf("LogoURL = %q", resp.LogoURL)
	}
	if resp.Group != "News" {
		t.Errorf("Group = %q", resp.Group)
	}
	if resp.EPGChannelID != "test.channel" {
		t.Errorf("EPGChannelID = %q", resp.EPGChannelID)
	}
	if !resp.Enabled {
		t.Error("Enabled should be true")
	}
	if resp.CurrentProgram != nil {
		t.Error("CurrentProgram should be nil")
	}
}

func TestToProgramResponse(t *testing.T) {
	start := time.Date(2026, 6, 13, 4, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 13, 5, 0, 0, 0, time.UTC)
	p := &livetv.Program{
		ID:           10,
		ChannelID:    1,
		StartTime:    start,
		EndTime:      end,
		Title:        "News",
		SubTitle:     "Edition",
		Description:  "Desc",
		Category:     "News",
		EpisodeTitle: "Ep 1",
		EpisodeNum:   1,
		SeasonNum:    1,
		IsNew:        true,
		IsMovie:      false,
	}
	resp := toProgramResponse(p)
	if resp.ID != 10 {
		t.Errorf("ID = %d", resp.ID)
	}
	if resp.ChannelID != 1 {
		t.Errorf("ChannelID = %d", resp.ChannelID)
	}
	if !resp.StartTime.Equal(start) {
		t.Errorf("StartTime = %v", resp.StartTime)
	}
	if !resp.EndTime.Equal(end) {
		t.Errorf("EndTime = %v", resp.EndTime)
	}
	if resp.Title != "News" {
		t.Errorf("Title = %q", resp.Title)
	}
	if !resp.IsNew {
		t.Error("IsNew should be true")
	}
	if resp.IsMovie {
		t.Error("IsMovie should be false")
	}
}

type mockChannelRepo struct {
	livetv.ChannelRepository
	channels []*livetv.Channel
	listErr  error
	upserted []*livetv.Channel
	upsertErr error
}

func (m *mockChannelRepo) ListByLibrary(ctx context.Context, libraryID int64) ([]*livetv.Channel, error) {
	return m.channels, m.listErr
}

func (m *mockChannelRepo) Upsert(ctx context.Context, ch *livetv.Channel) error {
	if m.upsertErr != nil {
		return m.upsertErr
	}
	m.upserted = append(m.upserted, ch)
	return nil
}

type mockProgramRepo struct {
	livetv.ProgramRepository
	programs    []*livetv.Program
	currentProg *livetv.Program
	currentErr  error
	getByIDProg *livetv.Program
	getByIDErr  error
	listErr     error
	bulkCreated []*livetv.Program
	bulkCreateErr error
	deleteErr   error
}

func (m *mockProgramRepo) GetCurrentByChannel(ctx context.Context, channelID int64, now time.Time) (*livetv.Program, error) {
	return m.currentProg, m.currentErr
}

func (m *mockProgramRepo) ListByLibrary(ctx context.Context, libraryID int64, from, to time.Time) ([]*livetv.Program, error) {
	return m.programs, m.listErr
}

func (m *mockProgramRepo) GetByID(ctx context.Context, id int64) (*livetv.Program, error) {
	return m.getByIDProg, m.getByIDErr
}

func (m *mockProgramRepo) BulkCreate(ctx context.Context, programs []*livetv.Program) error {
	if m.bulkCreateErr != nil {
		return m.bulkCreateErr
	}
	m.bulkCreated = programs
	return nil
}

func (m *mockProgramRepo) DeleteByLibrary(ctx context.Context, libraryID int64) error {
	return m.deleteErr
}

type mockLibraryRepo struct {
	library.Repository
	lib        *library.Library
	getByIDErr error
	updatedID  int64
	updatedCfg *library.MonitoringConfig
	updatedOn  bool
}

func (m *mockLibraryRepo) GetByID(ctx context.Context, id int64) (*library.Library, error) {
	return m.lib, m.getByIDErr
}

func (m *mockLibraryRepo) UpdateMonitoring(ctx context.Context, id int64, enabled bool, config *library.MonitoringConfig) error {
	m.updatedID = id
	m.updatedOn = enabled
	m.updatedCfg = config
	return nil
}

func TestListChannelsUseCase(t *testing.T) {
	channelRepo := &mockChannelRepo{channels: []*livetv.Channel{
		{ID: 1, LibraryID: 42, Name: "BBC One", StreamURL: "rtsp://example.com/bbc", Enabled: true},
		{ID: 2, LibraryID: 42, Name: "BBC Two", StreamURL: "rtsp://example.com/bbc2", Enabled: true},
	}}
	programRepo := &mockProgramRepo{
		currentProg: &livetv.Program{
			ID: 100, ChannelID: 1, Title: "News at Ten",
			StartTime: time.Now().Add(-30 * time.Minute),
			EndTime:   time.Now().Add(30 * time.Minute),
		},
	}

	uc := NewListChannelsUseCase(channelRepo, programRepo)
	resp, err := uc.Execute(context.Background(), 42)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(resp.Channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(resp.Channels))
	}
	if resp.Channels[0].Name != "BBC One" {
		t.Errorf("Name = %q", resp.Channels[0].Name)
	}
	if resp.Channels[0].CurrentProgram == nil {
		t.Error("expected current program for channel 1")
	} else if resp.Channels[0].CurrentProgram.Title != "News at Ten" {
		t.Errorf("CurrentProgram.Title = %q", resp.Channels[0].CurrentProgram.Title)
	}
}

func TestListChannelsUseCase_RepoError(t *testing.T) {
	channelRepo := &mockChannelRepo{listErr: errors.New("db error")}
	uc := NewListChannelsUseCase(channelRepo, &mockProgramRepo{})
	_, err := uc.Execute(context.Background(), 42)
	if err == nil {
		t.Error("expected error from repo")
	}
}

func TestListChannelsUseCase_CurrentProgramError(t *testing.T) {
	channelRepo := &mockChannelRepo{channels: []*livetv.Channel{
		{ID: 1, LibraryID: 42, Name: "BBC One", StreamURL: "rtsp://example.com/bbc", Enabled: true},
	}}
	programRepo := &mockProgramRepo{currentErr: errors.New("no current program")}
	uc := NewListChannelsUseCase(channelRepo, programRepo)
	resp, err := uc.Execute(context.Background(), 42)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(resp.Channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(resp.Channels))
	}
	if resp.Channels[0].CurrentProgram != nil {
		t.Error("expected nil current program when program repo errors")
	}
}

func TestListEPGUseCase_Execute(t *testing.T) {
	start := time.Date(2026, 6, 13, 4, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 13, 5, 0, 0, 0, time.UTC)
	programRepo := &mockProgramRepo{programs: []*livetv.Program{
		{ID: 1, ChannelID: 1, Title: "Morning News", StartTime: start, EndTime: end},
		{ID: 2, ChannelID: 1, Title: "Daytime Show", StartTime: end, EndTime: end.Add(1 * time.Hour)},
	}}
	uc := NewListEPGUseCase(programRepo)
	from := time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC)
	resp, err := uc.Execute(context.Background(), 42, from, to)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(resp.Programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(resp.Programs))
	}
	if resp.Programs[0].Title != "Morning News" {
		t.Errorf("Title = %q", resp.Programs[0].Title)
	}
}

func TestListEPGUseCase_RepoError(t *testing.T) {
	programRepo := &mockProgramRepo{listErr: errors.New("db error")}
	uc := NewListEPGUseCase(programRepo)
	_, err := uc.Execute(context.Background(), 42, time.Time{}, time.Time{})
	if err == nil {
		t.Error("expected error from repo")
	}
}

func TestListEPGUseCase_GetByID(t *testing.T) {
	programRepo := &mockProgramRepo{
		getByIDProg: &livetv.Program{
			ID: 1, ChannelID: 1, Title: "Found Program",
			StartTime: time.Now(), EndTime: time.Now().Add(1 * time.Hour),
		},
	}
	uc := NewListEPGUseCase(programRepo)
	resp, err := uc.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if resp.Title != "Found Program" {
		t.Errorf("Title = %q", resp.Title)
	}
}

func TestListEPGUseCase_GetByID_NotFound(t *testing.T) {
	programRepo := &mockProgramRepo{getByIDErr: errors.New("not found")}
	uc := NewListEPGUseCase(programRepo)
	_, err := uc.GetByID(context.Background(), 999)
	if err == nil {
		t.Error("expected error for not found")
	}
}
