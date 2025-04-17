<script lang="ts" setup>
import { ref, onMounted, computed } from "vue";
import CameraConfig from "./components/CameraConfig.vue";
import TimeSync from "./components/TimeSync.vue";

// 避免TypeScript错误的类型声明
declare global {
  interface Window {
    go?: {
      main?: {
        App?: any;
      };
    };
  }
}

// Import functions from Wails-generated bindings
// These will be available after the first build
// Using window.go.main.App for now as a workaround
const App = window.go?.main?.App;

interface Device {
  ip: string;
  buildTime: string;
  status: string;
}

interface UpdateResult {
  ip: string;
  success: boolean;
  message: string;
}

// State
const devices = ref<Device[]>([]);
const newDeviceIP = ref("");
const startIP = ref("");
const endIP = ref("");
const username = ref("");
const password = ref("");
const isLoading = ref(false);
const scanLoading = ref(false);
const updateResults = ref<UpdateResult[]>([]);
const activeTab = ref("devices");
const selectedFile = ref("");
const selectedMd5File = ref("");
const selectedDevices = ref<Record<string, boolean>>({});
const selectAll = ref(true);
// 组展开状态
const groupExpanded = ref<Record<string, boolean>>({});
const showFileHelp = ref(false);

// 计算属性：已选择的设备IP列表
const selectedDevicesList = computed(() => {
  return Object.entries(selectedDevices.value)
    .filter(([_, selected]) => selected)
    .map(([ip]) => ip);
});

// 计算属性：显示选择的设备数量
const selectedDevicesCount = computed(() => {
  return selectedDevicesList.value.length;
});

// 用于设备列表分页的变量
const currentPage = ref(1);
const itemsPerPage = ref(10);
const searchQuery = ref("");
const filterStatus = ref("all"); // 'all', 'online', 'offline'

// 计算属性：已筛选的设备
const filteredDevices = computed(() => {
  let result = devices.value;

  // 按状态筛选
  if (filterStatus.value !== "all") {
    result = result.filter((device) => device.status === filterStatus.value);
  }

  // 按搜索关键词筛选
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    result = result.filter(
      (device) =>
        device.ip.toLowerCase().includes(query) ||
        device.buildTime.toLowerCase().includes(query)
    );
  }

  return result;
});

// 计算属性：当前页的设备
const paginatedDevices = computed(() => {
  const start = (currentPage.value - 1) * itemsPerPage.value;
  const end = start + itemsPerPage.value;
  return filteredDevices.value.slice(start, end);
});

// 计算属性：总页数
const totalPages = computed(() => {
  return Math.ceil(filteredDevices.value.length / itemsPerPage.value);
});

// 导航到指定页
function goToPage(page: number) {
  currentPage.value = page;
}

// 计算属性：按首字母分组的设备
const groupedDevices = computed(() => {
  const groups: Record<string, Device[]> = {};

  filteredDevices.value.forEach((device) => {
    // 使用buildTime作为分组键，而不是IP地址
    const groupKey = device.buildTime || "未知版本";
    if (!groups[groupKey]) {
      groups[groupKey] = [];
    }
    groups[groupKey].push(device);
  });

  // 初始化组展开状态
  Object.keys(groups).forEach((key) => {
    if (groupExpanded.value[key] === undefined) {
      groupExpanded.value[key] = true; // 默认展开
    }
  });

  return groups;
});

// 一次性选择/取消选择一个组内的所有设备
function toggleGroup(groupKey: string, selected: boolean) {
  const group = groupedDevices.value[groupKey] || [];
  const newSelection = { ...selectedDevices.value };

  group.forEach((device) => {
    if (device.status === "online") {
      newSelection[device.ip] = selected;
    }
  });

  selectedDevices.value = newSelection;
}

// 检查一个组是否全部选中
function isGroupAllSelected(groupKey: string): boolean {
  const group = groupedDevices.value[groupKey] || [];
  const onlineDevices = group.filter((device) => device.status === "online");

  if (onlineDevices.length === 0) return false;

  return onlineDevices.every((device) => selectedDevices.value[device.ip]);
}

