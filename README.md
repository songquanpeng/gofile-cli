# Go File 命令行工具
> 为 [Go File](https://github.com/songquanpeng/go-file) 制作的相关命令行工具

<p>
  <a href="https://raw.githubusercontent.com/songquanpeng/gofile-cli/main/LICENSE">
    <img src="https://img.shields.io/github/license/songquanpeng/gofile-cli?color=brightgreen" alt="license">
  </a>
  <a href="https://github.com/songquanpeng/gofile-cli/releases/latest">
    <img src="https://img.shields.io/github/v/release/songquanpeng/gofile-cli?color=brightgreen&include_prereleases" alt="release">
  </a>
  <a href="https://github.com/songquanpeng/gofile-cli/releases/latest">
    <img src="https://img.shields.io/github/downloads/songquanpeng/gofile-cli/total?color=brightgreen&include_prereleases" alt="release">
  </a>
</p>

可在 [Release 页面](https://github.com/songquanpeng/gofile-cli/releases/latest)下载最新版本（Windows，macOS，Linux）。


## 功能
1. [WIP] 命令行文件上传。
2. P2P 文件分享。

## 截图展示
TODO

## 使用方法
[下载](https://github.com/songquanpeng/gofile-cli/releases/latest)可执行文件后，将其放到在 PATH 环境变量里的某个目录下。

```bash
# 发送方：
gofile-cli push ./path/to/file
# 服务器会返回一个 id，如下：
# 2022/09/18 21:18:30 Server response with id: 9908585559158765387

# 接收方：
gofile-cli pull 9908585559158765387
```

## 打包流程
```bash
git clone https://github.com/songquanpeng/gofile-cli
cd gofile-cli
go mod download
go build -ldflags "-s -w -extldflags '-static'" -o gofile-cli.exe
```