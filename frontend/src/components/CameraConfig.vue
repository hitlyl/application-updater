<template>
  <div class="camera-config-container">
    <h2>批量配置摄像头</h2>

    <div class="upload-section">
      <h3>上传Excel文件</h3>
      <div class="file-upload">
        <label for="fileInput" class="file-label">选择Excel文件</label>
        <input
          type="file"
          id="fileInput"
          class="file-input"
          @change="handleFileUpload"
          accept=".xlsx"
        />
        <span v-if="fileName" class="file-name">{{ fileName }}</span>
        <p v-if="errorMessage" class="error">{{ errorMessage }}</p>
      </div>
    </div>

    <div v-if="sheets.length > 0" class="sheets-section">
      <h3>Excel工作表</h3>
      <div class="tabs">
        <button
          v-for="(sheet, index) in sheets"
          :key="index"
          @click="selectSheet(index)"
          :class="{ active: selectedSheetIndex === index }"
          class="tab-button"
        >
          {{ sheet.name }}
        </button>
      </div>

      <div v-if="selectedSheetIndex !== null" class="sheet-content">
        <h4>{{ sheets[selectedSheetIndex].name }} 内容</h4>
        <div class="info-box">
          <p>
            <i class="info-icon">ℹ️</i>
            每个设备下的摄像头将从1开始自动编号，并设置到摄像头配置中的
            camera_index 字段。
          </p>
          <p>
            <i class="info-icon">📊</i> 当前共有
            {{ Object.keys(groupedByDevice).length }} 个设备，{{
              processedRows.length
            }}
            个摄像头。
          </p>
        </div>
        <div class="selection-actions">
          <button
            @click="toggleSelectAll"
            class="config-button"
            style="
              background-color: #6c757d;
              font-size: 13px;
              padding: 6px 10px;
            "
          >
            {{ isAllSelected ? "清除全选" : "全选" }}
          </button>
          <span class="selected-count"
            >已选择 {{ selectedCount }} /
            {{ processedRows.length }} 个摄像头</span
          >
        </div>
        <div class="table-container">
          <table>
            <thead>
              <tr>
                <th class="checkbox-column">
                  <input
                    type="checkbox"
                    :checked="isAllSelected"
                    :indeterminate="isPartiallySelected"
                    @change="toggleSelectAll"
                  />
                </th>
                <th class="number-column">序号</th>
                <th v-for="(header, index) in headers" :key="index">
                  {{ header }}
                </th>
                <th class="number-column">设备内索引</th>
              </tr>
            </thead>
            <tbody>
              <template v-for="(row, index) in processedRows" :key="index">
                <!-- 如果是设备的第一个摄像头，显示设备分组标头 -->
                <tr
                  v-if="isFirstCameraInDevice(row.deviceIp, index)"
                  class="device-group-header"
                >
                  <td colspan="5">
                    设备: {{ row.deviceIp }}
                    <span class="device-camera-count"
                      >(共
                      {{ groupedByDevice[row.deviceIp].count }} 个摄像头)</span
                    >
                  </td>
                </tr>
                <tr>
                  <td class="checkbox-column">
                    <input
                      type="checkbox"
                      v-model="row.selected"
                      @change="updateSelectionState"
                    />
                  </td>
                  <td class="number-column">
                    {{ getCameraIndex(row.deviceIp, index) }}
                  </td>
                  <td
                    v-for="(value, colIndex) in [
                      row.deviceIp,
                      row.cameraName,
                      row.cameraInfo,
                    ]"
                    :key="colIndex"
                  >
                    {{ value }}
                  </td>
                  <td class="number-column">
                    {{ row.deviceIndex || getCameraIndex(row.deviceIp, index) }}
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <div v-if="processedRows.length > 0" class="config-section">
      <h3>配置参数</h3>
      <div class="config-form">
        <div class="form-group">
          <label for="username">设备用户名</label>
          <input
            type="text"
            id="username"
            v-model="username"
            placeholder="设备用户名"
          />
        </div>
        <div class="form-group">
          <label for="password">设备密码</label>
          <input
            type="password"
            id="password"
            v-model="password"
            placeholder="设备密码"
          />
        </div>
        <div class="form-group">
          <label for="urlTemplate">摄像头URL模板</label>
          <input
            type="text"
            id="urlTemplate"
            v-model="urlTemplate"
            placeholder="例如: rtsp://admin:123@<ip>/av/stream"
            style="width: 100%"
          />
          <small>使用 &lt;ip&gt; 作为摄像头IP的占位符</small>
        </div>
        <div class="form-group">
          <label>算法选择</label>
          <div class="radio-group">
            <label>
              <input
                type="radio"
                name="algorithm"
                :value="6"
                v-model="algorithmType"
              />
              精准喷淋
            </label>
            <label>
              <input
                type="radio"
                name="algorithm"
                :value="7"
                v-model="algorithmType"
              />
              牛行为统计
            </label>
          </div>
        </div>
        <button
          @click="startConfiguration"
          class="config-button"
          :disabled="isConfiguring"
        >
          {{ isConfiguring ? "配置中..." : "开始配置" }}
        </button>
      </div>
    </div>

    <div v-if="configResults.length > 0" class="results-section">
      <h3>配置结果</h3>
      <div class="table-container">
        <table>
          <thead>
            <tr>
              <th class="number-column">序号</th>
              <th>设备IP</th>
              <th>摄像头名称</th>
              <th>设备内索引</th>
              <th>状态</th>
              <th>消息</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(result, index) in configResults"
              :key="index"
              :class="{ success: result.success, error: !result.success }"
            >
              <td class="number-column">{{ index + 1 }}</td>
              <td>{{ result.deviceIp }}</td>
              <td>{{ result.cameraName }}</td>
              <td>
                {{
                  getCameraIndexFromResult(result.deviceIp, result.cameraName)
                }}
              </td>
              <td>{{ result.success ? "成功" : "失败" }}</td>
              <td>{{ result.message }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import * as XLSX from "xlsx";

// 导入Go后端绑定
// 使用window.go.main.App作为临时解决方案
const App = window.go?.main?.App;

// 定义类型
interface SheetInfo {
  name: string;
}

interface ExcelRow {
  deviceIp: string;
  cameraName: string;
  cameraInfo: string;
  selected: boolean;
  deviceIndex?: number; // 设备内索引，从1开始
}

interface ConfigResult {
  deviceIp: string;
  cameraName: string;
  success: boolean;
  message: string;
}

// 状态变量
const fileName = ref<string>("");
const errorMessage = ref<string>("");
const sheets = ref<SheetInfo[]>([]);
const selectedSheetIndex = ref<number | null>(null);
const rawSheetData = ref<any[][]>([]);
const username = ref<string>("admin");
const password = ref<string>("admin");
const urlTemplate = ref<string>("rtsp://admin:123@<ip>/av/stream");
const algorithmType = ref<number>(6); // 默认精准喷淋
const isConfiguring = ref<boolean>(false);
const configResults = ref<ConfigResult[]>([]);

// 选择状态相关计算属性
const isAllSelected = computed<boolean>(() => {
  return (
    processedRows.value.length > 0 &&
    processedRows.value.every((row) => row.selected)
  );
});

const isPartiallySelected = computed<boolean>(() => {
  return (
    processedRows.value.some((row) => row.selected) && !isAllSelected.value
  );
});

const selectedCount = computed<number>(() => {
  return processedRows.value.filter((row) => row.selected).length;
});

// 切换全选/取消全选
const toggleSelectAll = () => {
  const newState = !isAllSelected.value;
  processedRows.value.forEach((row) => {
    row.selected = newState;
  });
};

// 更新选择状态
const updateSelectionState = () => {
  // 此方法保留为钩子，当改变单个项目时会触发
  // 实际计算由计算属性处理
};

// 处理Excel数据，合并单元格并过滤无效数据
const processedRows = ref<ExcelRow[]>([]);

// 根据设备IP分组摄像头并计算索引的计算属性
const groupedByDevice = computed(() => {
  const groups: Record<string, { rows: ExcelRow[]; count: number }> = {};

  processedRows.value.forEach((row) => {
    if (!groups[row.deviceIp]) {
      groups[row.deviceIp] = { rows: [], count: 0 };
    }
    groups[row.deviceIp].rows.push(row);
    groups[row.deviceIp].count++;
  });

  return groups;
});

// 获取指定行在其设备组中的索引（从1开始）
const getCameraIndex = (deviceIp: string, rowIndex: number) => {
  const row = processedRows.value[rowIndex];
  if (row.deviceIndex) {
    return row.deviceIndex;
  }

  let count = 0;
  for (let i = 0; i < processedRows.value.length; i++) {
    if (processedRows.value[i].deviceIp === deviceIp) {
      count++;
      if (i === rowIndex) {
        // 缓存索引
        processedRows.value[i].deviceIndex = count;
        return count;
      }
    }
  }
  return 0;
};

// 定义表头
const headers = ref(["设备IP", "摄像头名称", "摄像头IP/掩码/网关"]);

// 检查IP格式是否有效
const isValidIP = (ip: string): boolean => {
  // 检查IP地址格式 (IPv4)
  const ipPattern =
    /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;
  return ipPattern.test(ip);
};

// 处理文件上传
const handleFileUpload = (event: Event) => {
  const input = event.target as HTMLInputElement;
  if (!input.files || !input.files[0]) return;

  const file = input.files[0];
  fileName.value = file.name;
  errorMessage.value = "";

  const reader = new FileReader();

  reader.onload = (e: ProgressEvent<FileReader>) => {
    try {
      if (!e.target || !e.target.result) return;

      const data = new Uint8Array(e.target.result as ArrayBuffer);
      const workbook = XLSX.read(data, { type: "array" });

      // 获取工作表名称
      const sheetNames = workbook.SheetNames;
      sheets.value = sheetNames.map((name) => ({ name }));

      // 获取所有工作表数据
      rawSheetData.value = sheetNames.map((name) => {
        const worksheet = workbook.Sheets[name];
        return XLSX.utils.sheet_to_json(worksheet, { header: 1 });
      });

      // 默认选择第一个工作表
      if (sheets.value.length > 0) {
        selectedSheetIndex.value = 0;
        processSheetData();
      }
    } catch (error) {
      errorMessage.value = `解析Excel文件失败: ${
        error instanceof Error ? error.message : String(error)
      }`;
      sheets.value = [];
      rawSheetData.value = [];
      selectedSheetIndex.value = null;
      processedRows.value = [];
    }
  };

  reader.onerror = () => {
    errorMessage.value = "读取文件失败";
  };

  reader.readAsArrayBuffer(file);
};

// 处理工作表数据
const processSheetData = () => {
  if (selectedSheetIndex.value === null || !rawSheetData.value.length) {
    processedRows.value = [];
    return;
  }

  const data = rawSheetData.value[selectedSheetIndex.value];
  const rawRows: ExcelRow[] = [];
  let lastDeviceIp = "";

  // 从倒数第三列开始处理数据
  for (const row of data) {
    // 跳过标题行或无效行
    if (!row || row.length < 3) continue;

    // 提取后三列
    let cameraInfo = row[row.length - 1];
    const cameraName = row[row.length - 2];
    let deviceIp = row[row.length - 3] || lastDeviceIp; // 如果为空，使用上一行的值

    // 更新最后使用的设备IP
    if (deviceIp) {
      lastDeviceIp = deviceIp;
    }

    // 过滤掉"/"数据
    if (cameraInfo === "/") continue;
    if (deviceIp === "") continue;

    // 提取IP地址（如果包含掩码等）
    if (deviceIp.includes("/")) {
      deviceIp = deviceIp.split("/")[0];
    }

    // 从摄像头信息中提取摄像头IP
    let cameraIP = cameraInfo;
    if (cameraInfo.includes("/")) {
      cameraIP = cameraInfo.split("/")[0];
    }

    // 验证设备IP和摄像头IP是否符合IP格式
    if (!isValidIP(deviceIp) || !isValidIP(cameraIP)) {
      console.log(`跳过无效IP: 设备IP=${deviceIp}, 摄像头IP=${cameraIP}`);
      continue;
    }

    rawRows.push({
      deviceIp,
      cameraName,
      cameraInfo,
      selected: true,
    });
  }

  // 对处理后的数据按设备IP分组并为每组内的摄像头分配索引
  const deviceGroups: Record<string, ExcelRow[]> = {};

  // 先分组
  for (const row of rawRows) {
    if (!deviceGroups[row.deviceIp]) {
      deviceGroups[row.deviceIp] = [];
    }
    deviceGroups[row.deviceIp].push(row);
  }

  // 生成最终的处理结果，添加索引
  const result: ExcelRow[] = [];

  // 将分组后的数据展平为数组，并为每个设备内的摄像头分配索引
  for (const deviceIp in deviceGroups) {
    const deviceRows = deviceGroups[deviceIp];
    for (let i = 0; i < deviceRows.length; i++) {
      deviceRows[i].deviceIndex = i + 1; // 从1开始的索引
    }
    result.push(...deviceRows);
  }

  processedRows.value = result;
};

// 选择工作表
const selectSheet = (index: number) => {
  selectedSheetIndex.value = index;
  processSheetData();
};

// 开始配置摄像头
const startConfiguration = async () => {
  if (isConfiguring.value) return;

  if (!username.value || !password.value || !urlTemplate.value) {
    errorMessage.value = "请填写所有配置参数";
    return;
  }

  if (!urlTemplate.value.includes("<ip>")) {
    errorMessage.value = "URL模板必须包含<ip>占位符";
    return;
  }

  // 筛选出已选中的摄像头
  const selectedRows = processedRows.value.filter((row) => row.selected);

  if (selectedRows.length === 0) {
    errorMessage.value = "请至少选择一个摄像头进行配置";
    return;
  }

  // 按设备重新计算索引（考虑用户可能只选择了部分摄像头）
  const tempGroups: Record<string, ExcelRow[]> = {};
  selectedRows.forEach((row) => {
    if (!tempGroups[row.deviceIp]) {
      tempGroups[row.deviceIp] = [];
    }
    tempGroups[row.deviceIp].push({ ...row });
  });

  // 重新分配索引
  const processedSelectedRows: ExcelRow[] = [];
  for (const deviceIp in tempGroups) {
    const deviceRows = tempGroups[deviceIp];
    for (let i = 0; i < deviceRows.length; i++) {
      deviceRows[i].deviceIndex = i + 1;
      processedSelectedRows.push(deviceRows[i]);
    }
  }

  isConfiguring.value = true;
  errorMessage.value = "";
  configResults.value = [];

  try {
    // 调用Go后端方法配置摄像头
    if (App && App.ConfigureCamerasFromData) {
      const results = await App.ConfigureCamerasFromData(
        processedSelectedRows,
        username.value,
        password.value,
        urlTemplate.value,
        algorithmType.value
      );
      configResults.value = results;
    } else {
      throw new Error("后端方法未定义");
    }
  } catch (error) {
    errorMessage.value = `配置过程中发生错误: ${
      error instanceof Error ? error.message : String(error)
    }`;
  } finally {
    isConfiguring.value = false;
  }
};

// 获取结果中设备的索引
const getCameraIndexFromResult = (deviceIp: string, cameraName: string) => {
  // 首先在处理过的数据中查找匹配的行
  for (const row of processedRows.value) {
    if (
      row.deviceIp === deviceIp &&
      row.cameraName === cameraName &&
      row.selected
    ) {
      // 如果找到匹配的行并且有deviceIndex属性，直接返回
      if (row.deviceIndex) {
        return row.deviceIndex;
      }
      break;
    }
  }

  // 如果上述方法没找到，退回到原来的计数方法
  let deviceIndex = 0;
  for (const row of processedRows.value) {
    if (row.deviceIp === deviceIp && row.selected) {
      deviceIndex++;
      if (row.cameraName === cameraName) {
        return deviceIndex;
      }
    }
  }
  return "-";
};

// 检查是否是设备的第一个摄像头
const isFirstCameraInDevice = (deviceIp: string, rowIndex: number) => {
  if (rowIndex === 0) return true;

  const prevRow = processedRows.value[rowIndex - 1];
  return prevRow.deviceIp !== deviceIp;
};
</script>

<style scoped>
.camera-config-container {
  width: 100%;
  color: var(--text-color);
}

h2,
h3,
h4 {
  color: var(--text-color);
  margin-bottom: 15px;
}

.upload-section,
.sheets-section,
.config-section,
.results-section {
  margin-bottom: 20px;
  padding: 15px;
  border-radius: 5px;
  border: 1px solid var(--border-color);
  background-color: var(--card-background);
}

.file-upload {
  margin: 15px 0;
}

.file-input {
  display: none;
}

.file-label {
  display: inline-block;
  padding: 8px 16px;
  background-color: var(--primary-color);
  color: white;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.file-label:hover {
  background-color: var(--primary-hover);
}

.error {
  color: var(--danger-color);
  margin-top: 5px;
  font-size: 14px;
}

.success {
  color: var(--success-color);
}

.tabs {
  display: flex;
  flex-wrap: wrap;
  margin-bottom: 15px;
  border-bottom: 1px solid var(--border-color);
}

.tab-button {
  padding: 8px 15px;
  margin-right: 5px;
  margin-bottom: -1px;
  background-color: #f1f1f1;
  border: 1px solid var(--border-color);
  border-bottom: none;
  border-radius: 4px 4px 0 0;
  cursor: pointer;
}

.tab-button.active {
  background-color: var(--card-background);
  border-bottom: 1px solid var(--card-background);
}

.table-container {
  overflow-x: auto;
  margin-top: 15px;
}

table {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 15px;
}

th,
td {
  padding: 10px;
  text-align: left;
  border: 1px solid var(--border-color);
}

th {
  background-color: #f5f5f5;
  font-weight: 600;
}

.checkbox-column {
  width: 40px;
  text-align: center;
}

.number-column {
  width: 60px;
  text-align: center;
}

.selection-actions {
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.select-all-label {
  display: flex;
  align-items: center;
  cursor: pointer;
  font-weight: 500;
}

.select-all-label input[type="checkbox"] {
  margin-right: 8px;
}

.selected-count {
  font-size: 14px;
  color: #555;
}

tr.success {
  background-color: rgba(6, 214, 160, 0.1);
}

tr.success td {
  color: var(--text-color);
}

tr.error {
  background-color: rgba(239, 71, 111, 0.1);
}

tr.error td {
  color: var(--text-color);
}

.config-form {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.form-group {
  display: flex;
  flex-direction: column;
  margin-bottom: 15px;
}

label {
  margin-bottom: 5px;
  font-weight: 500;
}

input[type="text"],
input[type="password"] {
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 14px;
}

.radio-group {
  display: flex;
  gap: 15px;
}

.config-button {
  padding: 8px 16px;
  background-color: var(--primary-color);
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 500;
}

.config-button:hover:not(:disabled) {
  background-color: var(--primary-hover);
}

.config-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

small {
  color: #757575;
  margin-top: 3px;
}

.sheet-selector {
  padding: 8px;
  border-radius: 4px;
  border: 1px solid var(--border-color);
  width: 100%;
  max-width: 300px;
  margin-bottom: 15px;
  background-color: white;
}

.sheet-actions {
  display: flex;
  gap: 10px;
  margin-bottom: 15px;
}

.config-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.config-title {
  margin: 0;
  font-size: 18px;
}

.info-box {
  background-color: rgba(66, 153, 225, 0.1);
  border-left: 4px solid var(--primary-color);
  padding: 12px 15px;
  margin-bottom: 15px;
  border-radius: 4px;
}

.info-icon {
  margin-right: 8px;
}

.device-group-header {
  background-color: #f0f4f8;
}

.device-group-header td {
  font-weight: 600;
  padding: 8px 10px;
  color: var(--primary-color);
  border-top: 2px solid var(--primary-color);
}

.device-camera-count {
  font-weight: normal;
  font-size: 13px;
  color: #666;
  margin-left: 8px;
}
</style>
