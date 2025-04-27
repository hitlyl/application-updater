# 应用重构计划

## 当前项目结构

当前项目主要是一个大型的 Go 应用程序，所有的核心逻辑几乎都在一个单一的`app.go`文件中(3054 行)，这使得代码难以维护和理解。项目使用了 Wails 框架构建跨平台桌面应用程序，主要功能包括：设备管理、文件上传、摄像头配置、时间同步、备份和恢复等功能。

## 重构目标

1. 将单一的`app.go`文件拆分为多个模块化的包和文件
2. 改善代码组织结构，提高可读性和可维护性
3. 遵循 Go 的最佳实践
4. 保持功能的完整性和兼容性

## 重构步骤

### 步骤 1：创建必要的目录结构

```
application-updater/
├── cmd/
│   └── app/
│       └── main.go          # 入口点
├── internal/
│   ├── app/                 # 应用程序核心
│   │   ├── app.go           # 简化后的App结构和主要方法
│   │   ├── setup.go         # 应用程序初始化
│   │   └── cleanup.go       # 清理方法
│   ├── models/              # 数据模型
│   │   ├── device.go        # 设备相关模型
│   │   ├── camera.go        # 摄像头相关模型
│   │   ├── backup.go        # 备份相关模型
│   │   ├── excel.go         # Excel数据相关模型
│   │   └── sync.go          # 同步相关模型
│   ├── services/            # 业务逻辑服务
│   │   ├── device/          # 设备管理服务
│   │   │   ├── manager.go   # 设备管理
│   │   │   ├── scanner.go   # 设备扫描
│   │   │   └── auth.go      # 设备认证
│   │   ├── camera/          # 摄像头服务
│   │   │   ├── config.go    # 摄像头配置
│   │   │   └── tasks.go     # 摄像头任务
│   │   ├── backup/          # 备份服务
│   │   │   ├── backup.go    # 备份功能
│   │   │   └── restore.go   # 恢复功能
│   │   ├── time/
│   │   │   └── sync.go      # 时间同步
│   │   └── excel/
│   │       └── parser.go    # Excel解析
│   └── utils/               # 通用工具
│       ├── http.go          # HTTP工具
│       ├── file.go          # 文件操作工具
│       ├── ssh.go           # SSH工具
│       └── dialog.go        # 对话框工具
├── pkg/                     # 可能被外部使用的包
│   └── logger/              # 日志包
│       └── logger.go
└── go.mod
```

### 步骤 2：定义和规划数据模型

1. 将`app.go`中的所有类型定义移动到`internal/models/`目录下的相应文件中
   - `device.go`: Device, TimeSyncResult 等设备相关结构
   - `camera.go`: Camera, CameraConfig, Algorithm 等摄像头相关结构
   - `backup.go`: BackupResult, RestoreResult, BackupSettings 等备份相关结构
   - `excel.go`: ExcelSheetData, ExcelRow 等 Excel 相关结构

### 步骤 3：分离核心业务逻辑

1. 将设备管理相关函数移动到`internal/services/device/`

   - `manager.go`：GetDevices, SaveDevices, LoadDevices, AddDevice, RemoveDevice 等
   - `scanner.go`：ScanIPRange, RefreshDevices 等
   - `auth.go`：LoginToDevice 等

2. 将摄像头相关函数移动到`internal/services/camera/`

   - `config.go`：ConfigureCamera, GetCameraConfig, SetCameraIndex 等
   - `tasks.go`：GetCameraTasks 等

3. 将备份相关函数移动到`internal/services/backup/`

   - `backup.go`：BackupDevices, backupSingleDevice, SaveBackupSettings, GetBackupSettings 等
   - `restore.go`：RestoreDeviceDB, RestoreDevicesDB 等

4. 将时间同步相关函数移动到`internal/services/time/`

   - `sync.go`：SyncDeviceTime, syncSingleDeviceTime 等

