package tools

import "context"

type FeishuMessageGetter interface {
	GetMessage(ctx context.Context, messageID string) (any, error)
}

type FeishuMessageLister interface {
	ListMessages(ctx context.Context, containerID, containerType string, pageSize int, pageToken string) (any, error)
}

type FeishuMessageReplier interface {
	ReplyMessage(ctx context.Context, messageID, text string) error
}

type FeishuShareLinkResolver interface {
	GetMessageFromShareLink(ctx context.Context, shareLink string) (any, error)
}

type FeishuUserGetter interface {
	GetUserInfo(ctx context.Context, userID string) (any, error)
}

type FeishuUserLister interface {
	ListUsers(ctx context.Context, pageSize int, userIDType, pageToken string) (any, error)
}

type FeishuUserLookup interface {
	GetUserIDByEmail(ctx context.Context, email string) (string, error)
	GetUserIDByMobile(ctx context.Context, mobile string) (string, error)
}

type FeishuGroupCreator interface {
	CreateGroup(ctx context.Context, name string) (any, error)
}

type FeishuGroupGetter interface {
	GetGroupInfo(ctx context.Context, chatID string) (any, error)
}

type FeishuGroupMemberLister interface {
	ListGroupMembers(ctx context.Context, chatID string, pageSize int, pageToken string) (any, error)
}

type FeishuGroupLister interface {
	ListGroups(ctx context.Context, pageSize int, pageToken string) (any, error)
}

type FeishuGroupMessageSender interface {
	SendGroupMessage(ctx context.Context, chatID, text string) error
}

type FeishuDriveRootGetter interface {
	GetDriveRootFolder(ctx context.Context) (any, error)
}

type FeishuDriveFolderGetter interface {
	GetDriveFolder(ctx context.Context, folderToken string) (any, error)
}

type FeishuDriveFileGetter interface {
	GetDriveFile(ctx context.Context, fileToken string) (any, error)
}

type FeishuDriveFileLister interface {
	ListDriveFiles(ctx context.Context, folderToken, pageToken string, pageSize int) (any, error)
}

type FeishuDriveFileDownloader interface {
	DownloadDriveFile(ctx context.Context, fileToken string) (any, error)
}

type FeishuDriveFileDeleter interface {
	DeleteDriveFile(ctx context.Context, fileToken string) error
}

type FeishuDriveFileUploader interface {
	UploadDriveFile(ctx context.Context, parentToken, name string, data []byte) (any, error)
}

type FeishuMultipartInitiator interface {
	InitiateMultipartUpload(ctx context.Context, parentToken, name string, size int64) (any, error)
}

type FeishuMultipartChunkUploader interface {
	UploadMultipartChunk(ctx context.Context, uploadID string, seq int, data []byte) error
}

type FeishuMultipartCompleter interface {
	CompleteMultipartUpload(ctx context.Context, uploadID string, blockNum int) (any, error)
}

type FeishuImageSender interface {
	SendImageMessage(ctx context.Context, chatID string, data []byte, fileName string) error
}

type FeishuFileSender interface {
	SendFileMessage(ctx context.Context, chatID string, data []byte, fileName, fileType string) error
}

type FeishuChannelAdapter struct {
	messageGetter        FeishuMessageGetter
	messageLister        FeishuMessageLister
	messageReplier       FeishuMessageReplier
	shareLinkResolver    FeishuShareLinkResolver
	userGetter           FeishuUserGetter
	userLister           FeishuUserLister
	userLookup           FeishuUserLookup
	groupCreator         FeishuGroupCreator
	groupGetter          FeishuGroupGetter
	groupMemberLister    FeishuGroupMemberLister
	groupLister          FeishuGroupLister
	groupMessageSender   FeishuGroupMessageSender
	driveRootGetter      FeishuDriveRootGetter
	driveFolderGetter    FeishuDriveFolderGetter
	driveGetter          FeishuDriveFileGetter
	driveLister          FeishuDriveFileLister
	driveDownloader      FeishuDriveFileDownloader
	driveDeleter         FeishuDriveFileDeleter
	driveUploader        FeishuDriveFileUploader
	multipartInitiator   FeishuMultipartInitiator
	multipartUploader    FeishuMultipartChunkUploader
	multipartCompleter   FeishuMultipartCompleter
	imageSender          FeishuImageSender
	fileSender           FeishuFileSender
}

