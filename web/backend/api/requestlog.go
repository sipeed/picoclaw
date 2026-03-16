package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/requestlog"
)

func (h *Handler) registerRequestLogRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/logs/requests", h.handleGetRequestLogs)
	mux.HandleFunc("GET /api/stats/requests", h.handleGetRequestStats)
	mux.HandleFunc("GET /api/logs/requests/export", h.handleExportRequestLogs)
	mux.HandleFunc("POST /api/logs/requests/archive-now", h.handleArchiveNow)
	mux.HandleFunc("GET /api/config/requestlog", h.handleGetRequestLogConfig)
	mux.HandleFunc("PUT /api/config/requestlog", h.handlePutRequestLogConfig)
}

func (h *Handler) loadRequestLogConfig() requestlog.Config {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return requestlog.DefaultConfig()
	}
	return h.toRequestLogConfig(cfg.RequestLog)
}

func (h *Handler) saveRequestLogConfig(rlCfg requestlog.Config) error {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return err
	}
	cfg.RequestLog = h.fromRequestLogConfig(rlCfg)
	return config.SaveConfig(h.configPath, cfg)
}

func (h *Handler) toRequestLogConfig(cfg config.RequestLogConfig) requestlog.Config {
	if !cfg.Enabled && cfg.MaxFileSizeMB == 0 {
		return requestlog.DefaultConfig()
	}
	return requestlog.Config{
		Enabled:             cfg.Enabled,
		LogDir:              cfg.LogDir,
		MaxFileSizeMB:       cfg.MaxFileSizeMB,
		MaxFiles:            cfg.MaxFiles,
		RetentionDays:       cfg.RetentionDays,
		ArchiveInterval:     cfg.ArchiveInterval,
		CompressArchive:     cfg.CompressArchive,
		LogContentMaxLength: cfg.LogContentMaxLength,
		RecordMedia:         cfg.RecordMedia,
	}
}

func (h *Handler) fromRequestLogConfig(cfg requestlog.Config) config.RequestLogConfig {
	return config.RequestLogConfig{
		Enabled:             cfg.Enabled,
		LogDir:              cfg.LogDir,
		MaxFileSizeMB:       cfg.MaxFileSizeMB,
		MaxFiles:            cfg.MaxFiles,
		RetentionDays:       cfg.RetentionDays,
		ArchiveInterval:     cfg.ArchiveInterval,
		CompressArchive:     cfg.CompressArchive,
		LogContentMaxLength: cfg.LogContentMaxLength,
		RecordMedia:         cfg.RecordMedia,
	}
}

func (h *Handler) handleGetRequestLogs(w http.ResponseWriter, r *http.Request) {
	if h.requestLogReader == nil {
		http.Error(w, "request log not available", http.StatusServiceUnavailable)
		return
	}

	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	channel := r.URL.Query().Get("channel")
	senderID := r.URL.Query().Get("sender_id")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	var startTime, endTime time.Time
	var err error

	if startStr != "" {
		startTime, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			http.Error(w, "invalid start time format", http.StatusBadRequest)
			return
		}
	}

	if endStr != "" {
		endTime, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			http.Error(w, "invalid end time format", http.StatusBadRequest)
			return
		}
	}

	limit := 100
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}

	offset := 0
	if offsetStr != "" {
		offset, _ = strconv.Atoi(offsetStr)
	}

	opts := requestlog.QueryOptions{
		StartTime: startTime,
		EndTime:   endTime,
		Channel:   channel,
		SenderID:  senderID,
		Limit:     limit,
		Offset:    offset,
	}

	records, err := h.requestLogReader.Query(opts)
	if err != nil {
		http.Error(w, "failed to query logs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"records": records,
		"limit":   limit,
		"offset":  offset,
	})
}

