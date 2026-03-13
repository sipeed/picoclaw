package feishu

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type feishuMultipartDriveClient interface {
	InitiateMultipartUpload(context.Context, string, string, int64) (*MultipartUploadSession, error)
	UploadMultipartChunk(context.Context, string, int, []byte) error
	CompleteMultipartUpload(context.Context, string, int) (*DriveFileSummary, error)
}

func (c *FeishuChannel) UploadLargeDriveFile(ctx context.Context, parentToken, name string, r io.Reader, size int64) (*DriveFileSummary, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("file name is empty")
	}
	if r == nil {
		return nil, fmt.Errorf("file reader is nil")
	}
	if size <= 0 {
		return nil, fmt.Errorf("file size must be > 0")
	}
	client, ok := any(c).(feishuMultipartDriveClient)
	if !ok {
		return nil, fmt.Errorf("multipart drive upload is not supported in this build")
	}
	session, err := client.InitiateMultipartUpload(ctx, parentToken, name, size)
	if err != nil {
		return nil, err
	}
	blockSize := session.BlockSize
	if blockSize <= 0 {
		blockSize = 4 * 1024 * 1024
	}
	chunks, err := uploadMultipartFromReader(ctx, client, session.UploadID, r, blockSize)
	if err != nil {
		return nil, err
	}
	if chunks == 0 {
		return nil, fmt.Errorf("no multipart chunks uploaded")
	}
	return client.CompleteMultipartUpload(ctx, session.UploadID, chunks)
}

func (c *FeishuChannel) UploadLargeDriveFileFromPath(ctx context.Context, parentToken, path string) (*DriveFileSummary, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("file path is empty")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory: %s", path)
	}
	return c.UploadLargeDriveFile(ctx, parentToken, filepath.Base(path), file, info.Size())
}

func (c *FeishuChannel) uploadMultipartFromReader(ctx context.Context, uploadID string, r io.Reader, blockSize int) (int, error) {
	client, ok := any(c).(feishuMultipartDriveClient)
	if !ok {
		return 0, fmt.Errorf("multipart drive upload is not supported in this build")
	}
	return uploadMultipartFromReader(ctx, client, uploadID, r, blockSize)
}

func uploadMultipartFromReader(ctx context.Context, client feishuMultipartDriveClient, uploadID string, r io.Reader, blockSize int) (int, error) {
	if strings.TrimSpace(uploadID) == "" {
		return 0, fmt.Errorf("upload ID is empty")
	}
	if r == nil {
		return 0, fmt.Errorf("file reader is nil")
	}
	if blockSize <= 0 {
		return 0, fmt.Errorf("block size must be > 0")
	}
	buf := make([]byte, blockSize)
	seq := 0
	for {
		n, err := io.ReadFull(r, buf)
		if err == io.EOF {
			break
		}
		if err != nil && err != io.ErrUnexpectedEOF {
			return seq, err
		}
		if n <= 0 {
			break
		}
		chunk := append([]byte(nil), buf[:n]...)
		if err := client.UploadMultipartChunk(ctx, uploadID, seq, chunk); err != nil {
			return seq, err
		}
		seq++
		if err == io.ErrUnexpectedEOF {
			break
		}
	}
	return seq, nil
}
