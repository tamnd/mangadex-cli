package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) chaptersCmd() *cobra.Command {
	var lang string

	cmd := &cobra.Command{
		Use:   "chapters <uuid>",
		Short: "Chapters for a manga",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(50)
			a.progressf("fetching chapters for %s...", args[0])
			chapters, err := a.client.Chapters(cmd.Context(), args[0], lang, n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(chapters, len(chapters))
		},
	}
	cmd.Flags().StringVar(&lang, "lang", "en", "translated language filter")
	return cmd
}
