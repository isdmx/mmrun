package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/client"
	"github.com/isdmx/mmrun/internal/output"
	"github.com/isdmx/mmrun/internal/progress"
)

func newFileCmd(outputMode *string) *cobra.Command {
	file := &cobra.Command{Use: "file", Short: "File operations"}

	var outDir string
	var dlProgress bool
	var dlNoProgress bool
	download := &cobra.Command{
		Use:   "download <post-or-file-id>",
		Short: "Download attachments of a post, or a single file by its ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			if dlProgress && dlNoProgress {
				return fmt.Errorf("--progress and --no-progress are mutually exclusive")
			}
			dir := outDir
			if dir == "" {
				dir = app.downloadDir
			}
			paths, err := runFileDownload(app, args[0], dir, dlProgress, dlNoProgress)
			if err != nil {
				return err
			}
			res := output.Result{Title: "Downloaded", Columns: []string{"path"}}
			for _, p := range paths {
				res.Rows = append(res.Rows, output.Row{"path": p})
			}
			return app.render(cmd.OutOrStdout(), res)
		},
	}
	download.Flags().StringVar(&outDir, "out", "", "output directory (defaults to XDG download dir)")
	download.Flags().BoolVar(&dlProgress, "progress", false, "force progress bar on")
	download.Flags().BoolVar(&dlNoProgress, "no-progress", false, "force progress bar off")

	var message string
	var uploadTeam string
	var uploadDryRun bool
	var upProgress bool
	var upNoProgress bool
	upload := &cobra.Command{
		Use:   "upload <channel> <path>...",
		Short: "Upload one or more files, optionally with a message",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			if upProgress && upNoProgress {
				return fmt.Errorf("--progress and --no-progress are mutually exclusive")
			}
			return runFileUpload(app, args[0], args[1:], message, uploadTeam, uploadDryRun, upProgress, upNoProgress, cmd.OutOrStdout())
		},
	}
	upload.Flags().StringVar(&message, "message", "", "message to accompany the upload")
	upload.Flags().StringVar(&uploadTeam, "team", "", "team for resolving a bare channel name (defaults to your team if you have only one)")
	upload.Flags().BoolVar(&uploadDryRun, "dry-run", false, "resolve the target and preview without uploading")
	upload.Flags().BoolVar(&upProgress, "progress", false, "force progress bar on")
	upload.Flags().BoolVar(&upNoProgress, "no-progress", false, "force progress bar off")
	upload.ValidArgsFunction = completeChannelArg

	file.AddCommand(download, upload)
	return file
}

func runFileDownload(app *appContext, id, dir string, progressForce, noProgressForce bool) ([]string, error) {
	ctx := context.Background()
	infos, err := fileInfosFor(ctx, app, id)
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, fmt.Errorf("%q has no downloadable files (not a post with attachments or a file ID)", id)
	}
	totalSize := int64(0)
	for _, fi := range infos {
		totalSize += fi.Size
	}
	isTTY := output.IsTTY(os.Stderr)
	if shouldShowBar(isTTY, totalSize, progressForce, noProgressForce, app.outputMode, app.quiet, app.mustLogin) {
		bar := progress.NewBar(os.Stderr, totalSize)
		bar.SetLabel("downloading")
		defer bar.Done()
		app.api = client.NewWithToken(app.api.ServerURL(), app.api.Token(), bar)
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

func runFileUpload(app *appContext, channelRef string, paths []string, message, team string, dryRun bool, progressForce, noProgressForce bool, w io.Writer) error {
	ctx := context.Background()
	ch, err := app.resolveChannel(ctx, channelRef, team)
	if err != nil {
		return err
	}
	if dryRun {
		res := output.Result{
			Title:   "Dry run (not uploaded)",
			Columns: []string{"field", "value"},
			Rows: []output.Row{
				{"field": "channel", "value": ch.Id},
				{"field": "message", "value": message},
				{"field": "files", "value": strings.Join(paths, ", ")},
			},
		}
		return app.render(w, res)
	}
	totalSize := int64(0)
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			return err
		}
		totalSize += fi.Size()
	}
	isTTY := output.IsTTY(os.Stderr)
	if shouldShowBar(isTTY, totalSize, progressForce, noProgressForce, app.outputMode, app.quiet, app.mustLogin) {
		bar := progress.NewBar(os.Stderr, totalSize*2)
		defer bar.Done()
		app.api = client.NewWithToken(app.api.ServerURL(), app.api.Token(), bar)

		bar.SetLabel("reading")
		fileData := make([][]byte, len(paths))
		for i, p := range paths {
			f, ferr := os.Open(p)
			if ferr != nil {
				return ferr
			}
			var buf bytes.Buffer
			_, cerr := io.Copy(&buf, progress.NewReader(f, bar))
			if ferr = f.Close(); ferr != nil {
				return ferr
			}
			if cerr != nil {
				return cerr
			}
			fileData[i] = buf.Bytes()
		}

		bar.Reset()
		bar.SetLabel("uploading")
		ids, uerr := uploadFileData(ctx, app, ch.Id, paths, fileData)
		if uerr != nil {
			return uerr
		}
		post := &model.Post{ChannelId: ch.Id, Message: message, FileIds: ids}
		created, cerr := app.api.CreatePost(ctx, post)
		if cerr != nil {
			return cerr
		}
		return app.render(w, output.Result{Text: created.Id})
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
	return app.render(w, output.Result{Text: created.Id})
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

func uploadFileData(ctx context.Context, app *appContext, channelID string, paths []string, data [][]byte) ([]string, error) {
	var ids []string
	for i, p := range paths {
		resp, err := app.api.UploadFile(ctx, data[i], channelID, filepath.Base(p))
		if err != nil {
			return nil, err
		}
		for _, fi := range resp.FileInfos {
			ids = append(ids, fi.Id)
		}
	}
	return ids, nil
}

func shouldShowBar(isTTY bool, totalSize int64, progressForce, noProgressForce bool, outputMode string, quiet bool, isEnvAuth bool) bool {
	if noProgressForce {
		return false
	}
	if progressForce {
		return true
	}
	if outputMode == "json" || outputMode == "ai" {
		return false
	}
	if quiet {
		return false
	}
	if isEnvAuth {
		return false
	}
	if !isTTY {
		return false
	}
	if totalSize <= 1*1024*1024 {
		return false
	}
	return true
}
