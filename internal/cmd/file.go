package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dmitriev/mmrun/internal/config"
	"github.com/dmitriev/mmrun/internal/output"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"
)

func newFileCmd(outputMode *string) *cobra.Command {
	file := &cobra.Command{Use: "file", Short: "File operations"}

	var outDir string
	download := &cobra.Command{
		Use:   "download <post-id>",
		Short: "Download all attachments of a post",
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
	upload := &cobra.Command{
		Use:   "upload <channel> <path>",
		Short: "Upload a file, optionally with a message",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runFileUpload(app, args[0], args[1], message, cmd.OutOrStdout())
		},
	}
	upload.Flags().StringVar(&message, "message", "", "message to accompany the upload")

	file.AddCommand(download, upload)
	return file
}

func runFileDownload(app *appContext, postID, dir string) ([]string, error) {
	ctx := context.Background()
	infos, err := app.api.FileInfosForPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
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
	if len(written) == 0 {
		return nil, fmt.Errorf("post %s has no file attachments", postID)
	}
	return written, nil
}

func runFileUpload(app *appContext, channelRef, path, message string, w io.Writer) error {
	ctx := context.Background()
	ch, err := app.api.ResolveChannel(ctx, channelRef, app.defaultTeam)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	resp, err := app.api.UploadFile(ctx, data, ch.Id, filepath.Base(path))
	if err != nil {
		return err
	}
	post := &model.Post{ChannelId: ch.Id, Message: message}
	for _, fi := range resp.FileInfos {
		post.FileIds = append(post.FileIds, fi.Id)
	}
	created, err := app.api.CreatePost(ctx, post)
	if err != nil {
		return err
	}
	return output.New(app.outputMode, stdoutFile(w)).Render(w, output.Result{Text: created.Id})
}
