package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type stubInstanceDAO struct {
	instance *model.WorkorderInstance
	err      error
}

func (s *stubInstanceDAO) CreateInstance(context.Context, *model.WorkorderInstance) error {
	return nil
}
func (s *stubInstanceDAO) UpdateInstance(context.Context, *model.WorkorderInstance) error {
	return nil
}
func (s *stubInstanceDAO) DeleteInstance(context.Context, int) error { return nil }
func (s *stubInstanceDAO) GetInstanceByID(context.Context, int) (*model.WorkorderInstance, error) {
	return s.instance, s.err
}
func (s *stubInstanceDAO) GetInstanceByTitle(context.Context, string) (*model.WorkorderInstance, error) {
	return nil, nil
}
func (s *stubInstanceDAO) ListInstance(context.Context, *model.ListWorkorderInstanceReq) ([]*model.WorkorderInstance, int64, error) {
	return nil, 0, nil
}
func (s *stubInstanceDAO) GenerateSerialNumber(context.Context) (string, error) { return "", nil }
func (s *stubInstanceDAO) UpdateInstanceStatus(context.Context, int, int8) error { return nil }
func (s *stubInstanceDAO) UpdateInstanceAssignee(context.Context, int, *int) error {
	return nil
}

type stubCommentDAO struct {
	created *model.WorkorderInstanceComment
}

func (s *stubCommentDAO) CreateInstanceComment(_ context.Context, comment *model.WorkorderInstanceComment) error {
	comment.ID = 101
	s.created = comment
	return nil
}
func (s *stubCommentDAO) UpdateInstanceComment(context.Context, *model.WorkorderInstanceComment) error {
	return nil
}
func (s *stubCommentDAO) DeleteInstanceComment(context.Context, int) error { return nil }
func (s *stubCommentDAO) GetInstanceCommentByID(context.Context, int) (*model.WorkorderInstanceComment, error) {
	return nil, fmt.Errorf("not found")
}
func (s *stubCommentDAO) GetInstanceComments(context.Context, int) ([]*model.WorkorderInstanceComment, error) {
	return nil, nil
}
func (s *stubCommentDAO) GetInstanceCommentsTree(context.Context, int) ([]*model.WorkorderInstanceComment, error) {
	return nil, nil
}
func (s *stubCommentDAO) ListInstanceComments(context.Context, *model.ListWorkorderInstanceCommentReq) ([]*model.WorkorderInstanceComment, int64, error) {
	return nil, 0, nil
}

type stubAttachmentDAO struct {
	unboundCount   int64
	bindErr        error
	lastBindIDs    []int
	lastBindInstID int
	createCalled   bool
}

func (s *stubAttachmentDAO) CreateAttachment(_ context.Context, attachment *model.WorkorderInstanceCommentAttachment) error {
	s.createCalled = true
	attachment.ID = 1
	return nil
}
func (s *stubAttachmentDAO) GetAttachmentByID(context.Context, int) (*model.WorkorderInstanceCommentAttachment, error) {
	return nil, fmt.Errorf("not found")
}
func (s *stubAttachmentDAO) BindAttachmentsToComment(_ context.Context, _, instanceID, _ int, ids []int) error {
	s.lastBindInstID = instanceID
	s.lastBindIDs = append([]int(nil), ids...)
	return s.bindErr
}
func (s *stubAttachmentDAO) DeleteUnboundAttachment(context.Context, int, int) (*model.WorkorderInstanceCommentAttachment, error) {
	return nil, nil
}
func (s *stubAttachmentDAO) CountUnboundByOperator(context.Context, int, int) (int64, error) {
	return s.unboundCount, nil
}

func newCommentServiceForTest(att *stubAttachmentDAO, comment *stubCommentDAO) *instanceCommentService {
	return &instanceCommentService{
		dao:           comment,
		attachmentDAO: att,
		instanceDao: &stubInstanceDAO{
			instance: &model.WorkorderInstance{Model: model.Model{ID: 7}},
		},
		logger: zap.NewNop(),
	}
}

