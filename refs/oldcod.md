
// UpdateDevices updates all devices with the uploaded file
func (a *App) UpdateDevices(username, password string, selectedBuildTime string, md5FileName string) []UpdateResult {
	if UploadFilePath == "" {
		return []UpdateResult{{
			IP:      "",
			Success: false,
			Message: "No file selected for upload",
		}}
	}

	// 获取设备列表的只读副本
	a.mutex.RLock()
	allDevices := make([]Device, len(a.devices))
	copy(allDevices, a.devices)
	a.mutex.RUnlock()

	// 如果没有设备，直接返回
	if len(allDevices) == 0 {
		return []UpdateResult{{
			IP:      "",
			Success: false,
			Message: "No devices to update",
		}}
	}

	// 筛选具有指定buildTime的设备
	var devices []Device
	if selectedBuildTime == "" {
		devices = allDevices
	} else {
		devices = make([]Device, 0)
		for _, device := range allDevices {
			if device.BuildTime == selectedBuildTime {
				devices = append(devices, device)
			}
		}

		if len(devices) == 0 {
			return []UpdateResult{{
				IP:      "",
				Success: false,
				Message: "No devices with selected build time to update",
			}}
		}
	}

	// 使用结果通道收集结果
	resultChan := make(chan UpdateResult, len(devices))

	// 创建工作池控制并发
	maxConcurrent := 8 // 限制并发更新数量，避免网络和系统负载过高
	if len(devices) < maxConcurrent {
		maxConcurrent = len(devices)
	}

	// 使用通道控制并发
	deviceChan := make(chan Device, len(devices))
	for _, device := range devices {
		deviceChan <- device
	}
	close(deviceChan)

	// 启动工作线程
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for device := range deviceChan {
				// 尝试登录并更新设备
				token, err := a.LoginToDevice(device.IP, username, password)
				if err != nil {
					resultChan <- UpdateResult{
						IP:      device.IP,
						Success: false,
						Message: fmt.Sprintf("Login failed: %v", err),
					}
					continue
				}

				// 上传文件
				success, message := a.uploadBinaryWithMd5(device.IP, token, md5FileName)
				resultChan <- UpdateResult{
					IP:      device.IP,
					Success: success,
					Message: message,
				}
			}
		}()
	}

	// 等待所有更新完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	results := make([]UpdateResult, 0, len(devices))
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

