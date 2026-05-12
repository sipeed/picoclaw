package evolution

import (
	"context"
)

type CaseWriter struct {
	paths Paths
	store *Store
}

func NewCaseWriter(paths Paths) *CaseWriter {
	return &CaseWriter{
		paths: paths,
		store: NewStore(paths),
	}
}

func (w *CaseWriter) AppendCase(ctx context.Context, record LearningRecord) error {
	return w.store.AppendTaskRecord(ctx, record)
}
