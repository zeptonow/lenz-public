package canvases

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"

	config "openreplay/backend/internal/config/canvases"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/messages"
	"openreplay/backend/pkg/metrics/canvas"
	"openreplay/backend/pkg/objectstorage"
	"openreplay/backend/pkg/pool"
	"openreplay/backend/pkg/queue/types"
)

type ImageStorage struct {
	cfg          *config.Config
	log          logger.Logger
	basePath     string
	saverPool    pool.WorkerPool
	packerPool   pool.WorkerPool
	uploaderPool pool.WorkerPool
	objStorage   objectstorage.ObjectStorage
	producer     types.Producer
	metrics      canvas.Canvas
}

type saveTask struct {
	ctx       context.Context
	sessionID uint64
	name      string
	image     *bytes.Buffer
}

type packTask struct {
	ctx       context.Context
	sessionID uint64
	path      string
	name      string
}

type uploadTask struct {
	ctx  context.Context
	path string
	name string
}

func New(cfg *config.Config, log logger.Logger, objStorage objectstorage.ObjectStorage, producer types.Producer, metrics canvas.Canvas) (*ImageStorage, error) {
	switch {
	case cfg == nil:
		return nil, fmt.Errorf("config is empty")
	case log == nil:
		return nil, fmt.Errorf("logger is empty")
	case objStorage == nil:
		return nil, fmt.Errorf("objectStorage is empty")
	case producer == nil:
		return nil, fmt.Errorf("producer is empty")
	case metrics == nil:
		return nil, fmt.Errorf("metrics is empty")
	}
	path := cfg.FSDir + "/"
	if cfg.CanvasDir != "" {
		path += cfg.CanvasDir + "/"
	}
	s := &ImageStorage{
		cfg:        cfg,
		log:        log,
		basePath:   path,
		objStorage: objStorage,
		producer:   producer,
		metrics:    metrics,
	}
	s.saverPool = pool.NewPool(2, 2, s.writeToDisk)
	s.packerPool = pool.NewPool(8, 16, s.packCanvas)
	s.uploaderPool = pool.NewPool(8, 16, s.sendToS3)
	return s, nil
}

func (v *ImageStorage) Wait() {
	v.saverPool.Pause()
	v.uploaderPool.Pause()
}

func (v *ImageStorage) SaveCanvasToDisk(ctx context.Context, sessID uint64, data []byte) error {
	type canvasData struct {
		Name string
		Data []byte
	}
	var msg = &canvasData{}
	if err := json.Unmarshal(data, msg); err != nil {
		return fmt.Errorf("can't parse canvas message, err: %s", err)
	}
	v.saverPool.Submit(&saveTask{ctx: ctx, sessionID: sessID, name: msg.Name, image: bytes.NewBuffer(msg.Data)})
	return nil
}

func (v *ImageStorage) writeToDisk(payload interface{}) {
	task := payload.(*saveTask)
	path := filepath.Join(v.basePath, strconv.FormatUint(task.sessionID, 10))

	// Ensure the directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		v.log.Fatal(task.ctx, "can't create a dir, err: %s", err)
	}

	// Write images to disk
	baseName := filepath.Base(task.name)
	if baseName == "." || baseName == ".." || baseName != task.name ||
		strings.Contains(baseName, "/") || strings.Contains(baseName, "\\") {
		v.log.Warn(task.ctx, "skipping suspicious canvas name: %q", task.name)
		return
	}
	outFile, err := os.Create(filepath.Join(path, baseName))
	if err != nil {
		v.log.Fatal(task.ctx, "can't create an image: %s", err)
	}
	if _, err := io.Copy(outFile, task.image); err != nil {
		v.log.Fatal(task.ctx, "can't copy data to image: %s", err)
	}
	if outFile != nil {
		if err := outFile.Close(); err != nil {
			v.log.Warn(task.ctx, "can't close out file: %s", err)
		}
	}
	v.metrics.RecordCanvasImageSize(float64(task.image.Len()))
	v.metrics.IncreaseTotalSavedImages()

	v.log.Debug(task.ctx, "canvas image saved, name: %s, size: %3.3f mb", task.name, float64(task.image.Len())/1024.0/1024.0)
	return
}

func (v *ImageStorage) PrepareSessionCanvases(ctx context.Context, sessID uint64) error {
	start := time.Now()
	path := fmt.Sprintf("%s%d/", v.basePath, sessID)

	// Check that the directory exists
	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}

	// Build the list of canvas images sets
	names := make(map[string]int)
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".tar.zst") {
			continue // Skip already created archives
		}
		name := strings.Split(file.Name(), ".")
		parts := strings.Split(name[0], "_")
		if len(name) != 2 || len(parts) != 3 {
			v.log.Warn(ctx, "unknown file name: %s, skipping", file.Name())
			continue
		}
		canvasID := fmt.Sprintf("%s_%s", parts[0], parts[1])
		names[canvasID]++
	}

	for name, number := range names {
		msg := &messages.CustomEvent{
			Name:    name,
			Payload: path,
		}
		if err := v.producer.Produce(v.cfg.TopicCanvasTrigger, sessID, msg.Encode()); err != nil {
			v.log.Error(ctx, "can't send canvas trigger: %s", err)
		}
		v.metrics.RecordImagesPerCanvas(float64(number))
	}
	v.metrics.RecordCanvasesPerSession(float64(len(names)))
	v.metrics.RecordPreparingDuration(time.Since(start).Seconds())

	v.log.Debug(ctx, "session canvases (%d) prepared in %.3fs, session: %d", len(names), time.Since(start).Seconds(), sessID)
	return nil
}

