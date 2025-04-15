<template>
  <div class="time-sync-container">
    <h2>设备时间同步</h2>

    <div class="auth-section">
      <h3>认证信息</h3>
      <div class="config-form">
        <div class="form-group">
          <label for="username">SSH用户名</label>
          <input
            type="text"
            id="username"
            v-model="username"
            placeholder="SSH用户名 (需要root权限)"
          />
        </div>
        <div class="form-group">
          <label for="password">SSH密码</label>
          <input
            type="password"
            id="password"
            v-model="password"
            placeholder="SSH密码"
          />
        </div>
      </div>
    </div>

    <div class="device-selection">
      <h3>选择设备</h3>
      <div class="info-box">
        <p>
          <i class="info-icon">ℹ️</i>
          系统时间同步将使用当前计算机的时间，通过SSH连接设备并执行date命令设置设备时间。
        </p>
      </div>

      <div class="selection-actions">
        <button @click="toggleSelectAll" class="config-button secondary-button">
          {{ isAllSelected ? "清除全选" : "全选" }}
        </button>
        <span class="selected-count"
          >已选择 {{ selectedCount }} / {{ devices.length }} 个设备</span
        >
      </div>

      <div v-if="devices.length === 0" class="empty-state">
        未发现设备。请先在设备管理中添加设备。
      </div>

      <div v-else class="table-container">
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
              <th>IP地址</th>
              <th>构建时间</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(device, index) in devices" :key="device.ip">
              <td class="checkbox-column">
                <input
                  type="checkbox"
                  v-model="device.selected"
                  :disabled="device.status !== 'online'"
                  @change="updateSelectionState"
                />
              </td>
              <td class="number-column">{{ index + 1 }}</td>
              <td>{{ device.ip }}</td>
              <td>{{ device.buildTime }}</td>
              <td>
                <span
                  :class="[
                    'status',
                    device.status === 'online'
                      ? 'status-online'
                      : 'status-offline',
                  ]"
                >
                  {{ device.status === "online" ? "在线" : "离线" }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="sync-actions">
        <button
          @click="syncTime"
          class="config-button primary-button"
          :disabled="
            isProcessing || selectedCount === 0 || !username || !password
          "
        >
          {{ isProcessing ? "同步中..." : "开始时间同步" }}
        </button>
      </div>
    </div>

    <div v-if="syncResults.length > 0" class="results-section">
      <h3>同步结果</h3>
      <div class="table-container">
        <table>
          <thead>
            <tr>
              <th class="number-column">序号</th>
              <th>IP地址</th>
              <th>状态</th>
              <th>消息</th>
              <th>同步时间</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(result, index) in syncResults"
              :key="index"
              :class="{ success: result.success, error: !result.success }"
            >
              <td class="number-column">{{ index + 1 }}</td>
              <td>{{ result.ip }}</td>
              <td>{{ result.success ? "成功" : "失败" }}</td>
              <td>{{ result.message }}</td>
              <td>{{ result.timestamp }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from "vue";

// 使用window.go.main.App作为临时解决方案
const App = window.go?.main?.App;

// 定义接口类型
interface DeviceWithSelection {
  ip: string;
  buildTime: string;
  status: string;
  selected: boolean;
}

interface TimeSyncResult {
  ip: string;
  success: boolean;
  message: string;
  timestamp: string;
}

// 状态变量
const username = ref<string>("root");
const password = ref<string>("");
const devices = ref<DeviceWithSelection[]>([]);
const isProcessing = ref<boolean>(false);
const syncResults = ref<TimeSyncResult[]>([]);

// 选择状态相关计算属性
const isAllSelected = computed<boolean>(() => {
  if (devices.value.length === 0) return false;
  return devices.value
    .filter((device) => device.status === "online")
    .every((device) => device.selected);
});

const isPartiallySelected = computed<boolean>(() => {
  const onlineDevices = devices.value.filter(
    (device) => device.status === "online"
  );
  return (
    onlineDevices.some((device) => device.selected) && !isAllSelected.value
  );
});

const selectedCount = computed<number>(() => {
  return devices.value.filter((device) => device.selected).length;
});

// 切换全选/取消全选
const toggleSelectAll = () => {
  const newState = !isAllSelected.value;
  devices.value.forEach((device) => {
    if (device.status === "online") {
      device.selected = newState;
    }
  });
};

// 更新选择状态
const updateSelectionState = () => {
  // 此方法保留为钩子，当改变单个项目时会触发
};

// 加载设备列表
const loadDevices = async () => {
  try {
    if (!App) {
      console.error("后端App不可用");
      return;
    }

    const deviceList = await App.GetDevices();
    devices.value = deviceList.map((device: any) => ({
      ...device,
      selected: device.status === "online",
    }));
  } catch (error) {
    console.error("加载设备列表失败:", error);
  }
};

// 同步时间
const syncTime = async () => {
  try {
    isProcessing.value = true;
    syncResults.value = [];

    if (!App) {
      throw new Error("后端App不可用");
    }

    // 获取选中的设备
    const selectedDevices = devices.value
      .filter((device) => device.selected)
      .map((device) => device.ip);

    if (selectedDevices.length === 0) {
      throw new Error("未选择任何设备");
    }

    // 调用后端同步时间
    const results = await App.SyncDeviceTime(
      username.value,
      password.value,
      selectedDevices
    );

    syncResults.value = results;

    // 刷新设备列表
    await loadDevices();
  } catch (error) {
    console.error("同步时间失败:", error);
    alert(
      `同步时间失败: ${error instanceof Error ? error.message : String(error)}`
    );
  } finally {
    isProcessing.value = false;
  }
};

// 组件挂载时加载设备列表
onMounted(() => {
  loadDevices();
});
</script>

<style scoped>
.time-sync-container {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

h2 {
  margin-top: 0;
  color: var(--text-color);
}

h3 {
  margin-top: 0;
  color: var(--text-color);
  font-size: 1.2rem;
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 0.5rem;
}

.auth-section,
.device-selection,
.results-section {
  background-color: var(--card-background);
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  padding: 1.5rem;
}

.config-form {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  min-width: 200px;
  flex: 1;
}

.form-group label {
  margin-bottom: 0.5rem;
  font-weight: 500;
}

.form-group input {
  padding: 0.7rem;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 0.9rem;
}

.info-box {
  background-color: rgba(67, 97, 238, 0.1);
  border-left: 4px solid var(--primary-color);
  padding: 1rem;
  margin-bottom: 1rem;
  border-radius: 0 4px 4px 0;
}

.info-icon {
  margin-right: 0.5rem;
}

.selection-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1rem;
}

.selected-count {
  font-size: 0.9rem;
  color: #666;
}

.table-container {
  overflow-x: auto;
  margin-bottom: 1rem;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.9rem;
}

th,
td {
  text-align: left;
  padding: 0.75rem;
  border-bottom: 1px solid var(--border-color);
}

th {
  background-color: #f8f9fa;
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

.status {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 12px;
  font-size: 0.8rem;
}

.status-online {
  background-color: rgba(6, 214, 160, 0.2);
  color: #06d6a0;
}

.status-offline {
  background-color: rgba(239, 71, 111, 0.2);
  color: #ef476f;
}

.sync-actions {
  margin-top: 1.5rem;
  display: flex;
  justify-content: center;
}

.config-button {
  padding: 0.7rem 1.5rem;
  border: none;
  border-radius: 4px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.2s;
}

.config-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.primary-button {
  background-color: var(--primary-color);
  color: white;
}

.primary-button:hover:not(:disabled) {
  background-color: var(--primary-hover);
}

.secondary-button {
  background-color: #6c757d;
  color: white;
}

.secondary-button:hover:not(:disabled) {
  background-color: #5a6268;
}

.empty-state {
  padding: 2rem;
  text-align: center;
  color: #6c757d;
  background-color: #f8f9fa;
  border-radius: 4px;
}

.results-section table tr.success {
  background-color: rgba(6, 214, 160, 0.05);
}

.results-section table tr.error {
  background-color: rgba(239, 71, 111, 0.05);
}
</style>
