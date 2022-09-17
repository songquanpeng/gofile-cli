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
2. [WIP] P2P 文件分享。

## 截图展示
TODO

## 使用方法
### Windows 用户
直接双击 gofile-cli.exe 运行。

### macOS 用户
1. 给执行权限：`chmod u+x gofile-cli-macos`；
2. 之后直接双击运行 gofile-cli-macos 或在终端中运行都可。

### Linux 用户
同上，区别在于文件名换成 `gofile-cli`。

## 打包流程
```bash
go mod download
go build -ldflags "-s -w -extldflags '-static'" -o gofile-cli.exe
```