// 检查一个组是否部分选中
function isGroupPartiallySelected(groupKey: string): boolean {
  const group = groupedDevices.value[groupKey] || [];
  const onlineDevices = group.filter((device) => device.status === "online");

  if (onlineDevices.length === 0) return false;

  const selectedCount = onlineDevices.filter(
    (device) => selectedDevices.value[device.ip]
  ).length;
  return selectedCount > 0 && selectedCount < onlineDevices.length;
}

// 更新选中的buildTime列表计算属性
const selectedBuildTimes = computed(() => {
  const buildTimes = new Set<string>();

  Object.entries(selectedDevices.value)
    .filter(([_, selected]) => selected)
    .forEach(([ip]) => {
      const device = devices.value.find((d) => d.ip === ip);
      if (device && device.buildTime) {
        buildTimes.add(device.buildTime);
      }
    });

  return Array.from(buildTimes);
});

// Fetch devices on component mount
onMounted(async () => {
  try {
    await loadDevices();
  } catch (error) {
    console.error("加载设备失败:", error);
  }
});

// Load devices from backend
async function loadDevices() {
  if (!App) return;

  isLoading.value = true;
  try {
    devices.value = await App.GetDevices();
    // 初始化选择状态
    updateDeviceSelection();
  } catch (error) {
    console.error("加载设备出错:", error);
  } finally {
    isLoading.value = false;
  }
}

// 更新设备选择状态
function updateDeviceSelection() {
  const newSelection: Record<string, boolean> = {};
  devices.value.forEach((device) => {
    // 保留已有的选择状态，或者根据selectAll设置新设备的选择状态
    newSelection[device.ip] =
      selectedDevices.value[device.ip] !== undefined
        ? selectedDevices.value[device.ip]
        : selectAll.value;
  });
  selectedDevices.value = newSelection;
}

// 切换全选/全不选
function toggleSelectAll() {
  selectAll.value = !selectAll.value;
  devices.value.forEach((device) => {
    selectedDevices.value[device.ip] = selectAll.value;
  });
}

// Refresh device statuses
async function refreshDevices() {
  if (!App) return;

  isLoading.value = true;
  try {
    devices.value = await App.RefreshDevices();
    // 保留已有设备的选择状态
    updateDeviceSelection();
  } catch (error) {
    console.error("刷新设备出错:", error);
  } finally {
    isLoading.value = false;
  }
}

// Add a new device
async function addDevice() {
  if (!App) {
    alert("无法连接到后端服务，请重新加载页面");
    return;
  }

  if (!newDeviceIP.value) {
    alert("请输入设备IP地址");
    return;
  }

  // 简单的IP地址格式验证
  const ipPattern = /^(\d{1,3}\.){3}\d{1,3}$/;
  if (!ipPattern.test(newDeviceIP.value)) {
    alert("请输入有效的IP地址格式，例如: 192.168.1.100");
    return;
  }

  isLoading.value = true;
  try {
    console.log("开始添加设备:", newDeviceIP.value);

    // 设置超时，防止长时间阻塞
    const addDevicePromise = App.AddDevice(newDeviceIP.value);
    const timeoutPromise = new Promise((_, reject) =>
      setTimeout(
        () => reject(new Error("添加设备超时，请检查设备是否在线")),
        10000
      )
    );

    await Promise.race([addDevicePromise, timeoutPromise]);
    console.log("设备添加成功，正在刷新设备列表");

    // 添加成功后重新加载设备列表
    await loadDevices();
    newDeviceIP.value = "";
  } catch (error: unknown) {
    console.error("添加设备出错:", error);

    // 提供更友好的错误信息
    const errorMsg = String(error);
    if (errorMsg.includes("timeout") || errorMsg.includes("超时")) {
      alert("连接设备超时，请确保设备在线并且能够访问");
    } else if (errorMsg.includes("设备测试失败")) {
      alert("无法连接到设备，请确保设备在线且地址正确");
    } else if (errorMsg.includes("保存设备列表失败")) {
      alert("设备连接成功但保存失败，请重试");
    } else {
      alert(`添加设备失败: ${errorMsg}`);
    }
  } finally {
    isLoading.value = false;
  }
}

