package requestlog

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
)

type Config struct {
	Enabled             bool   `json:"enabled"`
	LogDir              string `json:"log_dir"`
	MaxFileSizeMB       int    `json:"max_file_size_mb"`
	MaxFiles            int    `json:"max_files"`
	RetentionDays       int    `json:"retention_days"`
	ArchiveInterval     string `json:"archive_interval"`
	CompressArchive     bool   `json:"compress_archive"`
	LogContentMaxLength int    `json:"log_content_max_length"`
	RecordMedia         bool   `json:"record_media"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:             true,
		LogDir:              "logs/requests",
		MaxFileSizeMB:       100,
		MaxFiles:            100,
		RetentionDays:       30,
		ArchiveInterval:     "24h",
		CompressArchive:     true,
		LogContentMaxLength: 1000,
		RecordMedia:         false,
	}
}

type RequestRecord struct {
	Timestamp      time.Time      `json:"timestamp"`
	RequestID      string         `json:"request_id"`
	Channel        string         `json:"channel"`
	SenderID       string         `json:"sender_id"`
	SenderInfo     bus.SenderInfo `json:"sender_info"`
	ChatID         string         `json:"chat_id"`
	Content        string         `json:"content"`
	ContentLength  int            `json:"content_length"`
	Peer           bus.Peer       `json:"peer"`
	MessageID      string         `json:"message_id"`
	MediaCount     int            `json:"media_count"`
	SessionKey     string         `json:"session_key"`
	ProcessingTime int64          `json:"processing_time_ms"`
	Response       string         `json:"response,omitempty"`
	ResponseLength int            `json:"response_length,omitempty"`
}

type Logger struct {
	config  Config
	msgBus  *bus.MessageBus
	storage *Storage
	mu      sync.Mutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	closed  chan struct{}
}

func (l *Logger) LogDir() string {
	return l.storage.logDir
}

func NewLogger(cfg Config, msgBus *bus.MessageBus, workspacePath string) *Logger {
	if cfg.LogDir == "" {
		cfg.LogDir = "logs/requests"
	}
	logDir := filepath.Join(workspacePath, cfg.LogDir)

	ctx, cancel := context.WithCancel(context.Background())

	return &Logger{
		config:  cfg,
		msgBus:  msgBus,
		storage: NewStorage(logDir, cfg.MaxFileSizeMB),
		running: false,
		ctx:     ctx,
		cancel:  cancel,
		closed:  make(chan struct{}),
	}
}

type Reader struct {
	config  Config
	storage *Storage
}

func NewReader(logDir string, maxFileSizeMB int) *Reader {
	return &Reader{
		config:  DefaultConfig(),
		storage: NewStorage(logDir, maxFileSizeMB),
	}
}

func (r *Reader) Query(opts QueryOptions) ([]RequestRecord, error) {
	return r.storage.Query(opts)
}

func (r *Reader) GetStats(startTime, endTime time.Time) (map[string]any, error) {
	opts := QueryOptions{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000,
	}

	records, err := r.storage.Query(opts)
	if err != nil {
		return nil, err
	}

	return calculateStats(records), nil
}

func (l *Logger) GetConfig() Config {
	return l.config
}

func (l *Logger) UpdateConfig(cfg Config) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config = cfg
	return nil
}

func (l *Logger) Start() error {
	if l.config.Enabled {
		l.mu.Lock()
		l.running = true
		l.mu.Unlock()

		if err := l.storage.Init(); err != nil {
			logger.ErrorCF("requestlog", "Failed to initialize storage", map[string]any{
				"error": err.Error(),
			})
			return err
		}

		go l.consumeLoop()

		logger.InfoCF("requestlog", "Request logger started", map[string]any{
			"log_dir": l.storage.logDir,
		})
	}
	return nil
}

func (l *Logger) Stop() error {
	l.mu.Lock()
	if !l.running {
		l.mu.Unlock()
		return nil
	}
	l.running = false
	l.mu.Unlock()

	l.cancel()

	<-l.closed

	if err := l.storage.Close(); err != nil {
		logger.ErrorCF("requestlog", "Failed to close storage", map[string]any{
			"error": err.Error(),
		})
		return err
	}

	logger.InfoCF("requestlog", "Request logger stopped", nil)
	return nil
}

func (l *Logger) consumeLoop() {
	defer close(l.closed)

	inboundCh, unsubInbound := l.msgBus.SubscribeInbound(l.ctx)
	defer unsubInbound()

	for {
		select {
		case <-l.ctx.Done():
			return
		case msg, ok := <-inboundCh:
			if !ok {
				return
			}
			l.recordInbound(msg)
		}
	}
}

func (l *Logger) recordInbound(msg bus.InboundMessage) {
	content := msg.Content
	if l.config.LogContentMaxLength > 0 && len(content) > l.config.LogContentMaxLength {
		content = content[:l.config.LogContentMaxLength]
	}

	record := RequestRecord{
		Timestamp:     time.Now(),
		RequestID:     uuid.New().String(),
		Channel:       msg.Channel,
		SenderID:      msg.SenderID,
		SenderInfo:    msg.Sender,
		ChatID:        msg.ChatID,
		Content:       content,
		ContentLength: len(msg.Content),
		Peer:          msg.Peer,
		MessageID:     msg.MessageID,
		MediaCount:    len(msg.Media),
		SessionKey:    msg.SessionKey,
	}

	l.writeRecord(record)
}

func (l *Logger) writeRecord(record RequestRecord) {
	data, err := json.Marshal(record)
	if err != nil {
		logger.ErrorCF("requestlog", "Failed to marshal record", map[string]any{
			"error": err.Error(),
		})
		return
	}

	if err := l.storage.Write(data); err != nil {
		logger.ErrorCF("requestlog", "Failed to write record", map[string]any{
			"error": err.Error(),
		})
	}
}

type Storage struct {
	logDir      string
	maxFileSize int64
	currentFile *os.File
	currentSize int64
	mu          sync.Mutex
}

func NewStorage(logDir string, maxFileSizeMB int) *Storage {
	return &Storage{
		logDir:      logDir,
		maxFileSize: int64(maxFileSizeMB) * 1024 * 1024,
	}
}

func (s *Storage) Init() error {
	if err := os.MkdirAll(s.logDir, 0o755); err != nil {
		return err
	}
	return s.rotateFile()
}

func (s *Storage) Write(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currentFile == nil {
		if err := s.rotateFile(); err != nil {
			return err
		}
	}

	if s.currentSize >= s.maxFileSize {
		if err := s.rotateFile(); err != nil {
			return err
		}
	}

	n, err := s.currentFile.Write(append(data, '\n'))
	if err != nil {
		return err
	}

	s.currentSize += int64(n)
	return nil
}

func (s *Storage) rotateFile() error {
	if s.currentFile != nil {
		s.currentFile.Close()
		s.currentFile = nil
	}

	dateStr := time.Now().Format("2006-01-02")
	filename := filepath.Join(s.logDir, "requests-"+dateStr+".jsonl")

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}

	s.currentFile = f
	s.currentSize = stat.Size()
	return nil
}

func (s *Storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.currentFile != nil {
		return s.currentFile.Close()
	}
	return nil
}

func calculateStats(records []RequestRecord) map[string]any {
	byChannel := make(map[string]int)
	byDay := make(map[string]int)
	topSenders := make(map[string]int)

	for _, rec := range records {
		byChannel[rec.Channel]++

		day := rec.Timestamp.Format("2006-01-02")
		byDay[day]++

		senderKey := rec.SenderID + ":" + rec.Channel
		topSenders[senderKey]++
	}

	result := map[string]any{
		"total":      len(records),
		"by_channel": byChannel,
		"by_day":     byDay,
	}

	type senderStat struct {
		Sender  string `json:"sender"`
		Channel string `json:"channel"`
		Count   int    `json:"count"`
	}

	var topList []senderStat
	for k, v := range topSenders {
		parts := strings.SplitN(k, ":", 2)
		if len(parts) == 2 {
			topList = append(topList, senderStat{
				Sender:  parts[0],
				Channel: parts[1],
				Count:   v,
			})
		}
	}

	sort.Slice(topList, func(i, j int) bool {
		return topList[i].Count > topList[j].Count
	})

	if len(topList) > 10 {
		topList = topList[:10]
	}
	result["top_senders"] = topList

	return result
}

type QueryOptions struct {
	StartTime time.Time
	EndTime   time.Time
	Channel   string
	SenderID  string
	Limit     int
	Offset    int
}

func (l *Logger) Query(opts QueryOptions) ([]RequestRecord, error) {
	return l.storage.Query(opts)
}

func (s *Storage) Query(opts QueryOptions) ([]RequestRecord, error) {
	files, err := os.ReadDir(s.logDir)
	if err != nil {
		return nil, err
	}

	var results []RequestRecord
	offset := opts.Offset
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if !strings.HasSuffix(name, ".jsonl") && !strings.HasSuffix(name, ".tar.gz") {
			continue
		}

		records, err := s.readFile(filepath.Join(s.logDir, name), opts)
		if err != nil {
			continue
		}

		for _, record := range records {
			if offset > 0 {
				offset--
				continue
			}
			if len(results) >= limit {
				break
			}
			results = append(results, record)
		}

		if len(results) >= limit {
			break
		}
	}

	return results, nil
}

func (s *Storage) readFile(path string, opts QueryOptions) ([]RequestRecord, error) {
	if strings.HasSuffix(path, ".tar.gz") {
		return s.readTarGz(path, opts)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var results []RequestRecord

	dec := json.NewDecoder(f)
	for dec.More() {
		var record RequestRecord
		if err := dec.Decode(&record); err != nil {
			continue
		}

		if !opts.StartTime.IsZero() && record.Timestamp.Before(opts.StartTime) {
			continue
		}
		if !opts.EndTime.IsZero() && record.Timestamp.After(opts.EndTime) {
			continue
		}
		if opts.Channel != "" && record.Channel != opts.Channel {
			continue
		}
		if opts.SenderID != "" && record.SenderID != opts.SenderID {
			continue
		}

		results = append(results, record)
	}

	return results, nil
}

func (s *Storage) readTarGz(path string, opts QueryOptions) ([]RequestRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	var results []RequestRecord

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		dec := json.NewDecoder(tarReader)
		for dec.More() {
			var record RequestRecord
			if err := dec.Decode(&record); err != nil {
				continue
			}

			if !opts.StartTime.IsZero() && record.Timestamp.Before(opts.StartTime) {
				continue
			}
			if !opts.EndTime.IsZero() && record.Timestamp.After(opts.EndTime) {
				continue
			}
			if opts.Channel != "" && record.Channel != opts.Channel {
				continue
			}
			if opts.SenderID != "" && record.SenderID != opts.SenderID {
				continue
			}

			results = append(results, record)
		}
	}

	return results, nil
}

func (l *Logger) GetStats(startTime, endTime time.Time) (map[string]any, error) {
	opts := QueryOptions{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000,
	}

	records, err := l.storage.Query(opts)
	if err != nil {
		return nil, err
	}

	return calculateStats(records), nil
}
