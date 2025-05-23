# MyFileBrowser

基于Go语言实现的轻量级Web文件浏览器，支持媒体文件预览和响应式布局。

## 功能特性
- 🖥️ 基于Web的文件目录浏览
- 🖼️ 图片/视频/音频文件即时预览
- 📂 面包屑导航路径追踪
- 🌓 自动暗黑模式支持
- 📱 响应式网格布局（支持手机端）
- 🔒 安全路径访问控制

## 安装步骤

### 前置要求
- Go 1.24+ 
- rice工具（go install github.com/GeertJohan/go.rice/rice@latest）

```bash
# 克隆仓库
git clone https://github.com/hailangjj/myfilebrowser.git
cd myfilebrowser

# 安装依赖
go mod download

# 编译项目
rice embed-go
go build -o myfilebrowser
```

## 使用说明
### 启动服务
```bash
./myfilebrowser \
  -dir "./srv" \      # 指定服务目录（默认./srv）
  -addr "0.0.0.0" \   # 监听地址
  -port "8080"        # 监听端口
```

### 访问文件浏览器
打开浏览器，访问 http://localhost:8080 即可访问文件浏览器。

## 注意事项
- 确保服务目录存在且可读。
- 确保浏览器和服务端在同一网络环境下。
- 支持的媒体文件类型：图片（jpg, jpeg, png, gif, webp）、视频（mp4, webm）、音频（mp3, wav, ogg）。

## 贡献
欢迎提交PR和Issue。

## 许可证
MIT