func (h *Handler) handleGetRequestStats(w http.ResponseWriter, r *http.Request) {
	if h.requestLogReader == nil {
		http.Error(w, "request log not available", http.StatusServiceUnavailable)
		return
	}

	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	var startTime, endTime time.Time
	var err error

	now := time.Now()
	if startStr != "" {
		startTime, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			http.Error(w, "invalid start time format", http.StatusBadRequest)
			return
		}
	} else {
		startTime = now.AddDate(0, 0, -30)
	}

	if endStr != "" {
		endTime, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			http.Error(w, "invalid end time format", http.StatusBadRequest)
			return
		}
	} else {
		endTime = now
	}

	stats, err := h.requestLogReader.GetStats(startTime, endTime)
	if err != nil {
		http.Error(w, "failed to get stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *Handler) handleExportRequestLogs(w http.ResponseWriter, r *http.Request) {
	if h.requestLogReader == nil {
		http.Error(w, "request log not available", http.StatusServiceUnavailable)
		return
	}

	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	channel := r.URL.Query().Get("channel")
	senderID := r.URL.Query().Get("sender_id")
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	var startTime, endTime time.Time
	var err error

	if startStr != "" {
		startTime, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			http.Error(w, "invalid start time format", http.StatusBadRequest)
			return
		}
	}

	if endStr != "" {
		endTime, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			http.Error(w, "invalid end time format", http.StatusBadRequest)
			return
		}
	}

	opts := requestlog.QueryOptions{
		StartTime: startTime,
		EndTime:   endTime,
		Channel:   channel,
		SenderID:  senderID,
		Limit:     10000,
	}

	records, err := h.requestLogReader.Query(opts)
	if err != nil {
		http.Error(w, "failed to query logs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	timestamp := time.Now().Format("2006-01-02-150405")
	filename := fmt.Sprintf("request-logs-%s", timestamp)

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", filename))
		h.exportCSV(w, records)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", filename))
		json.NewEncoder(w).Encode(records)
	}
}

func (h *Handler) exportCSV(w http.ResponseWriter, records []requestlog.RequestRecord) {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{
		"timestamp",
		"request_id",
		"channel",
		"sender_id",
		"chat_id",
		"content",
		"content_length",
		"message_id",
		"media_count",
		"session_key",
		"processing_time_ms",
	}
	writer.Write(header)

	for _, r := range records {
		row := []string{
			r.Timestamp.Format(time.RFC3339),
			r.RequestID,
			r.Channel,
			r.SenderID,
			r.ChatID,
			r.Content,
			strconv.Itoa(r.ContentLength),
			r.MessageID,
			strconv.Itoa(r.MediaCount),
			r.SessionKey,
			strconv.FormatInt(r.ProcessingTime, 10),
		}
		writer.Write(row)
	}
}

func (h *Handler) handleArchiveNow(w http.ResponseWriter, r *http.Request) {
	var cfg requestlog.Config
	var logDir string

	if h.requestLogger != nil {
		cfg = h.requestLogger.GetConfig()
		logDir = h.requestLogger.LogDir()
	} else {
		cfg = h.loadRequestLogConfig()
		if h.workspacePath != "" {
			logDir = filepath.Join(h.workspacePath, "logs", "requests")
		} else {
			http.Error(w, "workspace path not configured", http.StatusServiceUnavailable)
			return
		}
	}

	archiver := requestlog.NewArchiver(cfg, logDir)
	if err := archiver.ArchiveAll(); err != nil {
		http.Error(w, "archive failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func (h *Handler) handleGetRequestLogConfig(w http.ResponseWriter, r *http.Request) {
	var cfg requestlog.Config
	if h.requestLogger != nil {
		cfg = h.requestLogger.GetConfig()
	} else {
		cfg = h.loadRequestLogConfig()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

func (h *Handler) handlePutRequestLogConfig(w http.ResponseWriter, r *http.Request) {
	var cfg requestlog.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if h.requestLogger != nil {
		if err := h.requestLogger.UpdateConfig(cfg); err != nil {
			http.Error(w, "failed to update config: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.saveRequestLogConfig(cfg); err != nil {
			http.Error(w, "failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}
