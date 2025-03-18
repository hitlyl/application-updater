# 这是一个设备批量更新程序，功能分为三部分，设备管理，程序更新，批量操作

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


#批量配置摄像头
## 上传excel文件
### excel文件有多个sheet，每个sheet 使用tab方式显示，每个tab按表格显示该sheet内容。该sheet的格式为 使用后三列，倒数第三列为设备ip（此列用到了合并单元格，需要处理，即如果为空，则为上一列的数据）,倒数第二列为摄像头名称，倒数第一列为摄像头的ip/掩码/网关。 过滤掉数据为 "/" 字符串的内容
###每个设备下的每个摄像头自动增加个index，从1开始。并且每个设备单独统计。

### 选择一个tab后，下方需要配置 设备的用户名，密码，摄像头url模版（例如 rtsp://admin:123@<ip>/av/stream)
算法选择单选框 (6:精准喷淋,7:牛行为统计)
### 点击配置按钮后，根据excel表格，每个设备有1个或多个摄像头， 根据设备的ip，用户名，密码，先调用login获取token, 再调用 http POST 
header包括获取的 Token
http://192.168.3.123:8089/api/task/list
载荷 {"pageNo":1,"pageSize":100}
获取当前已有的摄像头配置。
返回数据示例为
{
    "code": 0,
    "msg": "ok",
    "result": {
        "total": 1,
        "pageSize": 100,
        "pageCount": 1,
        "pageNo": 1,
        "items": [
            {
                "taskId": "test1",
                "deviceName": "test1",
                "url": "rtsp://192.168.3.164:30554/record/cam_video/cow5.mp4",
                "status": 0,
                "errorReason": "",
                "abilities": [
                    "区域入侵检测(yolox)"
                ],
                "types": [
                    5
                ],
                "width": 1280,
                "height": 736,
                "codeName": "h264"
            }]
    }
}

taskId为该设备下该摄像头唯一标识，为摄像头的名称。
如果该摄像头没有配置，则调用添加接口
POST
http://192.168.3.123:8089/api/task/add
载荷:
{"taskId":"摄像头名称","deviceName":"摄像头名称","url":"摄像头url","types":[算法id]}

如果该摄像头已经配置，则调用修改接口
POST
http://192.168.3.123:8089/api/task/modify
载荷:
{"taskId":"摄像头名称","deviceName":"摄像头名称","url":"摄像头url","types":[算法id]}

## 自动设置摄像头index
### 先通过接口 POST http://192.168.3.123:8089/api/config/get 
载荷为 {"taskId":"5号牛舍-南1"}
获取这个摄像头的任务信息
例如 
{
    "code": 0,
    "msg": "ok",
    "result": {
        "device": {
            "codeName": "h264",
            "name": "5号牛舍-南1",
            "resolution": "1280*736",
            "url": "rtsp://192.168.3.164:30554/record/cam_video/cow5.mp4",
            "width": 1280,
            "height": 736
        },
        "algorithms": [
            {
                "Type": 7,
                "TrackInterval": 3,
                "DetectInterval": 3,
                "AlarmInterval": 1,
                "threshold": 50,
                "TargetSize": {
                    "MinDetect": 30,
                    "MaxDetect": 250
                },
                "DetectInfos": [
                    {
                        "Id": 1,
                        "HotArea": [
                            {
                                "X": 0,
                                "Y": 0
                            },
                            {
                                "X": 1280,
                                "Y": 0
                            },
                            {
                                "X": 1280,
                                "Y": 736
                            },
                            {
                                "X": 0,
                                "Y": 736
                            }
                        ]
                    }
                ],
                "TripWire": {
                    "LineStart": {
                        "X": 0,
                        "Y": 0
                    },
                    "LineEnd": {
                        "X": 0,
                        "Y": 0
                    },
                    "DirectStart": {
                        "X": 0,
                        "Y": 0
                    },
                    "DirectEnd": {
                        "X": 0,
                        "Y": 0
                    }
                },
                "ExtraConfig": {
                    "camera_index": "2",
                    "defs": [
                        {
                            "Name": "interval",
                            "Desc": "抑制时间",
                            "Type": "string",
                            "Unit": "*s/*m/*h",
                            "Default": "1s"
                        },
                        {
                            "Name": "save_picture",
                            "Desc": "保存图片",
                            "Type": "int",
                            "Unit": "0/1",
                            "Default": "0"
                        },
                        {
                            "Name": "camera_index",
                            "Desc": "摄像头索引",
                            "Type": "int",
                            "Unit": "",
                            "Default": "1"
                        }
                    ]
                }
            }
        ]
    }
}

其中result为完整的配置信息， 修改 ExtraConfig的 camera_index 为实际的摄像头index，然后调用
POST   http://192.168.3.123:8089/api/config/mod
载荷示例 
{"TaskID":"1号牛舍-南1","Algorithm":{"Type":6,"TrackInterval":3,"DetectInterval":3,"AlarmInterval":1,"TargetSize":{"MinDetect":30,"MaxDetect":250},"threshold":50,"DetectInfos":[{"Id":1,"ExtraConfig":{},"HotArea":[{"X":0,"Y":0},{"X":1920,"Y":0},{"X":1920,"Y":1080},{"X":0,"Y":1080}]}],"DetectPoints":[],"TripWire":{"LineStart":{"X":0,"Y":0},"LineEnd":{"X":0,"Y":0},"DirectStart":{"X":0,"Y":0},"DirectEnd":{"X":0,"Y":0}},"ExtraConfig":{"defs":[{"Name":"interval","Desc":"抑制时间","Type":"string","Unit":"*s/*m/*h","Default":"2s"},{"Name":"save_picture","Desc":"保存图片","Type":"int","Unit":"0/1","Default":"0"},{"Name":"camera_index","Desc":"摄像头索引(从1开始)","Type":"int","Unit":"","Default":"1"},{"Name":"precision","Desc":"精度","Type":"string","Unit":"","Default":"fp32"}],"camera_index":"2"}}}

进行修改。