// Remove a device
async function removeDevice(ip: string) {
  if (!App) {
    alert("无法连接到后端服务");
    return;
  }

  // 查找要移除的设备
  const deviceIndex = devices.value.findIndex((d) => d.ip === ip);
  if (deviceIndex === -1) {
    console.error(`找不到要删除的设备: ${ip}`);
    return;
  }

  // 标记为移除中
  devices.value[deviceIndex].status = "removing";

  try {
    console.log(`正在移除设备: ${ip}`);

    // 创建一个带超时的Promise
    const removeWithTimeout = Promise.race([
      App.RemoveDevice(ip),
      new Promise((_, reject) =>
        setTimeout(() => reject(new Error("设备移除操作超时")), 5000)
      ),
    ]);

    // 等待结果
    await removeWithTimeout;
    console.log(`设备 ${ip} 已成功移除`);

    // 直接从本地设备列表中移除，不重新加载
    devices.value = devices.value.filter((device) => device.ip !== ip);

    // 更新选择状态
    const newSelection = { ...selectedDevices.value };
    delete newSelection[ip];
    selectedDevices.value = newSelection;
  } catch (error) {
    console.error("移除设备出错:", error);

    // 尝试直接从前端移除设备，因为用户只想从列表移除设备
    console.log("从前端移除设备:", ip);
    devices.value = devices.value.filter((device) => device.ip !== ip);

    // 更新选择状态
    const newSelection = { ...selectedDevices.value };
    delete newSelection[ip];
    selectedDevices.value = newSelection;

    // 显示一个小提示，但不阻止操作完成
    console.warn(`注意: 设备可能未在后端完全移除: ${error}`);
  }
}

// Scan IP range for devices
async function scanDevices() {
  if (!App || !startIP.value || !endIP.value) return;

  scanLoading.value = true;
  try {
    await App.ScanIPRange(startIP.value, endIP.value);
    await loadDevices();
  } catch (error) {
    console.error("扫描设备出错:", error);
    alert(`扫描失败: ${error}`);
  } finally {
    scanLoading.value = false;
  }
}

// Handle file selection
async function handleFileSelect(event: Event) {
  if (!App) {
    alert("无法连接到后端服务");
    return;
  }

  const input = event.target as HTMLInputElement;
  if (input.files && input.files[0]) {
    // 获取文件信息
    const file = input.files[0];
    const filename = file.name;

    try {
      console.log("设置上传文件:", filename);

      // 告知用户我们正在寻找文件
      const fileStatus = document.getElementById("file-status");
      if (fileStatus) {
        fileStatus.textContent = "正在寻找文件...";
        fileStatus.className = "file-status searching";
      }

      // 调用后端设置上传文件
      const filePath = await App.SetUploadFile(filename);

      if (!filePath) {
        throw new Error("设置上传文件路径失败");
      }

      console.log("设置上传文件路径:", filePath);
      selectedFile.value = filename;

      // 更新文件状态UI
      if (fileStatus) {
        fileStatus.textContent = "文件已准备好";
        fileStatus.className = "file-status ready";
      }

      // 提供更详细的指导
      showFileHelp.value = true;
    } catch (error) {
      console.error("设置上传文件失败:", error);
      alert(`设置上传文件失败: ${error}`);
      selectedFile.value = "";

      // 更新文件状态UI
      const fileStatus = document.getElementById("file-status");
      if (fileStatus) {
        fileStatus.textContent = "文件未找到";
        fileStatus.className = "file-status error";
      }

      // 重置文件输入框
      input.value = "";
    }
  }
}

// 新增：处理MD5文件选择
async function handleMd5FileSelect(event: Event) {
  if (!App) {
    alert("无法连接到后端服务");
    return;
  }

  const input = event.target as HTMLInputElement;
  if (input.files && input.files[0]) {
    // 获取文件信息
    const file = input.files[0];
    const filename = file.name;

    try {
      console.log("设置MD5文件:", filename);

      // 告知用户我们正在寻找文件
      const fileStatus = document.getElementById("md5-file-status");
      if (fileStatus) {
        fileStatus.textContent = "正在寻找文件...";
        fileStatus.className = "file-status searching";
      }

      // 调用后端设置上传文件
      const filePath = await App.SetMd5File(filename);

      if (!filePath) {
        throw new Error("设置MD5文件路径失败");
      }

      console.log("设置MD5文件路径:", filePath);
      selectedMd5File.value = filename;

      // 更新文件状态UI
      if (fileStatus) {
        fileStatus.textContent = "MD5文件已准备好";
        fileStatus.className = "file-status ready";
      }
    } catch (error) {
      console.error("设置MD5文件失败:", error);
      alert(`设置MD5文件失败: ${error}`);
      selectedMd5File.value = "";

      // 更新文件状态UI
      const fileStatus = document.getElementById("md5-file-status");
      if (fileStatus) {
        fileStatus.textContent = "文件未找到";
        fileStatus.className = "file-status error";
      }

      // 重置文件输入框
      input.value = "";
    }
  }
}

