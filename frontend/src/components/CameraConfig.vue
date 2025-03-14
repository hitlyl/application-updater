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
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, index) in processedRows" :key="index">
                <td class="checkbox-column">
                  <input
                    type="checkbox"
                    v-model="row.selected"
                    @change="updateSelectionState"
                  />
                </td>
                <td class="number-column">{{ index + 1 }}</td>
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
              </tr>
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

// 定义表头
const headers = ref(["设备IP", "摄像头名称", "摄像头IP/掩码/网关"]);

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
  const result: ExcelRow[] = [];
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
    if (deviceIp.includes("/")) {
      deviceIp = deviceIp.split("/")[0];
    }
    if (cameraInfo.includes("/")) {
      cameraInfo = cameraInfo.split("/")[0];
    }

    result.push({
      deviceIp,
      cameraName,
      cameraInfo,
      selected: true,
    });
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

  isConfiguring.value = true;
  errorMessage.value = "";
  configResults.value = [];

  try {
    // 调用Go后端方法配置摄像头
    if (App && App.ConfigureCamerasFromData) {
      const results = await App.ConfigureCamerasFromData(
        selectedRows,
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
</style>
