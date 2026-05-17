package http

import (
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/offline_download/tool"
)

type SimpleHttp struct{}

func (s SimpleHttp) Name() string {
	return "SimpleHttp"
}

func (s SimpleHttp) Items() []model.SettingItem {
	return nil
}

func (s SimpleHttp) Init() (string, error) {
	delegate, err := s.delegate()
	if err != nil {
		return "", err
	}
	return delegate.Init()
}

func (s SimpleHttp) IsReady() bool {
	delegate, err := s.delegate()
	if err != nil {
		return false
	}
	return delegate.IsReady()
}

func (s SimpleHttp) AddURL(args *tool.AddUrlArgs) (string, error) {
	delegate, err := s.delegate()
	if err != nil {
		return "", err
	}
	return delegate.AddURL(args)
}

func (s SimpleHttp) Remove(task *tool.DownloadTask) error {
	delegate, err := s.delegate()
	if err != nil {
		return err
	}
	return delegate.Remove(task)
}

func (s SimpleHttp) Status(task *tool.DownloadTask) (*tool.Status, error) {
	delegate, err := s.delegate()
	if err != nil {
		return nil, err
	}
	return delegate.Status(task)
}

func (s SimpleHttp) Run(task *tool.DownloadTask) error {
	delegate, err := s.delegate()
	if err != nil {
		return err
	}
	return delegate.Run(task)
}

func (s SimpleHttp) delegate() (tool.Tool, error) {
	delegate, err := tool.Tools.Get("GuangYaPan")
	if err != nil {
		return nil, err
	}
	if delegate.Name() == s.Name() {
		return nil, errs.NotSupport
	}
	return delegate, nil
}

func init() {
	tool.Tools.Add(&SimpleHttp{})
}