// Update selected devices
async function updateSelectedDevices() {
  if (!App || !username.value || !password.value || !selectedFile.value) {
    alert("请填写所有字段并选择文件");
    return;
  }

  if (selectedDevicesList.value.length === 0) {
    alert("请至少选择一个设备进行更新");
    return;
  }

  isLoading.value = true;
  try {
    // 获取第一个选择设备的buildTime作为要更新的buildTime组
    const selectedBuildTime =
      selectedBuildTimes.value.length > 0 ? selectedBuildTimes.value[0] : "";

    // 更新请求中使用buildTime，而不是IP列表
    // 增加MD5文件参数
    updateResults.value = await App.UpdateDevices(
      username.value,
      password.value,
      selectedBuildTime,
      selectedMd5File.value
    );
  } catch (error: unknown) {
    console.error("更新设备出错:", error);
    const errorMessage = error instanceof Error ? error.message : String(error);
    alert(`更新失败: ${errorMessage}`);
  } finally {
    isLoading.value = false;
  }
}

// 修改按buildTime进行批量更新的函数
async function updateByBuildTime(buildTime: string) {
  if (!App || !username.value || !password.value || !selectedFile.value) {
    alert("请填写所有字段并选择文件");
    return;
  }

  isLoading.value = true;
  try {
    // 增加MD5文件参数
    updateResults.value = await App.UpdateDevices(
      username.value,
      password.value,
      buildTime,
      selectedMd5File.value
    );
  } catch (error: unknown) {
    console.error(`更新构建时间为 ${buildTime} 的设备出错:`, error);
    const errorMessage = error instanceof Error ? error.message : String(error);
    alert(`更新失败: ${errorMessage}`);
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <div class="container">
    <div class="header">
      <h1>设备更新管理 <span class="version">v1.1.5</span></h1>
    </div>

    <div class="tabs">
      <button
        :class="{ active: activeTab === 'devices' }"
        @click="activeTab = 'devices'"
      >
        设备管理
      </button>
      <button
        :class="{ active: activeTab === 'update' }"
        @click="activeTab = 'update'"
      >
        软件更新
      </button>
      <button
        :class="{ active: activeTab === 'camera' }"
        @click="activeTab = 'camera'"
      >
        摄像头配置
      </button>
      <button
        :class="{ active: activeTab === 'time' }"
        @click="activeTab = 'time'"
      >
        时间同步
      </button>
    </div>

    <!-- Device Management Tab -->
    <div v-if="activeTab === 'devices'" class="tab-content">
      <div class="card">
        <h2>添加设备</h2>
        <div class="form-group">
          <input
            v-model="newDeviceIP"
            placeholder="设备IP地址"
            @keyup.enter="addDevice"
          />
          <button @click="addDevice" :disabled="isLoading">添加设备</button>
        </div>
      </div>

      <div class="card">
        <h2>扫描IP范围</h2>
        <div class="form-group">
          <input v-model="startIP" placeholder="起始IP" />
          <input v-model="endIP" placeholder="结束IP" />
          <button @click="scanDevices" :disabled="scanLoading">
            {{ scanLoading ? "扫描中..." : "扫描" }}
          </button>
        </div>
      </div>

      <div class="card">
        <div class="header-with-action">
          <h2>设备列表</h2>
          <button
            @click="refreshDevices"
            :disabled="isLoading"
            class="refresh-button"
          >
            {{ isLoading ? "刷新中..." : "刷新" }}
          </button>
        </div>

        <div v-if="devices.length === 0" class="empty-state">
          未找到设备。请手动添加设备或扫描设备。
        </div>

        <table v-else class="device-table">
          <thead>
            <tr>
              <th>IP地址</th>
              <th>构建时间</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="device in paginatedDevices" :key="device.ip">
              <td>{{ device.ip }}</td>
              <td>{{ device.buildTime }}</td>
              <td>
                <span
                  :class="[
                    'status',
                    device.status === 'online'
                      ? 'status-online'
                      : device.status === 'removing'
                      ? 'status-removing'
                      : 'status-offline',
                  ]"
                >
                  {{
                    device.status === "online"
                      ? "在线"
                      : device.status === "removing"
                      ? "移除中..."
                      : "离线"
                  }}
                </span>
              </td>
              <td>
                <button
                  @click="removeDevice(device.ip)"
                  class="danger-button"
                  :disabled="device.status === 'removing'"
                >
                  {{ device.status === "removing" ? "移除中..." : "移除" }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>

        <div class="pagination">
          <button
            @click="goToPage(currentPage - 1)"
            :disabled="currentPage === 1"
          >
            上一页
          </button>
          <span>第 {{ currentPage }} 页，共 {{ totalPages }} 页</span>
          <button
            @click="goToPage(currentPage + 1)"
            :disabled="currentPage === totalPages"
          >
            下一页
          </button>
        </div>
      </div>
    </div>

    <!-- Update Devices Tab -->
    <div v-if="activeTab === 'update'" class="tab-content">
      <div class="card">
        <h2>更新设备</h2>

        <div class="form-group">
          <label>用户名</label>
          <input v-model="username" placeholder="用户名" />
        </div>

        <div class="form-group">
          <label>密码</label>
          <input v-model="password" type="password" placeholder="密码" />
        </div>

        <div class="form-group">
          <label>上传文件</label>
          <input type="file" @change="handleFileSelect" />
          <div v-if="selectedFile" class="selected-file">
            已选择: {{ selectedFile }}
            <div id="file-status" class="file-status"></div>
          </div>
          <div v-if="showFileHelp" class="file-help">
            <p><strong>文件处理说明:</strong></p>
            <p>系统将在以下位置查找您选择的文件:</p>
            <ul>
              <li>当前目录</li>
              <li>您的主目录 (~/{{ selectedFile }})</li>
              <li>您的下载文件夹 (~/Downloads/{{ selectedFile }})</li>
            </ul>
            <p>如果找到文件，将会自动复制到应用程序的临时目录中进行处理。</p>
            <p>
              如果您遇到"文件未找到"的错误，请确保文件位于上述位置之一，或重新选择文件。
            </p>
          </div>
        </div>

        <!-- 新增：MD5文件上传选项 -->
        <div class="form-group">
          <label>MD5文件（可选）</label>
          <input type="file" @change="handleMd5FileSelect" />
          <div v-if="selectedMd5File" class="selected-file">
            已选择: {{ selectedMd5File }}
            <div id="md5-file-status" class="file-status"></div>
          </div>
          <div v-if="selectedMd5File" class="file-help">
            <p><strong>MD5文件说明:</strong></p>
            <p>上传的MD5文件将用于验证设备上传过程的完整性。</p>
            <p>文件将以"md5file"的名称作为表单字段上传到设备。</p>
          </div>
        </div>

        <div class="card">
          <div class="header-with-action">
            <h3>选择要更新的设备</h3>
            <div class="actions-container">
              <div class="select-all-toggle">
                <label>
                  <input
                    type="checkbox"
                    :checked="selectAll"
                    @change="toggleSelectAll"
                  />
                  全选/全不选
                </label>
                <span v-if="devices.length > 0" class="selection-count">
                  已选择: {{ selectedDevicesCount }}/{{ devices.length }}
                </span>
              </div>

              <div class="filter-container">
                <input
                  v-model="searchQuery"
                  placeholder="搜索设备"
                  class="search-input"
                />
                <select v-model="filterStatus" class="filter-select">
                  <option value="all">所有状态</option>
                  <option value="online">在线</option>
                  <option value="offline">离线</option>
                </select>
              </div>
            </div>
          </div>

          <div v-if="devices.length === 0" class="empty-state">
            未找到设备。请先添加设备。
          </div>

          <div v-else class="device-groups">
            <div
              v-for="(devices, groupKey) in groupedDevices"
              :key="groupKey"
              class="device-group"
            >
              <div class="group-header">
                <label class="group-checkbox-label">
                  <input
                    type="checkbox"
                    :checked="isGroupAllSelected(groupKey)"
                    :indeterminate="isGroupPartiallySelected(groupKey)"
                    @change="
                      toggleGroup(groupKey, !isGroupAllSelected(groupKey))
                    "
                  />
                  <span class="group-title">
                    <span v-if="groupKey === '未知版本'">未知版本</span>
                    <span v-else>
                      构建时间: <span class="build-time">{{ groupKey }}</span>
                    </span>
                    <span class="device-count"
                      >({{ groupedDevices[groupKey].length }}台)</span
                    >
                  </span>
                </label>
                <div class="group-actions">
                  <button
                    v-if="
                      groupKey !== '未知版本' &&
                      groupedDevices[groupKey].some(
                        (d) => d.status === 'online'
                      )
                    "
                    @click="updateByBuildTime(groupKey)"
                    :disabled="
                      isLoading || !username || !password || !selectedFile
                    "
                    class="update-group-button"
                  >
                    {{ isLoading ? "更新中..." : "更新此版本" }}
                  </button>
                  <button
                    @click="groupExpanded[groupKey] = !groupExpanded[groupKey]"
                    class="group-toggle-button"
                  >
                    {{ groupExpanded[groupKey] ? "收起" : "展开" }}
                  </button>
                </div>
              </div>

              <div v-if="groupExpanded[groupKey]" class="device-selection-list">
                <div
                  v-for="device in devices"
                  :key="device.ip"
                  class="device-selection-item"
                  :class="{
                    'device-offline': device.status !== 'online',
                    'device-removing': device.status === 'removing',
                  }"
                >
                  <label class="checkbox-label">
                    <input
                      type="checkbox"
                      v-model="selectedDevices[device.ip]"
                      :disabled="device.status !== 'online'"
                    />
                    <span class="device-info">
                      <span class="device-ip">{{ device.ip }}</span>
                      <span
                        class="device-build-time"
                        v-if="device.buildTime && device.buildTime !== groupKey"
                      >
                        {{ device.buildTime }}
                      </span>
                      <span
                        :class="[
                          'status-dot',
                          device.status === 'online'
                            ? 'status-online'
                            : device.status === 'removing'
                            ? 'status-removing'
                            : 'status-offline',
                        ]"
                        :title="
                          device.status === 'online'
                            ? '在线'
                            : device.status === 'removing'
                            ? '移除中'
                            : '离线'
                        "
                      ></span>
                    </span>
                  </label>
                </div>
              </div>
            </div>
          </div>

          <!-- 分页 -->
          <div v-if="Object.keys(groupedDevices).length > 5" class="pagination">
            <button
              @click="goToPage(currentPage - 1)"
              :disabled="currentPage === 1"
            >
              上一页
            </button>
            <span>第 {{ currentPage }} 页，共 {{ totalPages }} 页</span>
            <button
              @click="goToPage(currentPage + 1)"
              :disabled="currentPage === totalPages"
            >
              下一页
            </button>
          </div>
        </div>

        <button
          @click="updateSelectedDevices"
          :disabled="
            isLoading ||
            !username ||
            !password ||
            !selectedFile ||
            selectedDevicesList.length === 0
          "
          class="primary-button"
        >
          {{
            isLoading ? "更新中..." : `更新选中的设备 (${selectedDevicesCount})`
          }}
        </button>
      </div>

      <div v-if="updateResults.length > 0" class="card">
        <h2>更新结果</h2>
        <table class="device-table">
          <thead>
            <tr>
              <th>IP地址</th>
              <th>状态</th>
              <th>消息</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="result in updateResults" :key="result.ip">
              <td>{{ result.ip }}</td>
              <td>
                <span
                  :class="[
                    'status',
                    result.success ? 'status-online' : 'status-offline',
                  ]"
                >
                  {{ result.success ? "成功" : "失败" }}
                </span>
              </td>
              <td>{{ result.message }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Camera Configuration Tab -->
    <div v-if="activeTab === 'camera'" class="tab-content">
      <CameraConfig />
    </div>

    <!-- Time Sync Tab -->
    <div v-if="activeTab === 'time'" class="tab-content">
      <TimeSync />
    </div>
  </div>
</template>

<style>
:root {
  --primary-color: #4361ee;
  --primary-hover: #3a56d4;
  --danger-color: #ef476f;
  --danger-hover: #d63d63;
  --success-color: #06d6a0;
  --warning-color: #ffd166;
  --background-color: #f8f9fa;
  --card-background: white;
  --text-color: #212529;
  --border-color: #dee2e6;
}

body {
  font-family: "Segoe UI", Tahoma, Geneva, Verdana, sans-serif;
  background-color: var(--background-color);
  color: var(--text-color);
  margin: 0;
  padding: 0;
}

.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

h1 {
  color: var(--primary-color);
  margin-bottom: 20px;
  text-align: center;
}

h2 {
  color: var(--text-color);
  margin-top: 0;
}

h3 {
  color: var(--text-color);
  margin-top: 0;
  font-size: 16px;
}
.version {
  font-size: 16px;
  color: #6c757d;
}
.card {
  background-color: var(--card-background);
  border-radius: 8px;
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.1);
  padding: 20px;
  margin-bottom: 20px;
}

