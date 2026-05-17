package tool

import (
	"context"
	"path/filepath"
	"strings"

	_115 "github.com/alist-org/alist/v3/drivers/115"
	"github.com/alist-org/alist/v3/drivers/pikpak"
	"github.com/alist-org/alist/v3/drivers/thunder"
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/internal/setting"
	"github.com/alist-org/alist/v3/internal/task"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type DeletePolicy string

const (
	DeleteOnUploadSucceed DeletePolicy = "delete_on_upload_succeed"
	DeleteOnUploadFailed  DeletePolicy = "delete_on_upload_failed"
	DeleteNever           DeletePolicy = "delete_never"
	DeleteAlways          DeletePolicy = "delete_always"
)

type AddURLArgs struct {
	URL          string
	DstDirPath   string
	Tool         string
	DeletePolicy DeletePolicy
}

func NormalizeToolName(toolName string) string {
	if toolName == "SimpleHttp" {
		// Preserve the legacy entry name while routing requests through
		// GuangYaPan's offline-download implementation.
		return "GuangYaPan"
	}
	return toolName
}

func ResolveDstDirPath(toolName, requestedPath string) string {
	if NormalizeToolName(toolName) != "GuangYaPan" {
		return requestedPath
	}
	tempDir := strings.TrimSpace(setting.GetStr(conf.GuangYaPanTempDir))
	if tempDir == "" {
		return requestedPath
	}
	return tempDir
}

func AddURL(ctx context.Context, args *AddURLArgs) (task.TaskExtensionInfo, error) {
	actualToolName := NormalizeToolName(args.Tool)
	dstDirPath := ResolveDstDirPath(args.Tool, args.DstDirPath)

	// check storage
	storage, dstDirActualPath, err := op.GetStorageAndActualPath(dstDirPath)
	if err != nil {
		return nil, errors.WithMessage(err, "failed get storage")
	}
	// check is it could upload
	if storage.Config().NoUpload {
		return nil, errors.WithStack(errs.UploadNotSupported)
	}
	// check path is valid
	obj, err := op.Get(ctx, storage, dstDirActualPath)
	if err != nil {
		if !errs.IsObjectNotFound(err) {
			return nil, errors.WithMessage(err, "failed get object")
		}
	} else {
		if !obj.IsDir() {
			// can't add to a file
			return nil, errors.WithStack(errs.NotFolder)
		}
	}

	// get tool
	tool, err := Tools.Get(actualToolName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed get tool")
	}
	// check tool is ready
	if !tool.IsReady() {
		// try to init tool
		if _, err := tool.Init(); err != nil {
			return nil, errors.Wrapf(err, "failed init tool %s", actualToolName)
		}
	}

	uid := uuid.NewString()
	tempDir := filepath.Join(conf.Conf.TempDir, actualToolName, uid)
	deletePolicy := args.DeletePolicy

	// If destination storage matches the offline tool's native storage,
	// download directly into the destination path to avoid an extra transfer.
	switch actualToolName {
	case "115 Cloud":
		if _, ok := storage.(*_115.Pan115); ok {
			tempDir = dstDirPath
		} else {
			tempDir = filepath.Join(setting.GetStr(conf.Pan115TempDir), uid)
		}
	case "PikPak":
		if _, ok := storage.(*pikpak.PikPak); ok {
			tempDir = dstDirPath
		} else {
			tempDir = filepath.Join(setting.GetStr(conf.PikPakTempDir), uid)
		}
	case "Thunder":
		if _, ok := storage.(*thunder.Thunder); ok {
			tempDir = dstDirPath
		} else {
			tempDir = filepath.Join(setting.GetStr(conf.ThunderTempDir), uid)
		}
	case "GuangYaPan":
		tempBase := strings.TrimSpace(setting.GetStr(conf.GuangYaPanTempDir))
		if tempBase == "" {
			return nil, errors.New("GuangYaPan temp dir is not set")
		}
		tempDir = tempBase
	}

	taskCreator, _ := ctx.Value("user").(*model.User) // taskCreator is nil when convert failed
	t := &DownloadTask{
		TaskExtension: task.TaskExtension{
			Creator: taskCreator,
		},
		Url:          args.URL,
		DstDirPath:   dstDirPath,
		TempDir:      tempDir,
		DeletePolicy: deletePolicy,
		Toolname:     actualToolName,
		tool:         tool,
	}
	DownloadTaskManager.Add(t)
	return t, nil
}