func (v *ImageStorage) ProcessSessionCanvas(ctx context.Context, sessID uint64, path, name string) error {
	v.packerPool.Submit(&packTask{ctx: ctx, sessionID: sessID, path: path, name: name})
	return nil
}

func (v *ImageStorage) packCanvas(payload interface{}) {
	task := payload.(*packTask)
	start := time.Now()
	sessionID := strconv.FormatUint(task.sessionID, 10)

	// Save to archives
	dir := filepath.Clean(task.path)
	base := filepath.Clean(v.basePath)
	baseWithSep := base + string(os.PathSeparator)
	if !(dir == base || strings.HasPrefix(dir, baseWithSep)) {
		v.log.Warn(task.ctx, "refusing to pack canvases from unexpected path: %q", dir)
		return
	}

	prefix := filepath.Base(task.name)
	if prefix == "." || prefix == ".." || prefix != task.name ||
		strings.Contains(prefix, "/") || strings.Contains(prefix, "\\") {
		v.log.Warn(task.ctx, "refusing to pack canvases with suspicious prefix: %q", task.name)
		return
	}

	// Replay-only build: avoid `sh -c` pipelines (command injection risk).
	// Old implementation (kept for later):
	// archPath := fmt.Sprintf("%s%s.tar.zst", task.path, task.name)
	// fullCmd := fmt.Sprintf("find %s -type f -name '%s*' ! -name '*.tar.zst' | tar -cf - --files-from=- | zstd -f -o %s",
	// 	task.path, task.name, archPath)
	// cmd := exec.Command("sh", "-c", fullCmd)
	// var stdout, stderr bytes.Buffer
	// cmd.Stdout = &stdout
	// cmd.Stderr = &stderr
	// err := cmd.Run()
	// if err != nil {
	// 	v.log.Warn(task.ctx, "failed to execute command, err: %s, stderr: %v", err, stderr.String())
	// }

	entries, err := os.ReadDir(dir)
	if err != nil {
		v.log.Warn(task.ctx, "failed to read canvas dir: %s", err)
		return
	}
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(n, ".tar.zst") {
			continue
		}
		if strings.HasPrefix(n, prefix) {
			files = append(files, n)
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		return
	}

	archPath := filepath.Join(dir, prefix+".tar.zst")
	out, err := os.Create(archPath)
	if err != nil {
		v.log.Warn(task.ctx, "failed to create canvas archive: %s", err)
		return
	}
	defer func() { _ = out.Close() }()

	zw, err := zstd.NewWriter(out)
	if err != nil {
		v.log.Warn(task.ctx, "failed to init zstd writer: %s", err)
		return
	}
	defer zw.Close()

	tw := tar.NewWriter(zw)
	defer tw.Close()

	for _, name := range files {
		p := filepath.Join(dir, name)
		fi, err := os.Stat(p)
		if err != nil {
			v.log.Warn(task.ctx, "failed to stat %q: %s", name, err)
			return
		}
		hdr, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			v.log.Warn(task.ctx, "failed to create tar header for %q: %s", name, err)
			return
		}
		hdr.Name = name
		if err := tw.WriteHeader(hdr); err != nil {
			v.log.Warn(task.ctx, "failed to write tar header for %q: %s", name, err)
			return
		}
		f, err := os.Open(p)
		if err != nil {
			v.log.Warn(task.ctx, "failed to open %q: %s", name, err)
			return
		}
		_, copyErr := io.Copy(tw, f)
		_ = f.Close()
		if copyErr != nil {
			v.log.Warn(task.ctx, "failed to write %q to archive: %s", name, copyErr)
			return
		}
	}

	v.metrics.RecordArchivingDuration(time.Since(start).Seconds())
	v.metrics.IncreaseTotalCreatedArchives()

	v.log.Debug(task.ctx, "canvas packed successfully in %.3fs, session: %d", time.Since(start).Seconds(), task.sessionID)
	v.uploaderPool.Submit(&uploadTask{ctx: task.ctx, path: archPath, name: sessionID + "/" + task.name + ".tar.zst"})
}

func (v *ImageStorage) sendToS3(payload interface{}) {
	task := payload.(*uploadTask)
	start := time.Now()
	video, err := os.ReadFile(task.path)
	if err != nil {
		v.log.Fatal(task.ctx, "failed to read canvas archive: %s", err)
	}
	if err := v.objStorage.Upload(bytes.NewReader(video), task.name, "application/octet-stream", objectstorage.NoContentEncoding, objectstorage.Zstd); err != nil {
		v.log.Fatal(task.ctx, "failed to upload canvas to storage: %s", err)
	}
	v.metrics.RecordUploadingDuration(time.Since(start).Seconds())
	v.metrics.RecordArchiveSize(float64(len(video)))

	v.log.Debug(task.ctx, "replay file (size: %d) uploaded successfully in %.3fs", len(video), time.Since(start).Seconds())
}