func NewFeishuChannelAdapter(target any) *FeishuChannelAdapter {
	adapter := &FeishuChannelAdapter{}
	if v, ok := target.(FeishuMessageGetter); ok { adapter.messageGetter = v }
	if v, ok := target.(FeishuMessageLister); ok { adapter.messageLister = v }
	if v, ok := target.(FeishuMessageReplier); ok { adapter.messageReplier = v }
	if v, ok := target.(FeishuShareLinkResolver); ok { adapter.shareLinkResolver = v }
	if v, ok := target.(FeishuUserGetter); ok { adapter.userGetter = v }
	if v, ok := target.(FeishuUserLister); ok { adapter.userLister = v }
	if v, ok := target.(FeishuUserLookup); ok { adapter.userLookup = v }
	if v, ok := target.(FeishuGroupCreator); ok { adapter.groupCreator = v }
	if v, ok := target.(FeishuGroupGetter); ok { adapter.groupGetter = v }
	if v, ok := target.(FeishuGroupMemberLister); ok { adapter.groupMemberLister = v }
	if v, ok := target.(FeishuGroupLister); ok { adapter.groupLister = v }
	if v, ok := target.(FeishuGroupMessageSender); ok { adapter.groupMessageSender = v }
	if v, ok := target.(FeishuDriveRootGetter); ok { adapter.driveRootGetter = v }
	if v, ok := target.(FeishuDriveFolderGetter); ok { adapter.driveFolderGetter = v }
	if v, ok := target.(FeishuDriveFileGetter); ok { adapter.driveGetter = v }
	if v, ok := target.(FeishuDriveFileLister); ok { adapter.driveLister = v }
	if v, ok := target.(FeishuDriveFileDownloader); ok { adapter.driveDownloader = v }
	if v, ok := target.(FeishuDriveFileDeleter); ok { adapter.driveDeleter = v }
	if v, ok := target.(FeishuDriveFileUploader); ok { adapter.driveUploader = v }
	if v, ok := target.(FeishuMultipartInitiator); ok { adapter.multipartInitiator = v }
	if v, ok := target.(FeishuMultipartChunkUploader); ok { adapter.multipartUploader = v }
	if v, ok := target.(FeishuMultipartCompleter); ok { adapter.multipartCompleter = v }
	if v, ok := target.(FeishuImageSender); ok { adapter.imageSender = v }
	if v, ok := target.(FeishuFileSender); ok { adapter.fileSender = v }
	return adapter
}

func (a *FeishuChannelAdapter) GetMessage(ctx context.Context, messageID string) (any, error) {
	if a == nil || a.messageGetter == nil { return nil, errFeishuAdapterNotReady("GetMessage") }
	return a.messageGetter.GetMessage(ctx, messageID)
}

func (a *FeishuChannelAdapter) ListMessages(ctx context.Context, containerID, containerType string, pageSize int, pageToken string) (any, error) {
	if a == nil || a.messageLister == nil { return nil, errFeishuAdapterNotReady("ListMessages") }
	return a.messageLister.ListMessages(ctx, containerID, containerType, pageSize, pageToken)
}

func (a *FeishuChannelAdapter) ReplyMessage(ctx context.Context, messageID, text string) error {
	if a == nil || a.messageReplier == nil { return errFeishuAdapterNotReady("ReplyMessage") }
	return a.messageReplier.ReplyMessage(ctx, messageID, text)
}

func (a *FeishuChannelAdapter) GetMessageFromShareLink(ctx context.Context, shareLink string) (any, error) {
	if a == nil || a.shareLinkResolver == nil { return nil, errFeishuAdapterNotReady("GetMessageFromShareLink") }
	return a.shareLinkResolver.GetMessageFromShareLink(ctx, shareLink)
}