.form-group {
  margin-bottom: 15px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.form-group label {
  display: block;
  margin-bottom: 5px;
  font-weight: 500;
  width: 100%;
}

input {
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 14px;
  flex: 1;
}

button {
  padding: 8px 16px;
  background-color: var(--primary-color);
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

button:hover:not(:disabled) {
  background-color: var(--primary-hover);
}

button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.danger-button {
  background-color: var(--danger-color);
}

.danger-button:hover:not(:disabled) {
  background-color: var(--danger-hover);
}

.primary-button {
  background-color: var(--primary-color);
  padding: 10px 20px;
  font-size: 16px;
  width: 100%;
  margin-top: 10px;
}

.tabs {
  display: flex;
  margin-bottom: 20px;
  border-bottom: 1px solid var(--border-color);
}

.tabs button {
  padding: 10px 20px;
  background-color: transparent;
  color: var(--text-color);
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  font-size: 16px;
  transition: all 0.2s;
  margin-right: 10px;
}

.tabs button.active {
  color: var(--primary-color);
  border-bottom: 2px solid var(--primary-color);
}

.tab-content {
  margin-top: 20px;
}

.device-table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 10px;
}

.device-table th,
.device-table td {
  padding: 12px;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}

.device-table th {
  background-color: #f8f9fa;
  font-weight: 600;
}

.header-with-action {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
  flex-wrap: wrap;
}

.refresh-button {
  padding: 6px 12px;
  font-size: 14px;
}

.status {
  display: inline-block;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.status-online {
  background-color: var(--success-color);
  color: white;
}

.status-removing {
  background-color: var(--warning-color);
  color: #333;
  animation: pulse 1.5s infinite;
}

.status-offline {
  background-color: var(--danger-color);
  color: white;
}

@keyframes pulse {
  0% {
    opacity: 0.7;
  }
  50% {
    opacity: 1;
  }
  100% {
    opacity: 0.7;
  }
}

.empty-state {
  text-align: center;
  padding: 20px;
  color: #6c757d;
}

.selected-file {
  margin-top: 5px;
  font-size: 14px;
  color: #6c757d;
  word-break: break-all;
}

/* 设备选择相关的样式 */
.device-selection-card {
  max-height: 400px;
  overflow-y: auto;
  margin-top: 15px;
  padding: 15px;
}

.device-selection-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 10px;
  margin-top: 8px;
  padding-left: 25px;
}

.device-selection-item {
  padding: 8px;
  border-radius: 4px;
  border: 1px solid var(--border-color);
  transition: background-color 0.2s;
}

.device-selection-item:hover {
  background-color: #f8f9fa;
}

.device-offline {
  opacity: 0.6;
}

.device-removing {
  opacity: 0.6;
  background-color: #fff9db;
  animation: pulse-bg 1.5s infinite;
}

@keyframes pulse-bg {
  0% {
    background-color: #fff9db;
  }
  50% {
    background-color: #ffec99;
  }
  100% {
    background-color: #fff9db;
  }
}

.checkbox-label {
  display: flex;
  align-items: center;
  cursor: pointer;
  width: 100%;
}

.checkbox-label input[type="checkbox"] {
  margin-right: 8px;
  flex: none;
}

.device-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.device-ip {
  font-weight: 500;
  margin-right: auto;
}

.device-build-time {
  font-size: 11px;
  color: #666;
  margin: 0 8px;
  background-color: #f0f0f0;
  padding: 2px 6px;
  border-radius: 3px;
}

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  display: inline-block;
  margin-left: 8px;
}

