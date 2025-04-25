<template>
  <div class="backup-container">
    <h2>设备备份管理</h2>

    <!-- 通知组件 -->
    <div v-if="notification.show" class="notification" :class="notification.type">
      <span class="notification-message">{{ notification.message }}</span>
      <button class="notification-close" @click="notification.show = false">×</button>
    </div>

    <!-- 确认对话框 -->
    <div v-if="confirmDialog.show" class="confirm-dialog-overlay">
      <div class="confirm-dialog">
        <div class="confirm-dialog-header">
          <h3>{{ confirmDialog.title }}</h3>
        </div>
        <div class="confirm-dialog-body">
          <p v-html="confirmDialog.message.replace(/\n/g, '<br>')"></p>
        </div>
        <div class="confirm-dialog-footer">
          <button @click="confirmDialog.onCancel" class="config-button secondary-button">取消</button>
          <button @click="confirmDialog.onConfirm" class="config-button danger-button">确认</button>
        </div>
      </div>
    </div>

    <div class="tabs">
      <button
        :class="{ active: activeSubTab === 'backup' }"
        @click="activeSubTab = 'backup'"
      >
        数据备份
      </button>
      <button
        :class="{ active: activeSubTab === 'restore' }"
        @click="activeSubTab = 'restore'"
      >
        数据恢复
      </button>
    </div>

    <!-- 数据备份页签 -->
    <div v-if="activeSubTab === 'backup'" class="tab-content">
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

      <div class="backup-settings">
        <h3>备份设置</h3>
        <div class="config-form">
          <div class="form-group">
            <label for="storageFolder">本地存储文件夹</label>
            <input
              type="text"
              id="storageFolder"
              v-model="storageFolder"
              placeholder="本地存储文件夹路径"
            />
            <button 
              @click="selectStorageFolder" 
              class="browse-button"
              title="选择文件夹"
            >
              浏览...
            </button>
          </div>
          <div class="form-group">
            <label for="regionName">区域名称</label>
            <input
              type="text"
              id="regionName"
              v-model="regionName"
              placeholder="区域名称"
            />
          </div>
        </div>
      </div>

      <div class="info-box">
        <p>
          <i class="info-icon">ℹ️</i>
          备份将从每个设备获取 /var/lib/application-web/db/application-web.db
          文件，并保存到指定的本地文件夹下，按照区域名称和IP地址进行组织。
        </p>
      </div>

      <div class="device-selection">
        <h3>选择设备</h3>

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

        <div class="backup-actions">
          <button
            @click="backupDevices"
            class="config-button primary-button"
            :disabled="
              isProcessing ||
              selectedCount === 0 ||
              !username ||
              !password ||
              !storageFolder ||
              !regionName
            "
          >
            {{ isProcessing ? "备份中..." : "开始备份" }}
          </button>
        </div>
      </div>

      <div v-if="backupResults.length > 0" class="results-section">
        <h3>备份结果</h3>
        <div class="table-container">
          <table>
            <thead>
              <tr>
                <th class="number-column">序号</th>
                <th>IP地址</th>
                <th>状态</th>
                <th>消息</th>
                <th>备份路径</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(result, index) in backupResults"
                :key="index"
                :class="{ success: result.success, error: !result.success }"
              >
                <td class="number-column">{{ index + 1 }}</td>
                <td>{{ result.ip }}</td>
                <td>{{ result.success ? "成功" : "失败" }}</td>
                <td>{{ result.message }}</td>
                <td>{{ result.backupPath }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- 数据恢复页签 -->
    <div v-if="activeSubTab === 'restore'" class="tab-content">
      <div class="auth-section">
        <h3>认证信息</h3>
        <div class="config-form">
          <div class="form-group">
            <label for="username-restore">SSH用户名</label>
            <input
              type="text"
              id="username-restore"
              v-model="username"
              placeholder="SSH用户名 (需要root权限)"
            />
          </div>
          <div class="form-group">
            <label for="password-restore">SSH密码</label>
            <input
              type="password"
              id="password-restore"
              v-model="password"
              placeholder="SSH密码"
            />
          </div>
        </div>
      </div>

      <div class="restore-settings">
        <h3>恢复设置</h3>
        <div class="config-form">
          <div class="form-group">
            <label for="storageFolder-restore">本地存储文件夹</label>
            <input
              type="text"
              id="storageFolder-restore"
              v-model="storageFolder"
              placeholder="本地存储文件夹路径"
            />
            <button 
              @click="selectStorageFolder" 
              class="browse-button"
              title="选择文件夹"
            >
              浏览...
            </button>
          </div>
          <div class="form-group">
            <label for="regionName-restore">区域名称</label>
            <input
              type="text"
              id="regionName-restore"
              v-model="regionName"
              placeholder="区域名称"
            />
          </div>
        </div>
      </div>

      <div class="info-box warning">
        <p>
          <i class="info-icon">⚠️</i>
          数据恢复将:
          <ol>
            <li>停止设备上的application-web服务</li>
            <li>备份当前数据库文件 (带有时间戳)</li>
            <li>复制所选备份文件到设备</li>
            <li>重新启动application-web服务</li>
          </ol>
          系统将为每台设备使用对应的备份文件: <strong>{本地存储文件夹}/{区域名称}/{设备IP}/application-web.db</strong>
        </p>
      </div>

      <div class="device-selection">
        <h3>选择设备</h3>

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

        <div class="restore-actions">
          <button
            @click="restoreDevices"
            class="config-button danger-button"
            :disabled="
              isProcessing ||
              selectedCount === 0 ||
              !username ||
              !password ||
              !storageFolder ||
              !regionName
            "
          >
            {{ isProcessing ? "恢复中..." : "开始恢复" }}
          </button>
        </div>
      </div>

      <div v-if="restoreResults.length > 0" class="results-section">
        <h3>恢复结果</h3>
        <div class="table-container">
          <table>
            <thead>
              <tr>
                <th class="number-column">序号</th>
                <th>IP地址</th>
                <th>状态</th>
                <th>消息</th>
                <th>原始备份路径</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(result, index) in restoreResults"
                :key="index"
                :class="{ success: result.success, error: !result.success }"
              >
                <td class="number-column">{{ index + 1 }}</td>
                <td>{{ result.ip }}</td>
                <td>{{ result.success ? "成功" : "失败" }}</td>
                <td>{{ result.message }}</td>
                <td>{{ result.backupPath }}</td>
              </tr>
            </tbody>
          </table>
        </div>
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

interface BackupResult {
  ip: string;
  success: boolean;
  message: string;
  backupPath: string;
}

interface RestoreResult {
  ip: string;
  success: boolean;
  message: string;
  originalPath: string;
  backupPath: string;
}

// 通知状态
interface Notification {
  show: boolean;
  message: string;
  type: 'info' | 'success' | 'error' | 'warning';
}

// 确认对话框状态
interface ConfirmDialog {
  show: boolean;
  title: string;
  message: string;
  onConfirm: () => void;
  onCancel: () => void;
}

// 状态变量
const activeSubTab = ref<string>("backup");
const username = ref<string>("root");
const password = ref<string>("ematech");
const storageFolder = ref<string>("");
const regionName = ref<string>("");
const devices = ref<DeviceWithSelection[]>([]);
const isProcessing = ref<boolean>(false);
const backupResults = ref<BackupResult[]>([]);
const restoreResults = ref<RestoreResult[]>([]);
const notification = ref<Notification>({
  show: false,
  message: "",
  type: 'info'
});
const confirmDialog = ref<ConfirmDialog>({
  show: false,
  title: "",
  message: "",
  onConfirm: () => {},
  onCancel: () => {}
});

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

// 显示通知的函数
const showNotification = (message: string, type: 'info' | 'success' | 'error' | 'warning' = 'info') => {
  notification.value = {
    show: true,
    message,
    type
  };
  
  // 5秒后自动关闭通知
  setTimeout(() => {
    notification.value.show = false;
  }, 5000);
};

// 显示确认对话框的函数
const showConfirmDialog = (title: string, message: string): Promise<boolean> => {
  return new Promise((resolve) => {
    confirmDialog.value = {
      show: true,
      title,
      message,
      onConfirm: () => {
        confirmDialog.value.show = false;
        resolve(true);
      },
      onCancel: () => {
        confirmDialog.value.show = false;
        resolve(false);
      }
    };
  });
};

// 选择存储文件夹
const selectStorageFolder = async () => {
  try {
    if (!App) {
      console.error("后端App不可用");
      return;
    }
    
    // 保存当前值，以便取消时恢复
    const currentValue = storageFolder.value;

    const folderPath = await App.SelectFolder();
    if (folderPath) {
      storageFolder.value = folderPath;
      console.log("已选择存储文件夹:", folderPath);
    }
  } catch (error: any) {
    // 如果是取消操作，不显示错误
    if (error.toString().includes("CANCELED")) {
      console.log("用户取消了文件夹选择");
      return;
    }
    
    console.error("选择文件夹失败:", error);
    showNotification(`选择文件夹失败: ${error}`, 'error');
  }
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

// 备份设备
const backupDevices = async () => {
  try {
    isProcessing.value = true;
    backupResults.value = [];

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
    
    // 使用自定义确认对话框
    const confirmMessage = 
      `确定要备份 ${selectedDevices.length} 台设备的数据库？\n` +
      `备份将保存到: ${storageFolder.value}/${regionName.value}/{设备IP}/application-web.db`;
      
    const confirmed = await showConfirmDialog("确认备份操作", confirmMessage);
    
    if (!confirmed) {
      isProcessing.value = false;
      console.log("用户取消了备份操作");
      return; // 静默退出
    }

    // 调用后端备份功能
    const results = await App.BackupDevices(
      username.value,
      password.value,
      storageFolder.value,
      regionName.value,
      selectedDevices
    );

    // 更新结果
    backupResults.value = results;
    
    // 显示成功通知
    showNotification("数据备份操作已完成，请查看结果", 'success');
  } catch (error: any) {
    console.error("备份失败:", error);
    // 显示错误信息
    backupResults.value = [
      {
        ip: "系统错误",
        success: false,
        message: `备份过程出错: ${error}`,
        backupPath: "",
      },
    ];
    
    // 显示错误通知
    showNotification(`备份过程出错: ${error}`, 'error');
  } finally {
    isProcessing.value = false;
  }
};

// 恢复设备
const restoreDevices = async () => {
  console.log("恢复按钮被点击，开始执行恢复流程");
  console.log("当前状态：", {
    isProcessing: isProcessing.value,
    selectedCount: selectedCount.value,
    username: username.value,
    password: password.value ? "[已设置]" : "[未设置]",
    storageFolder: storageFolder.value,
    regionName: regionName.value
  });
  
  try {
    isProcessing.value = true;
    console.log("isProcessing 设置为 true");
    restoreResults.value = [];

    if (!App) {
      console.error("App对象不可用");
      throw new Error("后端App不可用");
    }

    // 获取选中的设备
    const selectedDevices = devices.value
      .filter((device) => device.selected)
      .map((device) => device.ip);
    
    console.log("选中的设备：", selectedDevices);

    if (selectedDevices.length === 0) {
      throw new Error("未选择任何设备");
    }

    // 检查必填字段
    if (!storageFolder.value || !regionName.value) {
      throw new Error("请指定本地存储文件夹和区域名称");
    }

    // 使用自定义确认对话框
    console.log("准备显示确认对话框");
    const confirmMessage = 
      `确定要将 ${selectedDevices.length} 台设备恢复到各自的备份？\n` +
      `备份位置格式: ${storageFolder.value}/${regionName.value}/{设备IP}/application-web.db\n` +
      `这将重启设备上的application-web服务。`;
    
    // 使用自定义确认对话框
    const confirmed = await showConfirmDialog("确认恢复操作", confirmMessage);
    
    if (!confirmed) {
      isProcessing.value = false;
      console.log("用户取消了恢复操作");
      return; // 静默退出，不显示错误
    }
    
    console.log("用户确认了恢复操作，开始调用后端API");

    // 调用后端恢复功能
    const results = await App.RestoreDevicesDB(
      username.value,
      password.value,
      storageFolder.value,
      regionName.value,
      selectedDevices
    );
    
    console.log("恢复操作完成，结果：", results);

    // 更新结果
    restoreResults.value = results;
    
    // 显示成功通知
    showNotification("数据恢复操作已完成，请查看结果", 'success');
  } catch (error: any) {
    console.error("恢复失败:", error);
    
    // 检查是否是"操作已取消"错误，如果是则不显示错误结果
    if (error.toString().includes("操作已取消")) {
      console.log("用户取消了恢复操作");
      // 清空错误结果
      restoreResults.value = [];
    } else {
      // 显示错误信息
      restoreResults.value = [
        {
          ip: "系统错误",
          success: false,
          message: `恢复过程出错: ${error}`,
          originalPath: "",
          backupPath: "",
        },
      ];
      
      // 显示错误通知
      showNotification(`恢复过程出错: ${error}`, 'error');
    }
  } finally {
    console.log("恢复流程结束，重置isProcessing状态");
    isProcessing.value = false;
  }
};

// 加载保存的备份设置
const loadSavedBackupSettings = async () => {
  try {
    if (!App) {
      console.error("后端App不可用");
      return;
    }

    const settings = await App.GetBackupSettings();
    if (settings) {
      storageFolder.value = settings.storageFolder || "";
      regionName.value = settings.regionName || "";
      username.value = settings.username || "root";
      password.value = settings.password || "ematech";
      console.log("已加载保存的备份设置");
    }
  } catch (error) {
    console.error("加载备份设置失败:", error);
  }
};

// 组件挂载时加载设备列表和备份设置
onMounted(() => {
  loadDevices();
  loadSavedBackupSettings();
});
</script>

<style scoped>
.backup-container {
  max-width: 100%;
  margin: 0;
  padding: 0 20px;
  width: 100%;
  box-sizing: border-box;
}

.tabs {
  display: flex;
  margin-bottom: 20px;
  border-bottom: 1px solid #dee2e6;
  width: 100%;
}

.tabs button {
  padding: 10px 20px;
  background-color: transparent;
  color: #212529;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  font-size: 16px;
  transition: all 0.2s;
  margin-right: 10px;
}

.tabs button.active {
  color: #4361ee;
  border-bottom: 2px solid #4361ee;
}

.tab-content {
  margin-top: 20px;
  width: 100%;
  background-color: inherit;
}

.config-form {
  margin-bottom: 20px;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 15px;
}

.form-group {
  margin-bottom: 10px;
  display: flex;
  align-items: center;
}

.form-group label {
  display: block;
  margin-bottom: 5px;
  font-weight: 500;
  width: 100%;
}

.browse-button {
  margin-left: 10px;
  flex-shrink: 0;
}

.info-box {
  background-color: #f8f9fa;
  border-left: 4px solid #4361ee;
  padding: 10px 15px;
  margin-bottom: 20px;
  border-radius: 4px;
}

.info-box.warning {
  background-color: #fff3cd;
  border-left: 4px solid #ffc107;
}

.info-icon {
  margin-right: 8px;
}

.selection-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.selected-count {
  font-size: 14px;
  color: #6c757d;
}

.table-container {
  width: 100%;
  overflow-x: auto;
  margin-bottom: 20px;
  box-sizing: border-box;
}

table {
  width: 100%;
  border-collapse: collapse;
}

th,
td {
  padding: 10px;
  text-align: left;
  border-bottom: 1px solid #dee2e6;
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
}

.status {
  display: inline-block;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.status-online {
  background-color: #06d6a0;
  color: white;
}

.status-offline {
  background-color: #ef476f;
  color: white;
}

.empty-state {
  padding: 20px;
  text-align: center;
  background-color: #f8f9fa;
  border-radius: 4px;
  color: #6c757d;
}

.backup-actions,
.restore-actions {
  margin-top: 20px;
}

.primary-button {
  background-color: #4361ee;
  padding: 10px 20px;
  font-size: 16px;
}

.danger-button {
  background-color: #ef476f;
  padding: 10px 20px;
  font-size: 16px;
}

.secondary-button {
  background-color: #6c757d;
}

.results-section {
  margin-top: 30px;
  width: 100%;
  box-sizing: border-box;
}

.results-section table tr.success td {
  background-color: rgba(6, 214, 160, 0.1);
}

.results-section table tr.error td {
  background-color: rgba(239, 71, 111, 0.1);
}

.config-button {
  padding: 8px 16px;
  background-color: #4361ee;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.config-button:hover:not(:disabled) {
  background-color: #3a56d4;
}

.config-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.config-button.danger-button {
  background-color: #ef476f;
}

.config-button.danger-button:hover:not(:disabled) {
  background-color: #d63d63;
}

/* 通知样式 */
.notification {
  position: relative;
  padding: 12px 16px;
  margin-bottom: 16px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  animation: slide-in 0.3s ease-out;
}

@keyframes slide-in {
  from {
    transform: translateY(-20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.notification-message {
  flex-grow: 1;
}

.notification-close {
  background: transparent;
  border: none;
  color: inherit;
  font-size: 18px;
  cursor: pointer;
  padding: 0 4px;
  margin-left: 8px;
}

.notification.info {
  background-color: #e3f2fd;
  border-left: 4px solid #2196f3;
  color: #0d47a1;
}

.notification.success {
  background-color: #e8f5e9;
  border-left: 4px solid #4caf50;
  color: #2e7d32;
}

.notification.warning {
  background-color: #fff3e0;
  border-left: 4px solid #ff9800;
  color: #e65100;
}

.notification.error {
  background-color: #ffebee;
  border-left: 4px solid #f44336;
  color: #b71c1c;
}

/* 确认对话框样式 */
.confirm-dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
  animation: fade-in 0.2s ease-out;
}

@keyframes fade-in {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.confirm-dialog {
  background-color: white;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
  width: 90%;
  max-width: 500px;
  max-height: 90vh;
  overflow-y: auto;
  animation: dialog-slide-up 0.3s ease-out;
}

@keyframes dialog-slide-up {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.confirm-dialog-header {
  padding: 16px 20px;
  border-bottom: 1px solid #eeeeee;
}

.confirm-dialog-header h3 {
  margin: 0;
  font-size: 18px;
  color: #333;
}

.confirm-dialog-body {
  padding: 20px;
  max-height: 60vh;
  overflow-y: auto;
}

.confirm-dialog-body p {
  margin: 0 0 16px 0;
  line-height: 1.5;
}

.confirm-dialog-footer {
  padding: 16px 20px;
  border-top: 1px solid #eeeeee;
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