func (a *FeishuChannelAdapter) GetUserInfo(ctx context.Context, userID string) (any, error) {
	if a == nil || a.userGetter == nil { return nil, errFeishuAdapterNotReady("GetUserInfo") }
	return a.userGetter.GetUserInfo(ctx, userID)
}

func (a *FeishuChannelAdapter) ListUsers(ctx context.Context, pageSize int, userIDType, pageToken string) (any, error) {
	if a == nil || a.userLister == nil { return nil, errFeishuAdapterNotReady("ListUsers") }
	return a.userLister.ListUsers(ctx, pageSize, userIDType, pageToken)
}

func (a *FeishuChannelAdapter) GetUserIDByEmail(ctx context.Context, email string) (string, error) {
	if a == nil || a.userLookup == nil { return "", errFeishuAdapterNotReady("GetUserIDByEmail") }
	return a.userLookup.GetUserIDByEmail(ctx, email)
}

func (a *FeishuChannelAdapter) GetUserIDByMobile(ctx context.Context, mobile string) (string, error) {
	if a == nil || a.userLookup == nil { return "", errFeishuAdapterNotReady("GetUserIDByMobile") }
	return a.userLookup.GetUserIDByMobile(ctx, mobile)
}

func (a *FeishuChannelAdapter) CreateGroup(ctx context.Context, name string) (any, error) {
	if a == nil || a.groupCreator == nil { return nil, errFeishuAdapterNotReady("CreateGroup") }
	return a.groupCreator.CreateGroup(ctx, name)
}

func (a *FeishuChannelAdapter) GetGroupInfo(ctx context.Context, chatID string) (any, error) {
	if a == nil || a.groupGetter == nil { return nil, errFeishuAdapterNotReady("GetGroupInfo") }
	return a.groupGetter.GetGroupInfo(ctx, chatID)
}

func (a *FeishuChannelAdapter) ListGroupMembers(ctx context.Context, chatID string, pageSize int, pageToken string) (any, error) {
	if a == nil || a.groupMemberLister == nil { return nil, errFeishuAdapterNotReady("ListGroupMembers") }
	return a.groupMemberLister.ListGroupMembers(ctx, chatID, pageSize, pageToken)
}

func (a *FeishuChannelAdapter) ListGroups(ctx context.Context, pageSize int, pageToken string) (any, error) {
	if a == nil || a.groupLister == nil { return nil, errFeishuAdapterNotReady("ListGroups") }
	return a.groupLister.ListGroups(ctx, pageSize, pageToken)
}

func (a *FeishuChannelAdapter) SendGroupMessage(ctx context.Context, chatID, text string) error {
	if a == nil || a.groupMessageSender == nil { return errFeishuAdapterNotReady("SendGroupMessage") }
	return a.groupMessageSender.SendGroupMessage(ctx, chatID, text)
}

func (a *FeishuChannelAdapter) GetDriveRootFolder(ctx context.Context) (any, error) {
	if a == nil || a.driveRootGetter == nil { return nil, errFeishuAdapterNotReady("GetDriveRootFolder") }
	return a.driveRootGetter.GetDriveRootFolder(ctx)
}

func (a *FeishuChannelAdapter) GetDriveFolder(ctx context.Context, folderToken string) (any, error) {
	if a == nil || a.driveFolderGetter == nil { return nil, errFeishuAdapterNotReady("GetDriveFolder") }
	return a.driveFolderGetter.GetDriveFolder(ctx, folderToken)
}

func (a *FeishuChannelAdapter) GetDriveFile(ctx context.Context, fileToken string) (any, error) {
	if a == nil || a.driveGetter == nil { return nil, errFeishuAdapterNotReady("GetDriveFile") }
	return a.driveGetter.GetDriveFile(ctx, fileToken)
}

func (a *FeishuChannelAdapter) ListDriveFiles(ctx context.Context, folderToken, pageToken string, pageSize int) (any, error) {
	if a == nil || a.driveLister == nil { return nil, errFeishuAdapterNotReady("ListDriveFiles") }
	return a.driveLister.ListDriveFiles(ctx, folderToken, pageToken, pageSize)
}

