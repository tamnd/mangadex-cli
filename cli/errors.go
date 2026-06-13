package cli

import (
	"errors"

	"github.com/tamnd/mangadex-cli/mangadex"
)

func isNotFound(err error) bool {
	return errors.Is(err, mangadex.ErrNotFound)
}
