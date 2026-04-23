package simplykb

import "errors"

var ErrDocumentChangedConcurrently = errors.New("document changed concurrently")