.select-all-toggle {
  display: flex;
  align-items: center;
  font-size: 14px;
  margin-bottom: 10px;
}

.select-all-toggle label {
  margin-right: 15px;
  display: flex;
  align-items: center;
  cursor: pointer;
}

.select-all-toggle input[type="checkbox"] {
  margin-right: 5px;
}

.selection-count {
  font-size: 12px;
  color: #6c757d;
  font-weight: 500;
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  margin-top: 15px;
  gap: 10px;
}

.pagination button {
  padding: 6px 12px;
  font-size: 14px;
}

.pagination span {
  font-size: 14px;
  color: #555;
}

/* 新增样式 - 设备组和搜索/筛选 */
.device-groups {
  max-height: 500px;
  overflow-y: auto;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  padding: 5px;
}

.device-group {
  margin-bottom: 10px;
  border: 1px solid #eee;
  border-radius: 4px;
  overflow: hidden;
}

.group-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background-color: #f5f5f5;
  border-bottom: 1px solid #eee;
}

.group-checkbox-label {
  display: flex;
  align-items: center;
  font-weight: 500;
  cursor: pointer;
}

.group-checkbox-label input[type="checkbox"] {
  margin-right: 8px;
}

.group-title {
  font-size: 14px;
  display: flex;
  align-items: center;
}

