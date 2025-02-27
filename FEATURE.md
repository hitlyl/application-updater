# 这是一个设备批量更新程序，功能分为两部分，IP 管理，程序更新

## 设备管理

### 使用本地文件的方式管理设备列表

### 根据输入的 ip 地址范围，自动搜索设备

#### 通过 http 方式，测试一个 url，根据是否能够连通，以及获取的内容，获取设备信息，例如 http://192.168.3.124:8089/api/buildTime

正常的设备会返回 {"code":0,"msg":"ok","result":{"buildTime":"2025-02-20_11:31:45"}}

#### 将 ip 和 buildTime 加入设备列表

### 手动添加一个设备的 ip，当添加时，通过http://192.168.3.124:8089/api/buildTime 这样的接口测试设备，如果正常则加入设备列表。

### 刷新

对每个设备调用测试连通接口，刷新 buildTime

### 设备列表可删除设备

## 程序更新
### 可以选择要更新的设备，默认是全部选择 
### 需要所有要更新设备的用户名，密码，统一用1个。

### 每个设备的登录方式

需要通过 http post 方式登录，例如 http://192.168.3.123:8089/api/login 将 {username: "admin", password: "admin"} post 输入的用户名密码信息，获取 token， 返回示例为 {"code":0,"msg":"ok","result":{"token":"1d4317b3e26040778388b26c5b724f72"}}

### 每个设备的更新方式

更新接口示例 POST http://192.168.3.123:8089/api/system/upgrade 通过 form 方式上传文件，例如 Content-Disposition: form-data; name="binary"; filename="application-web"
Content-Type: application/octet-stream

### 提供一个上传按钮，选择上传文件后，对设备列表中的每个设备，依次调用 登录，上传。并显示每个设备的操作结果
