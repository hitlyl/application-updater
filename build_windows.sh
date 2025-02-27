# 设置编译环境变量
export CGO_ENABLED=1
export GOOS=windows
export GOARCH=amd64
export CC=x86_64-w64-mingw32-gcc
export CXX=x86_64-w64-mingw32-g++

# 获取版本信息
VERSION=$(git describe --tags 2>/dev/null || echo "dev")
BUILD_TIME=$(date +"%Y%m%d%H%M%S")
OUTPUT_NAME="application-updater-${VERSION}-${BUILD_TIME}.exe"

echo "======================================================="
echo "开始构建Windows应用程序 - application-updater"
echo "======================================================="
echo "构建环境:"
echo "GOOS: $GOOS"
echo "GOARCH: $GOARCH"
echo "CGO_ENABLED: $CGO_ENABLED"
echo "构建版本: $VERSION"
echo "构建时间: $(date +"%Y-%m-%d %H:%M:%S")"
echo "======================================================="

# 确保依赖已安装
echo "正在更新依赖..."
go mod tidy

# 在构建前添加架构切换逻辑
export GOOS_backup=$GOOS
export GOARCH_backup=$GOARCH

# 临时切换回宿主机构建绑定
echo "正在生成绑定（使用宿主机架构）..."
(
    unset CC CXX  # 关键修复：清除交叉编译工具链
    export GOOS=darwin
    export GOARCH=$(go env GOARCH)
    export CGO_ENABLED=1  # 确保本地CGO可用
    wails generate bindings
)
if [ $? -ne 0 ]; then
    echo "绑定生成失败"
    exit 1
fi

# 恢复交叉编译设置
export GOOS=$GOOS_backup
export GOARCH=$GOARCH_backup

# 修改构建命令
echo "开始构建Windows应用程序..."
wails build -platform windows -clean -ldflags "-s -w" -trimpath

# 检查构建结果
if [ $? -eq 0 ]; then
    echo "======================================================="
    echo "构建成功!"
    echo "Windows可执行文件位于: build/bin/application-updater.exe"
    echo "======================================================="
else
    echo "======================================================="
    echo "构建失败，请检查错误信息"
    echo "======================================================="
    exit 1
fi

# 创建发布目录
RELEASE_DIR="release"
mkdir -p "$RELEASE_DIR"

# 复制并重命名构建产物
echo "复制可执行文件到发布目录..."
cp "build/bin/application-updater.exe" "$RELEASE_DIR/$OUTPUT_NAME"

# 生成校验文件
pushd "$RELEASE_DIR" > /dev/null
sha256sum "$OUTPUT_NAME" > "$OUTPUT_NAME.sha256"
popd > /dev/null

echo "构建产物已保存到：$RELEASE_DIR/"
echo "文件列表："
ls -lh "$RELEASE_DIR"/