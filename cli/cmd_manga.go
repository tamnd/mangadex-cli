package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) mangaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "manga <uuid>",
		Short: "Manga details by UUID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.progressf("fetching manga %s...", args[0])
			m, err := a.client.GetManga(cmd.Context(), args[0])
			if err != nil {
				return mapFetchErr(err)
			}
			return a.render(m)
		},
	}
}