// uploadBinaryWithMd5 uploads a binary file to a device with optional MD5 file
func (a *App) uploadBinaryWithMd5(ip, token, md5FileName string) (bool, string) {
	// 获取上传文件路径
	filePath := a.GetUploadFile()
	fmt.Printf("DEBUG: 开始上传文件: %s 到设备 %s\n", filePath, ip)

	// 检查文件是否存在
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("ERROR: 文件不存在或无法访问: %s, 错误: %v\n", filePath, err)
		return false, fmt.Sprintf("File not found or inaccessible: %v", err)
	}

	fmt.Printf("DEBUG: 文件大小: %d 字节\n", fileInfo.Size())

	// 检查是否提供了MD5文件
	var md5FilePath string
	if md5FileName != "" {
		md5FilePath = a.GetMd5File()
		fmt.Printf("DEBUG: 将包含MD5文件: %s\n", md5FilePath)
		
		// 检查MD5文件是否存在
		if _, err := os.Stat(md5FilePath); err != nil {
			fmt.Printf("WARNING: MD5文件不存在或无法访问: %s, 错误: %v\n", md5FilePath, err)
			// 继续而不使用MD5文件
			md5FilePath = ""
		}
	}

	// 设置超时（基于文件大小，每MB至少10秒，最少30秒）
	fileSize := fileInfo.Size()
	timeoutSeconds := int(math.Max(30, float64(fileSize)/(1024*1024)*10))
	fmt.Printf("DEBUG: 设置上传超时为 %d 秒\n", timeoutSeconds)

	// 创建上传URL
	url := fmt.Sprintf("http://%s:8089/api/system/upgrade", ip)
	fmt.Printf("DEBUG: 上传URL: %s\n", url)

	// 创建一个缓冲区来存储multipart数据
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("ERROR: 无法打开文件: %s, 错误: %v\n", filePath, err)
		return false, fmt.Sprintf("Cannot open file: %v", err)
	}
	defer file.Close()

	fmt.Printf("DEBUG: 创建multipart表单字段 'binary'\n")
	part, err := writer.CreateFormFile("binary", filepath.Base(filePath))
	if err != nil {
		fmt.Printf("ERROR: 创建表单文件字段失败: %v\n", err)
		return false, fmt.Sprintf("Failed to create form file: %v", err)
	}

	// 复制文件内容到表单字段
	bytesWritten, err := io.Copy(part, file)
	if err != nil {
		fmt.Printf("ERROR: 复制文件内容失败: %v\n", err)
		return false, fmt.Sprintf("Failed to copy file content: %v", err)
	}
	fmt.Printf("DEBUG: 已写入 %d 字节到表单\n", bytesWritten)

	// 如果指定了MD5文件，也加入到请求中
	if md5FilePath != "" {
		md5File, err := os.Open(md5FilePath)
		if err != nil {
			fmt.Printf("ERROR: 无法打开MD5文件: %s, 错误: %v\n", md5FilePath, err)
			// 继续而不使用MD5文件
		} else {
			defer md5File.Close()
			
			fmt.Printf("DEBUG: 创建multipart表单字段 'md5file'\n")
			md5Part, err := writer.CreateFormFile("md5file", filepath.Base(md5FilePath))
			if err != nil {
				fmt.Printf("ERROR: 创建MD5表单文件字段失败: %v\n", err)
				// 继续而不使用MD5文件
			} else {
				// 复制MD5文件内容到表单字段
				md5BytesWritten, err := io.Copy(md5Part, md5File)
				if err != nil {
					fmt.Printf("ERROR: 复制MD5文件内容失败: %v\n", err)
					// 继续而不使用MD5文件
				} else {
					fmt.Printf("DEBUG: 已写入 %d 字节MD5数据到表单\n", md5BytesWritten)
				}
			}
		}
	}

	// 完成multipart表单
	err = writer.Close()
	if err != nil {
		fmt.Printf("ERROR: 关闭multipart writer失败: %v\n", err)
		return false, fmt.Sprintf("Failed to close multipart writer: %v", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		fmt.Printf("ERROR: 创建HTTP请求失败: %v\n", err)
		return false, fmt.Sprintf("Failed to create request: %v", err)
	}

	// 设置Content-Type头，包含动态生成的boundary
	contentType := writer.FormDataContentType()
	fmt.Printf("DEBUG: Content-Type: %s\n", contentType)
	req.Header.Set("Content-Type", contentType)

	// 使用Token头部
	req.Header.Set("Token", token)
	fmt.Printf("DEBUG: Token: %s\n", token)

	// 打印完整请求头
	fmt.Printf("DEBUG: 请求头:\n")
	for key, values := range req.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	// 设置一个专用于上传的客户端，具有更长的超时时间
	uploadClient := &http.Client{
		Transport: a.client.Transport,
		Timeout:   time.Duration(timeoutSeconds) * time.Second,
	}

	fmt.Printf("DEBUG: 发送HTTP请求...\n")
	resp, err := uploadClient.Do(req)
	if err != nil {
		fmt.Printf("ERROR: 发送请求失败: %v\n", err)
		return false, fmt.Sprintf("Upload failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: 收到响应状态码: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("DEBUG: 响应头:\n")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	if resp.StatusCode != http.StatusOK {
		// 读取错误响应以获取更详细的信息
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		fmt.Printf("ERROR: 上传失败. 响应体: %s\n", string(errBody))
		return false, fmt.Sprintf("Upload failed with status: %s, details: %s", resp.Status, string(errBody))
	}

	// 读取成功响应
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	fmt.Printf("DEBUG: 上传成功. 响应体: %s\n", string(respBody))

	// 解析JSON响应体，检查code字段
	var responseObj struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.Unmarshal(respBody, &responseObj); err != nil {
		fmt.Printf("ERROR: 无法解析响应JSON: %v\n", err)
		return false, fmt.Sprintf("Failed to parse response: %v", err)
	}

	// 检查code字段
	if responseObj.Code != 0 {
		fmt.Printf("ERROR: 上传失败. 错误码: %d, 错误信息: %s\n", responseObj.Code, responseObj.Msg)
		return false, fmt.Sprintf("Upload failed: %s", responseObj.Msg)
	}

	return true, "Update successful"
}