func TestCreateInstanceCommentRequiresContentOrAttachment(t *testing.T) {
	svc := newCommentServiceForTest(&stubAttachmentDAO{}, &stubCommentDAO{})
	err := svc.CreateInstanceComment(context.Background(), &model.CreateWorkorderInstanceCommentReq{
		InstanceID: 7,
		OperatorID: 1,
		Content:    "   ",
	})
	if err == nil || !strings.Contains(err.Error(), "至少填写一项") {
		t.Fatalf("expected content-or-attachment error, got %v", err)
	}
}

func TestCreateInstanceCommentAttachmentOnly(t *testing.T) {
	att := &stubAttachmentDAO{}
	commentDAO := &stubCommentDAO{}
	svc := newCommentServiceForTest(att, commentDAO)

	err := svc.CreateInstanceComment(context.Background(), &model.CreateWorkorderInstanceCommentReq{
		InstanceID:    7,
		OperatorID:    3,
		OperatorName:  "tester",
		AttachmentIDs: []int{11, 12},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if commentDAO.created == nil || commentDAO.created.Content != "" {
		t.Fatalf("expected empty content comment, got %#v", commentDAO.created)
	}
	if len(att.lastBindIDs) != 2 || att.lastBindInstID != 7 {
		t.Fatalf("unexpected bind call: ids=%v instance=%d", att.lastBindIDs, att.lastBindInstID)
	}
}

func TestCreateInstanceCommentBindWrongInstanceFails(t *testing.T) {
	att := &stubAttachmentDAO{bindErr: fmt.Errorf("存在无效或已绑定的附件")}
	svc := newCommentServiceForTest(att, &stubCommentDAO{})

	err := svc.CreateInstanceComment(context.Background(), &model.CreateWorkorderInstanceCommentReq{
		InstanceID:    7,
		OperatorID:    3,
		AttachmentIDs: []int{99},
	})
	if err == nil || !strings.Contains(err.Error(), "绑定评论附件失败") {
		t.Fatalf("expected bind failure, got %v", err)
	}
}

func TestUploadCommentAttachmentRejectsType(t *testing.T) {
	dir := t.TempDir()
	viper.Set("workorder.attachment_dir", dir)
	viper.Set("workorder.attachment_max_size_mb", 10)
	viper.Set("workorder.attachment_max_count", 5)
	defer viper.Reset()

	att := &stubAttachmentDAO{}
	svc := newCommentServiceForTest(att, &stubCommentDAO{})
	header := multipartHeader("evil.exe", "application/octet-stream", []byte("MZ"))

	_, err := svc.UploadCommentAttachment(context.Background(), 7, 3, header)
	if err == nil || !strings.Contains(err.Error(), "不支持的文件类型") {
		t.Fatalf("expected type rejection, got %v", err)
	}
	if att.createCalled {
		t.Fatal("should not persist rejected file")
	}
}

func TestUploadCommentAttachmentRejectsSize(t *testing.T) {
	dir := t.TempDir()
	viper.Set("workorder.attachment_dir", dir)
	viper.Set("workorder.attachment_max_size_mb", 1)
	viper.Set("workorder.attachment_max_count", 5)
	defer viper.Reset()

	att := &stubAttachmentDAO{}
	svc := newCommentServiceForTest(att, &stubCommentDAO{})
	payload := bytes.Repeat([]byte("a"), 2*1024*1024)
	header := multipartHeader("big.png", "image/png", payload)

	_, err := svc.UploadCommentAttachment(context.Background(), 7, 3, header)
	if err == nil || !strings.Contains(err.Error(), "文件大小不能超过") {
		t.Fatalf("expected size rejection, got %v", err)
	}
	if att.createCalled {
		t.Fatal("should not persist oversized file")
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "7"))
	if len(entries) > 0 {
		t.Fatalf("expected no leftover files, got %d", len(entries))
	}
}

func multipartHeader(filename, contentType string, body []byte) *multipart.FileHeader {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	h.Set("Content-Type", contentType)
	part, err := w.CreatePart(h)
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(part, bytes.NewReader(body)); err != nil {
		panic(err)
	}
	_ = w.Close()

	r := multipart.NewReader(&buf, w.Boundary())
	form, err := r.ReadForm(32 << 20)
	if err != nil {
		panic(err)
	}
	files := form.File["file"]
	if len(files) == 0 {
		panic("no file part")
	}
	return files[0]
}