5. 将 Excel 处理相关函数(从 excel.go)移动到`internal/services/excel/`
   - `parser.go`：ParseExcelSheet, ProcessExcelData 等

### 步骤 4：提取通用工具函数

1. 将 HTTP 相关工具函数移动到`internal/utils/http.go`

   - createOptimizedTransport 等

2. 将文件操作相关函数移动到`internal/utils/file.go`

   - copyFile, SetUploadFile, GetUploadFile 等

3. 将 SSH 相关函数移动到`internal/utils/ssh.go`

   - executeSSHCommand, scpFileToRemote 等

4. 将对话框相关函数移动到`internal/utils/dialog.go`
   - SelectFolder 等

### 步骤 5：简化 App 结构

1. 在`internal/app/app.go`中保留简化后的 App 结构，包含：

   - 必要的字段
   - 对各个 service 的引用
   - 面向前端 UI 的方法委托给相应的 service

2. 在`internal/app/setup.go`中保留应用程序初始化逻辑

   - startup 方法
   - 服务初始化

3. 在`internal/app/cleanup.go`中保留清理逻辑
   - cleanUploadsDirectory
   - cleanDirectory

### 步骤 6：更新入口点

1. 修改`cmd/app/main.go`以使用新的模块化结构

### 步骤 7：添加单元测试

1. 为每个模块添加单元测试
2. 确保所有功能正常工作

### 步骤 8：更新文档

1. 更新 README.md 以反映新的项目结构
2. 为每个主要模块添加文档

## Wails 项目特殊处理事项

作为一个基于 Wails 框架的项目，在重构过程中需要注意以下特殊处理：

1. **保持绑定结构完整性**：

   - Wails 通过`Bind`选项将 Go 结构体及其方法暴露给前端 JavaScript/TypeScript
   - 确保`App`结构体仍是主要绑定点，或适当修改绑定逻辑
   - 公开方法的签名必须保持一致，以维持前端 API 兼容性

2. **前端集成注意事项**：

   - 保留`//go:embed all:frontend/dist`指令，确保前端资源正确嵌入
   - 重构后测试所有前端 API 调用，确保它们仍能正常工作
   - 如果前端有直接依赖于 App 结构的 TypeScript 类型定义，需要同步更新

3. **Wails 配置文件**：

   - 检查并更新`wails.json`中的路径引用
   - 确保`wails.json`中的`main`字段指向正确的入口点

4. **入口点调整**：

   - 确保`cmd/app/main.go`中正确引用重构后的 App 结构
   - 在`wails.Run()`中保持相同的配置和绑定逻辑

5. **开发模式兼容性**：

   - 确保`wails dev`命令在重构后仍能正常工作
   - 测试热重载功能

6. **构建流程**：

   - 验证`wails build`命令在重构后能正确构建应用
   - 确保任何特定于平台的配置或资源仍能正确包含

7. **实现委托模式**：

   - 在`internal/app/app.go`中实现委托模式，保持对外 API 不变
   - 示例：
     ```go
     // App结构体保持对外API不变，但内部委托给专门的服务
     func (a *App) AddDevice(ip string) (Device, error) {
         // 委托给设备管理服务
         return a.deviceService.AddDevice(ip)
     }
     ```

8. **避免循环导入**：
   - 在分离服务和模型时，特别注意避免循环导入问题
   - 考虑使用接口而非直接依赖具体类型

## 重构注意事项

1. **保持向后兼容性**：确保重构不会破坏现有的功能
2. **分阶段进行**：考虑一次重构一个模块，而不是一次性重构所有内容
3. **添加日志**：在关键位置添加日志，以便更容易地追踪问题
4. **保持清晰的依赖关系**：避免循环依赖
5. **常规测试**：每完成一个模块就进行测试，确保功能正常

## 下一步计划

完成重构后，考虑以下改进：

1. 添加配置文件支持，替代硬编码的设置
2. 改进错误处理机制
3. 添加更全面的日志系统
4. 考虑添加依赖注入以简化测试
5. 优化并发处理以提高性能