func (a *FeishuChannelAdapter) DownloadDriveFile(ctx context.Context, fileToken string) (any, error) {
	if a == nil || a.driveDownloader == nil { return nil, errFeishuAdapterNotReady("DownloadDriveFile") }
	return a.driveDownloader.DownloadDriveFile(ctx, fileToken)
}

func (a *FeishuChannelAdapter) DeleteDriveFile(ctx context.Context, fileToken string) error {
	if a == nil || a.driveDeleter == nil { return errFeishuAdapterNotReady("DeleteDriveFile") }
	return a.driveDeleter.DeleteDriveFile(ctx, fileToken)
}

func (a *FeishuChannelAdapter) UploadDriveFile(ctx context.Context, parentToken, name string, data []byte) (any, error) {
	if a == nil || a.driveUploader == nil { return nil, errFeishuAdapterNotReady("UploadDriveFile") }
	return a.driveUploader.UploadDriveFile(ctx, parentToken, name, data)
}

func (a *FeishuChannelAdapter) InitiateMultipartUpload(ctx context.Context, parentToken, name string, size int64) (any, error) {
	if a == nil || a.multipartInitiator == nil { return nil, errFeishuAdapterNotReady("InitiateMultipartUpload") }
	return a.multipartInitiator.InitiateMultipartUpload(ctx, parentToken, name, size)
}

func (a *FeishuChannelAdapter) UploadMultipartChunk(ctx context.Context, uploadID string, seq int, data []byte) error {
	if a == nil || a.multipartUploader == nil { return errFeishuAdapterNotReady("UploadMultipartChunk") }
	return a.multipartUploader.UploadMultipartChunk(ctx, uploadID, seq, data)
}

func (a *FeishuChannelAdapter) CompleteMultipartUpload(ctx context.Context, uploadID string, blockNum int) (any, error) {
	if a == nil || a.multipartCompleter == nil { return nil, errFeishuAdapterNotReady("CompleteMultipartUpload") }
	return a.multipartCompleter.CompleteMultipartUpload(ctx, uploadID, blockNum)
}

func (a *FeishuChannelAdapter) SendImageMessage(ctx context.Context, chatID string, data []byte, fileName string) error {
	if a == nil || a.imageSender == nil { return errFeishuAdapterNotReady("SendImageMessage") }
	return a.imageSender.SendImageMessage(ctx, chatID, data, fileName)
}

func (a *FeishuChannelAdapter) SendFileMessage(ctx context.Context, chatID string, data []byte, fileName, fileType string) error {
	if a == nil || a.fileSender == nil { return errFeishuAdapterNotReady("SendFileMessage") }
	return a.fileSender.SendFileMessage(ctx, chatID, data, fileName, fileType)
}

func (a *FeishuChannelAdapter) Ready() bool {
	return a != nil &&
		a.messageGetter != nil &&
		a.messageLister != nil &&
		a.messageReplier != nil &&
		a.shareLinkResolver != nil &&
		a.userGetter != nil &&
		a.userLister != nil &&
		a.userLookup != nil &&
		a.groupCreator != nil &&
		a.groupGetter != nil &&
		a.groupMemberLister != nil &&
		a.groupLister != nil &&
		a.groupMessageSender != nil &&
		a.driveRootGetter != nil &&
		a.driveFolderGetter != nil &&
		a.driveGetter != nil &&
		a.driveLister != nil &&
		a.driveDownloader != nil &&
		a.driveDeleter != nil &&
		a.driveUploader != nil &&
		a.multipartInitiator != nil &&
		a.multipartUploader != nil &&
		a.multipartCompleter != nil &&
		a.imageSender != nil &&
		a.fileSender != nil
}

type feishuAdapterError struct{ method string }

func (e feishuAdapterError) Error() string { return "feishu adapter not ready for method: " + e.method }

func errFeishuAdapterNotReady(method string) error { return feishuAdapterError{method: method} }
