package livetv

import (
	"context"
	"fmt"
	"time"

	"github.com/mantonx/viewra/internal/domain/livetv"
)

type ListEPGUseCase struct {
	programRepo livetv.ProgramRepository
}

func NewListEPGUseCase(programRepo livetv.ProgramRepository) *ListEPGUseCase {
	return &ListEPGUseCase{
		programRepo: programRepo,
	}
}

func (uc *ListEPGUseCase) Execute(ctx context.Context, libraryID int64, from, to time.Time) (ListEPGResponse, error) {
	programs, err := uc.programRepo.ListByLibrary(ctx, libraryID, from, to)
	if err != nil {
		return ListEPGResponse{}, fmt.Errorf("failed to list programs: %w", err)
	}

	resp := make([]ProgramResponse, len(programs))
	for i, p := range programs {
		resp[i] = toProgramResponse(p)
	}

	return ListEPGResponse{Programs: resp}, nil
}

func (uc *ListEPGUseCase) GetByID(ctx context.Context, programID int64) (*ProgramResponse, error) {
	program, err := uc.programRepo.GetByID(ctx, programID)
	if err != nil {
		return nil, fmt.Errorf("failed to get program: %w", err)
	}

	resp := toProgramResponse(program)
	return &resp, nil
}
