package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/config"
	"github.com/isdmx/mmrun/internal/output"
)

func newFileCmd(outputMode *string) *cobra.Command {
	file := &cobra.Command{Use: "file", Short: "File operations"}

	var outDir string
	download := &cobra.Command{
		Use:   "download <post-or-file-id>",
		Short: "Download attachments of a post, or a single file by its ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			dir := outDir
			if dir == "" {
				dir = config.Paths().DownloadDir
			}
			paths, err := runFileDownload(app, args[0], dir)
			if err != nil {
				return err
			}
			res := output.Result{Title: "Downloaded", Columns: []string{"path"}}
			for _, p := range paths {
				res.Rows = append(res.Rows, output.Row{"path": p})
			}
			return output.New(app.outputMode, stdoutFile(cmd.OutOrStdout())).Render(cmd.OutOrStdout(), res)
		},
	}
	download.Flags().StringVar(&outDir, "out", "", "output directory (defaults to XDG download dir)")

	var message string
	var uploadTeam string
	upload := &cobra.Command{
		Use:   "upload <channel> <path>...",
		Short: "Upload one or more files, optionally with a message",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runFileUpload(app, args[0], args[1:], message, uploadTeam, cmd.OutOrStdout())
		},
	}
	upload.Flags().StringVar(&message, "message", "", "message to accompany the upload")
	upload.Flags().StringVar(&uploadTeam, "team", "", "team for resolving a bare channel name (defaults to your team if you have only one)")

	file.AddCommand(download, upload)
	return file
}

func runFileDownload(app *appContext, id, dir string) ([]string, error) {
	ctx := context.Background()
	infos, err := fileInfosFor(ctx, app, id)
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, fmt.Errorf("%q has no downloadable files (not a post with attachments or a file ID)", id)
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	var written []string
	for _, fi := range infos {
		data, err := app.api.GetFile(ctx, fi.Id)
		if err != nil {
			return nil, err
		}
		dest := filepath.Join(dir, fi.Name)
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return nil, err
		}
		written = append(written, dest)
	}
	return written, nil
}

// fileInfosFor resolves an argument that may be either a post ID (returning its
// attachments) or a single file ID (returning that one file's info).
func fileInfosFor(ctx context.Context, app *appContext, id string) ([]*model.FileInfo, error) {
	infos, err := app.api.FileInfosForPost(ctx, id)
	if err == nil && len(infos) > 0 {
		return infos, nil
	}
	// Fall back to treating the argument as a file ID.
	if fi, ferr := app.api.FileInfo(ctx, id); ferr == nil && fi != nil {
		return []*model.FileInfo{fi}, nil
	}
	// No file match: surface the original post-lookup error if there was one.
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func runFileUpload(app *appContext, channelRef string, paths []string, message, team string, w io.Writer) error {
	ctx := context.Background()
	ch, err := app.resolveChannel(ctx, channelRef, team)
	if err != nil {
		return err
	}
	fileIDs, err := uploadFiles(ctx, app, ch.Id, paths)
	if err != nil {
		return err
	}
	post := &model.Post{ChannelId: ch.Id, Message: message, FileIds: fileIDs}
	created, err := app.api.CreatePost(ctx, post)
	if err != nil {
		return err
	}
	return output.New(app.outputMode, stdoutFile(w)).Render(w, output.Result{Text: created.Id})
}

// uploadFiles uploads each path to the channel and returns the resulting file
// IDs, ready to attach to a single post.
func uploadFiles(ctx context.Context, app *appContext, channelID string, paths []string) ([]string, error) {
	var ids []string
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		resp, err := app.api.UploadFile(ctx, data, channelID, filepath.Base(p))
		if err != nil {
			return nil, err
		}
		for _, fi := range resp.FileInfos {
			ids = append(ids, fi.Id)
		}
	}
	return ids, nil
}