.group-title .build-time {
  font-weight: 600;
  color: var(--primary-color);
}

.group-toggle-button {
  padding: 4px 8px;
  font-size: 12px;
  background-color: transparent;
  color: #555;
  border: 1px solid #ccc;
}

.group-toggle-button:hover {
  background-color: #eee;
}

.actions-container {
  display: flex;
  justify-content: space-between;
  width: 100%;
  margin-top: 10px;
  flex-wrap: wrap;
  gap: 10px;
}

.filter-container {
  display: flex;
  align-items: center;
  gap: 8px;
}

.search-input {
  width: 200px;
  padding: 6px 10px;
  font-size: 13px;
}

.filter-select {
  padding: 6px 10px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background-color: white;
  font-size: 13px;
}

@media (max-width: 768px) {
  .header-with-action {
    flex-direction: column;
    align-items: flex-start;
  }

  .actions-container {
    flex-direction: column;
    width: 100%;
  }

  .filter-container {
    width: 100%;
  }

  .search-input {
    width: 100%;
  }

  .device-selection-list {
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  }
}

/* 文件状态样式 */
.file-status {
  display: inline-block;
  margin-left: 10px;
  padding: 3px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.file-status.searching {
  background-color: #ffedd5;
  color: #9a3412;
  animation: pulse 1.5s infinite;
}

.file-status.ready {
  background-color: #dcfce7;
  color: #166534;
}

.file-status.error {
  background-color: #fee2e2;
  color: #b91c1c;
}

.file-help {
  margin-top: 15px;
  padding: 10px 15px;
  background-color: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  font-size: 14px;
}

.file-help p {
  margin: 5px 0;
}

.file-help ul {
  margin: 5px 0;
  padding-left: 20px;
}

.file-help strong {
  color: #0f766e;
}

.group-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.update-group-button {
  padding: 4px 8px;
  font-size: 12px;
  background-color: var(--primary-color);
  color: white;
}

.update-group-button:hover:not(:disabled) {
  background-color: var(--primary-hover);
}

.update-group-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.device-count {
  font-size: 12px;
  color: #666;
  margin-left: 4px;
}
</style>
