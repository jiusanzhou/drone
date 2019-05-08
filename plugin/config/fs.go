package config

import (
	"context"
	"path/filepath"
	"io/ioutil"

	"github.com/drone/drone/core"
)

type fs struct {
	root string
}

// LocalStorage ...
func LocalStorage(root string) core.ConfigService {
	return &fs{root}
}

func(f *fs) Find(ctx context.Context, in *core.ConfigArgs) (*core.Config, error) {

	// 从文件系统读取 yaml 文件

	fpath := filepath.Join(f.root, in.Repo.Config)

	data, err := ioutil.ReadFile(fpath)

	if err != nil {
		// TODO: 找不到
		// 我们可以尝试去其他地方找
		// 所以把文件路径去掉吗?
		return nil, nil
	}
	
	return &core.Config{
		Kind: "",
		Data: string(data),
	}, nil
}