// @ts-nocheck
<script lang="ts" setup>
import { ref, onMounted, computed, watch } from "vue";
import CameraConfig from "./components/CameraConfig.vue";
import TimeSync from "./components/TimeSync.vue";
import DeviceBackup from "./components/DeviceBackup.vue";

// 导入Wails生成的绑定
import * as backend from "../wailsjs/wailsjs/go/main/App";

// 定义后端API类型，避免TypeScript错误
type BackendAPI = typeof backend;

// 声明全局window.go属性
declare global {
  interface Window {
    go?: {
      main?: {
        App?: any;
      };
    };
  }
}

// 可用后端引用
let wailsBackend: any = backend;

// 状态变量声明
const appInitialized = ref(false);
const connectionError = ref(false);

// 明确定义设备类型
interface Device {
  id: string; // 设备唯一标识
  ip: string; // 设备IP地址
  buildTime: string; // 构建时间
  status: string; // 状态：online或offline
  region?: string; // 可选的区域标识
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

// 区域管理相关变量
const regions = ref<string[]>([]);
const currentRegion = ref<string>("");
const newRegion = ref<string>("");
const customRegion = ref<boolean>(false);
const regionLoading = ref<boolean>(false);

// 通知状态
interface Notification {
  show: boolean;
  message: string;
  type: "info" | "success" | "error" | "warning";
}

// 确认对话框状态
interface ConfirmDialog {
  show: boolean;
  title: string;
  message: string;
  onConfirm: () => void;
  onCancel: () => void;
}

// 添加通知和确认对话框的状态变量
const notification = ref<Notification>({
  show: false,
  message: "",
  type: "info",
});

const confirmDialog = ref<ConfirmDialog>({
  show: false,
  title: "",
  message: "",
  onConfirm: () => {},
  onCancel: () => {},
});

// 显示通知的函数
const showNotification = (
  message: string,
  type: "info" | "success" | "error" | "warning" = "info"
) => {
  notification.value = {
    show: true,
    message,
    type,
  };

  // 5秒后自动关闭通知
  setTimeout(() => {
    notification.value.show = false;
  }, 5000);
};

// 显示确认对话框的函数
const showConfirmDialog = (
  title: string,
  message: string
): Promise<boolean> => {
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
      },
    };
  });
};

// 计算属性：已选择的设备ID列表
const selectedDevicesList = computed(() => {
  return Object.entries(selectedDevices.value)
    .filter(([id, selected]) => {
      if (!selected) return false;
      const device = devices.value.find((d) => d.id === id);
      return device && device.status === "online";
    })
    .map(([id]) => id);
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

  // 按区域筛选
  if (currentRegion.value) {
    // 当选择了区域时，显示该区域的设备和无区域的设备
    result = result.filter(
      (device) => device.region === currentRegion.value || !device.region
    );
  }

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

// 计算无区域设备的数量
const devicesWithoutRegion = computed(() => {
  return devices.value.filter((device) => !device.region).length;
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
    // 只允许选择在线设备
    if (device.status === "online") {
      newSelection[device.id] = selected;
    }
  });

  selectedDevices.value = newSelection;
}

// 切换单个设备选择状态
function toggleDeviceSelection(device: Device, buildTime: string) {
  // 只允许切换在线设备的状态
  if (device.status === "online") {
    const newSelection = { ...selectedDevices.value };
    newSelection[device.id] = !newSelection[device.id];
    selectedDevices.value = newSelection;
  }
}

// 检查一个组是否全部选中
function isGroupAllSelected(groupKey: string): boolean {
  const group = groupedDevices.value[groupKey] || [];
  const onlineDevices = group.filter((device) => device.status === "online");

  if (onlineDevices.length === 0) return false;

  return onlineDevices.every((device) => selectedDevices.value[device.id]);
}

// 检查一个组是否部分选中
function isGroupPartiallySelected(groupKey: string): boolean {
  const group = groupedDevices.value[groupKey] || [];
  const onlineDevices = group.filter((device) => device.status === "online");

  if (onlineDevices.length === 0) return false;

  const selectedCount = onlineDevices.filter(
    (device) => selectedDevices.value[device.id]
  ).length;
  return selectedCount > 0 && selectedCount < onlineDevices.length;
}

// 更新选中的buildTime列表计算属性
const selectedBuildTimes = computed(() => {
  const buildTimes = new Set<string>();

  Object.entries(selectedDevices.value)
    .filter(([_, selected]) => selected)
    .forEach(([id]) => {
      const device = devices.value.find((d) => d.id === id);
      if (device && device.buildTime) {
        buildTimes.add(device.buildTime);
      }
    });

  return Array.from(buildTimes);
});

// Setup on mount
onMounted(async () => {
  console.log("APP VUE: onMounted执行，开始初始化...");
  try {
    // 检查导入的backend是否可用
    if (typeof backend.GetDevices === "function") {
      console.log("APP.vue: 成功使用导入的方式初始化后端绑定");
      wailsBackend = backend;
    }
    // 尝试使用window.go（可能在完全加载后才会存在）
    else if (window.go?.main?.App) {
      console.log("APP.vue: 成功使用window.go初始化后端绑定");
      wailsBackend = window.go.main.App;
    }
    // 最后尝试延迟等待window.go初始化
    else {
      console.log("APP.vue: 等待window.go初始化...");
      // 等待500ms尝试重新检查window.go是否存在
      await new Promise((resolve) => setTimeout(resolve, 500));

      if (window.go?.main?.App) {
        console.log("APP.vue: 成功使用延迟初始化的window.go绑定");
        wailsBackend = window.go.main.App;
      } else {
        throw new Error("找不到Wails后端绑定");
      }
    }

    console.log("后端API已连接，开始加载数据...");

    // 初始化完成，加载设备列表
    appInitialized.value = true;
    connectionError.value = false;
    await loadDevices();
  } catch (error) {
    console.error("初始化应用失败:", error);
    connectionError.value = true;
    showNotification("无法连接到后端服务，请重启应用", "error");
  }
});

// Load devices from backend
async function loadDevices() {
  isLoading.value = true;
  try {
    devices.value = await wailsBackend.GetDevices();

    // 获取当前应用的区域过滤
    currentRegion.value = await wailsBackend.GetCurrentRegion();

    // 加载所有区域
    await loadRegions();

    // 更新设备选择状态
    updateDeviceSelection();
  } catch (error) {
    console.error("加载设备出错:", error);
    connectionError.value = true;
    showNotification("无法连接到后端服务，请重启应用", "error");
  } finally {
    isLoading.value = false;
  }
}

// 加载区域列表
async function loadRegions() {
  regionLoading.value = true;
  try {
    regions.value = await wailsBackend.GetRegions();
    console.log("加载区域列表成功:", regions.value);
  } catch (error) {
    console.error("加载区域列表失败:", error);
  } finally {
    regionLoading.value = false;
  }
}

// 切换自定义区域输入
function toggleCustomRegion() {
  customRegion.value = !customRegion.value;

  if (customRegion.value) {
    // 切换到自定义区域输入模式
    newRegion.value = currentRegion.value;
  } else {
    // 从自定义区域切换回选择模式
    if (newRegion.value) {
      currentRegion.value = newRegion.value;
      newRegion.value = "";
      // 应用区域过滤
      applyRegionFilter();
    }

    // 重新从后端加载区域列表，确保包含所有新添加的区域
    loadRegions();
  }
}

// 添加新的函数处理自定义区域回车事件
async function applyCustomRegion() {
  if (!newRegion.value) {
    showNotification("请输入区域名称", "warning");
    return;
  }

  try {
    // 调用后端设置区域过滤
    devices.value = await wailsBackend.SetRegionFilter(newRegion.value);

    // 更新当前区域值，保持UI状态一致
    currentRegion.value = newRegion.value;

    // 更新设备选择状态
    updateDeviceSelection();

    showNotification(`已过滤显示区域 "${newRegion.value}" 的设备`, "info");
  } catch (error) {
    console.error("设置区域过滤失败:", error);
    showNotification(`设置区域过滤失败: ${error}`, "error");
  }
}

// 应用区域筛选
async function applyRegionFilter() {
  const region = customRegion.value ? newRegion.value : currentRegion.value;

  try {
    // 调用后端设置区域过滤
    devices.value = await wailsBackend.SetRegionFilter(region);

    // 更新设备选择状态
    updateDeviceSelection();

    if (region) {
      showNotification(`已过滤显示区域 "${region}" 的设备`, "info");
    } else {
      showNotification("显示全部设备", "info");
    }
  } catch (error) {
    console.error("设置区域过滤失败:", error);
    showNotification(`设置区域过滤失败: ${error}`, "error");
  }
}

// 监听区域变化
watch(currentRegion, (newValue) => {
  if (!customRegion.value) {
    applyRegionFilter();
  }
});

// 修改刷新设备函数
async function refreshDevices() {
  isLoading.value = true;
  try {
    console.log("开始刷新设备列表...");

    // 设置请求超时
    const refreshPromise = wailsBackend.RefreshDevices();
    const timeoutPromise = new Promise((_, reject) =>
      setTimeout(() => reject(new Error("刷新设备超时，请检查网络连接")), 30000)
    );

    // 使用 Promise.race 来处理超时
    devices.value = await Promise.race([refreshPromise, timeoutPromise]);

    console.log("设备列表刷新成功，获取到", devices.value.length, "个设备");

    // 更新设备选择状态
    updateDeviceSelection();

    showNotification("设备列表已刷新", "success");
  } catch (error) {
    console.error("刷新设备出错:", error);

    try {
      // 尝试重新获取设备列表，即使刷新失败
      console.log("尝试重新获取设备列表...");
      devices.value = await wailsBackend.GetDevices();
      updateDeviceSelection();
    } catch (secondError) {
      console.error("获取设备列表也失败:", secondError);
      connectionError.value = true;
    }

    showNotification(`刷新设备失败: ${error}`, "error");
  } finally {
    console.log("设备刷新过程结束");
    isLoading.value = false;
  }
}

// Add a new device
async function addDevice() {
  if (!newDeviceIP.value) {
    showNotification("请输入设备IP地址", "warning");
    return;
  }

  // 简单的IP地址格式验证
  const ipPattern = /^(\d{1,3}\.){3}\d{1,3}$/;
  if (!ipPattern.test(newDeviceIP.value)) {
    showNotification("请输入有效的IP地址格式，例如: 192.168.1.100", "warning");
    return;
  }

  isLoading.value = true;
  try {
    console.log("开始添加设备:", newDeviceIP.value);

    // 获取要应用的区域
    const regionToApply = customRegion.value
      ? newRegion.value
      : currentRegion.value;

    // 设置超时，防止长时间阻塞
    const addDevicePromise = wailsBackend.AddDevice(
      newDeviceIP.value,
      regionToApply || ""
    );
    const timeoutPromise = new Promise((_, reject) =>
      setTimeout(
        () => reject(new Error("添加设备超时，请检查设备是否在线")),
        10000
      )
    );

    const device = await Promise.race([addDevicePromise, timeoutPromise]);
    console.log("设备添加成功，正在刷新设备列表");

    // 由于区域已在AddDevice中设置，不再需要单独设置区域
    showNotification(
      regionToApply
        ? `设备已添加并分配到区域: ${regionToApply}`
        : "设备添加成功",
      "success"
    );

    // 清空输入框
    newDeviceIP.value = "";
    await loadDevices();
  } catch (error) {
    console.error("添加设备失败:", error);
    showNotification(`添加设备失败: ${error}`, "error");
  } finally {
    isLoading.value = false;
  }
}

// Remove a device
async function removeDevice(device: Device) {
  if (!wailsBackend || typeof wailsBackend.RemoveDevice !== "function") {
    alert("无法连接到后端服务");
    return;
  }

  // 查找要移除的设备
  const deviceIndex = devices.value.findIndex((d) => d.id === device.id);
  if (deviceIndex === -1) {
    console.error(`找不到要删除的设备: ${device.ip}`);
    return;
  }

  // 标记为移除中
  devices.value[deviceIndex].status = "removing";

  try {
    console.log(`正在移除设备: ${device.ip} (ID: ${device.id})`);

    // 创建一个带超时的Promise
    const removeWithTimeout = Promise.race([
      wailsBackend.RemoveDevice(device.id),
      new Promise((_, reject) =>
        setTimeout(() => reject(new Error("设备移除操作超时")), 5000)
      ),
    ]);

    // 等待结果
    await removeWithTimeout;
    console.log(`设备 ${device.ip} 已成功移除`);

    // 直接从本地设备列表中移除，不重新加载
    devices.value = devices.value.filter((d) => d.id !== device.id);

    // 更新选择状态
    const newSelection = { ...selectedDevices.value };
    delete newSelection[device.id];
    selectedDevices.value = newSelection;
  } catch (error) {
    console.error("移除设备出错:", error);

    // 尝试直接从前端移除设备，因为用户只想从列表移除设备
    console.log("从前端移除设备:", device.ip);
    devices.value = devices.value.filter((d) => d.id !== device.id);

    // 更新选择状态
    const newSelection = { ...selectedDevices.value };
    delete newSelection[device.id];
    selectedDevices.value = newSelection;

    // 显示一个小提示，但不阻止操作完成
    console.warn(`注意: 设备可能未在后端完全移除: ${error}`);
  }
}

// Scan IP range for devices
async function scanDevices() {
  if (!startIP.value || !endIP.value) {
    showNotification("请输入起始IP和结束IP", "warning");
    return;
  }

  const ipPattern = /^(\d{1,3}\.){3}\d{1,3}$/;
  if (!ipPattern.test(startIP.value) || !ipPattern.test(endIP.value)) {
    showNotification("请输入有效的IP地址格式", "warning");
    return;
  }

  scanLoading.value = true;
  try {
    // 不再传递用户名和密码
    const scannedDevices = await wailsBackend.ScanIPRange(
      startIP.value,
      endIP.value
    );
    console.log("扫描完成，发现设备:", scannedDevices);

    if (scannedDevices && scannedDevices.length > 0) {
      // 如果选择了区域，设置扫描到的设备的区域
      const regionToApply = customRegion.value
        ? newRegion.value
        : currentRegion.value;

      if (regionToApply) {
        const deviceIPs = scannedDevices.map((device: any) => device.ip);
        try {
          await wailsBackend.SetDevicesRegion(deviceIPs, regionToApply);
          showNotification(
            `扫描完成，发现 ${scannedDevices.length} 台设备并分配到区域: ${regionToApply}`,
            "success"
          );
        } catch (err) {
          console.error("设置设备区域失败:", err);
          showNotification(
            `扫描完成，发现 ${scannedDevices.length} 台设备，但未能设置区域: ${err}`,
            "warning"
          );
        }
      } else {
        showNotification(
          `扫描完成，发现 ${scannedDevices.length} 台设备`,
          "success"
        );
      }
    } else {
      showNotification("扫描完成，未发现设备", "info");
    }

    await loadDevices();
  } catch (error) {
    console.error("扫描设备失败:", error);
    showNotification(`扫描设备失败: ${error}`, "error");
  } finally {
    scanLoading.value = false;
  }
}

// Handle file selection
async function handleFileSelect(event: Event) {
  if (!wailsBackend || typeof wailsBackend.SetUploadFile !== "function") {
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
      const filePath = await wailsBackend.SetUploadFile(filename);

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
  const input = event.target as HTMLInputElement;
  if (!input.files || input.files.length === 0) {
    return;
  }

  const filename = input.files[0].name;

  if (!filename.endsWith(".md5")) {
    alert("请选择.md5格式的文件");
    input.value = "";
    return;
  }

  // 检查Wails后端是否可用
  if (!wailsBackend || typeof wailsBackend.SetMd5File !== "function") {
    alert("后端服务不可用");
    return;
  }

  try {
    console.log("设置MD5文件:", filename);

    // 告知用户我们正在寻找文件
    const fileStatus = document.getElementById("md5-file-status");
    if (fileStatus) {
      fileStatus.textContent = "正在寻找文件...";
      fileStatus.className = "file-status searching";
    }

    // SetMd5File不存在的情况下，简单地使用文件名
    // const filePath = await backend.SetMd5File(filename);
    const filePath = filename;

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

// Update selected devices
async function updateSelectedDevices() {
  // 检查必要条件
  if (!selectedFile.value) {
    showNotification("请选择更新文件", "warning");
    return;
  }

  if (!username.value || !password.value) {
    showNotification("请输入用户名和密码", "warning");
    return;
  }

  // 检查选中的设备
  const selectedDevices = selectedDevicesList.value;
  if (selectedDevices.length === 0) {
    showNotification("请至少选择一个在线设备进行更新", "warning");
    return;
  }

  try {
    isLoading.value = true;
    showNotification("正在更新设备...", "info");

    // 调用后端的更新方法
    const results = await wailsBackend.UpdateDevicesFile(
      selectedFile.value,
      0 // 更新所有选中的设备
    );

    // 处理结果并显示
    updateResults.value = results || [];
    processUpdateResults(results);
  } catch (error) {
    console.error("更新设备失败:", error);
    showNotification(`更新失败: ${error}`, "error");
  } finally {
    isLoading.value = false;
  }
}

// 根据BUILD_TIME更新设备
async function updateByBuildTime(buildTime) {
  if (!selectedFile.value) {
    showNotification("请选择更新文件", "error");
    return;
  }

  if (!username.value || !password.value) {
    showNotification("请输入用户名和密码", "error");
    return;
  }

  try {
    isLoading.value = true;
    showNotification(`正在更新构建时间为 ${buildTime} 之前的设备...`, "info");

    // 将buildTime转换为数值
    const selectedBuildTime = parseInt(buildTime, 10);

    // 调用后端的更新方法
    const results = await backend.UpdateDevicesFile(
      selectedFile.value,
      selectedBuildTime
    );

    // 处理结果并显示
    updateResults.value = results || [];
    processUpdateResults(results);
  } catch (error) {
    console.error("更新设备失败:", error);
    showNotification(`更新失败: ${error}`, "error");
  } finally {
    isLoading.value = false;
  }
}

// 处理更新结果
function processUpdateResults(results) {
  if (results && Array.isArray(results)) {
    const successCount = results.filter((r) => r.success).length;
    const totalCount = results.length;

    if (successCount === totalCount) {
      showNotification(`成功更新 ${successCount} 台设备`, "success");
    } else {
      showNotification(
        `更新完成: ${successCount}/${totalCount} 台设备成功`,
        "warning"
      );
    }

    // 显示详细结果
    console.log("更新结果:", results);
  } else {
    showNotification("更新失败: 无法获取更新结果", "error");
  }
}

// 批量更新所有设备
async function updateAllDevices() {
  if (!username.value || !password.value || !selectedFile.value) {
    showNotification("请填写所有字段并选择文件", "warning");
    return;
  }

  isLoading.value = true;
  try {
    // 使用新的后端方法
    if (checkBackendFunction("UpdateDevicesFile")) {
      // 调用UpdateDevicesFile，传入0表示更新所有设备
      const results = await backend.UpdateDevicesFile(selectedFile.value, 0);
      updateResults.value = results || [];
      processUpdateResults(results);
    } else if (checkBackendFunction("UpdateDevices")) {
      // 如果新方法不可用，回退到旧方法
      updateResults.value = await wailsBackend.UpdateDevices(
        username.value,
        password.value,
        "", // 空字符串表示更新所有设备
        selectedMd5File.value || ""
      );
      showNotification("所有设备的更新任务已提交", "success");
    } else {
      showNotification("更新功能暂时不可用，请联系开发人员", "warning");
      return;
    }
  } catch (error) {
    console.error("更新设备失败:", error);
    showNotification(`更新设备失败: ${error}`, "error");
  } finally {
    isLoading.value = false;
  }
}

// 应用区域到当前选中的设备
async function applyRegion() {
  const regionToApply = customRegion.value
    ? newRegion.value
    : currentRegion.value;

  if (!regionToApply) {
    showNotification("请选择或输入区域名称", "warning");
    return;
  }

  const selectedIDs = selectedDevicesList.value;
  if (selectedIDs.length === 0) {
    showNotification("请选择要设置区域的设备", "warning");
    return;
  }

  // 收集选中设备的信息以便显示
  const selectedDevicesInfo = selectedIDs
    .map((id) => devices.value.find((d) => d.id === id))
    .filter((device) => device !== undefined);

  // 检查是否有设备已经设置了区域
  const devicesWithRegion = selectedDevicesInfo.filter(
    (device) => device && device.region && device.region !== regionToApply
  );

  let confirmMessage = `确定要将 ${selectedIDs.length} 台设备设置为区域: ${regionToApply}?`;

  // 如果有设备已经有区域，添加警告信息
  if (devicesWithRegion.length > 0) {
    confirmMessage += `\n\n注意: 其中 ${devicesWithRegion.length} 台设备已有区域设置，将被覆盖:`;
    devicesWithRegion.forEach((device) => {
      if (device) {
        confirmMessage += `\n- ${device.ip}: ${device.region} → ${regionToApply}`;
      }
    });
  }

  // 使用自定义确认对话框
  const confirmed = await showConfirmDialog("确认设置区域", confirmMessage);
  if (!confirmed) {
    return;
  }

  try {
    await wailsBackend.SetDevicesRegion(selectedIDs, regionToApply);
    showNotification(
      `已成功将 ${selectedIDs.length} 台设备设置为区域: ${regionToApply}`,
      "success"
    );
    await loadDevices(); // 重新加载设备列表
  } catch (error) {
    console.error("设置设备区域失败:", error);
    showNotification(`设置设备区域失败: ${error}`, "error");
  }
}

// 应用区域到所有没有区域的设备
async function applyRegionToDevicesWithoutRegion() {
  const regionToApply = customRegion.value
    ? newRegion.value
    : currentRegion.value;

  if (!regionToApply) {
    showNotification("请选择或输入区域名称", "warning");
    return;
  }

  // 筛选出没有区域的设备
  const devicesWithoutRegion = devices.value.filter((device) => !device.region);
  if (devicesWithoutRegion.length === 0) {
    showNotification("没有找到无区域的设备", "warning");
    return;
  }

  // 提取设备ID列表
  const devicesWithoutRegionIDs = devicesWithoutRegion.map(
    (device) => device.id
  );

  // 使用自定义确认对话框
  const confirmMessage = `确定要将 ${devicesWithoutRegion.length} 台无区域设备设置为区域: ${regionToApply}?`;
  const confirmed = await showConfirmDialog("确认设置区域", confirmMessage);
  if (!confirmed) {
    return;
  }

  try {
    await wailsBackend.SetDevicesRegion(devicesWithoutRegionIDs, regionToApply);
    showNotification(
      `已成功将 ${devicesWithoutRegion.length} 台无区域设备设置为区域: ${regionToApply}`,
      "success"
    );
    await loadDevices(); // 重新加载设备列表
  } catch (error) {
    console.error("设置设备区域失败:", error);
    showNotification(`设置设备区域失败: ${error}`, "error");
  }
}

// 保存设备列表
async function saveDeviceList() {
  try {
    await wailsBackend.SaveDevices();
    alert("设备列表保存成功");
  } catch (error) {
    console.error("保存设备列表失败:", error);
    alert(`保存设备列表失败: ${error}`);
  }
}

// 清空设备列表
async function clearDeviceList() {
  // 准备确认消息，根据是否有区域选择来调整
  let confirmMessage = "";
  if (currentRegion.value) {
    confirmMessage = `确定要清空区域 "${currentRegion.value}" 中的所有设备吗？此操作不可恢复！`;
  } else {
    confirmMessage = "确定要清空所有设备列表吗？此操作不可恢复！";
  }

  const confirmed = await showConfirmDialog("确认清空设备列表", confirmMessage);

  if (!confirmed) {
    return;
  }

  try {
    if (currentRegion.value) {
      // 如果选择了区域，只清空该区域的设备
      const deviceIDs = devices.value
        .filter((device) => device.region === currentRegion.value)
        .map((device) => device.id);

      if (deviceIDs.length === 0) {
        showNotification(
          `区域 "${currentRegion.value}" 中没有设备可清空`,
          "info"
        );
        return;
      }

      for (const id of deviceIDs) {
        await wailsBackend.RemoveDevice(id);
      }

      showNotification(`区域 "${currentRegion.value}" 的设备已清空`, "success");
    } else {
      // 如果没有选择区域，清空所有设备
      await wailsBackend.ClearDevices();
      showNotification("所有设备已清空", "success");
    }

    await loadDevices(); // 重新加载设备列表
  } catch (error) {
    console.error("清空设备列表失败:", error);
    showNotification(`清空设备列表失败: ${error}`, "error");
  }
}

// 更新设备选择状态
function updateDeviceSelection() {
  const newSelection: Record<string, boolean> = {};
  devices.value.forEach((device) => {
    // 只为在线设备设置选择状态
    if (device.status === "online") {
      // 保留已有的选择状态，或者根据selectAll设置新设备的选择状态
      newSelection[device.id] =
        selectedDevices.value[device.id] !== undefined
          ? selectedDevices.value[device.id]
          : selectAll.value;
    }
  });
  selectedDevices.value = newSelection;
}

// 切换全选/全不选
function toggleSelectAll() {
  selectAll.value = !selectAll.value;
  const newSelection = { ...selectedDevices.value };
  devices.value.forEach((device) => {
    if (device.status === "online") {
      newSelection[device.id] = selectAll.value;
    }
  });
  selectedDevices.value = newSelection;
}

async function setMd5File(filename: string) {
  const input = document.getElementById("md5-file-input") as HTMLInputElement;

  if (!filename) {
    alert("请选择MD5文件");
    return;
  }

  if (!filename.endsWith(".md5")) {
    alert("请选择.md5格式的文件");
    input.value = "";
    return;
  }

  try {
    console.log("设置MD5文件:", filename);

    // 告知用户我们正在寻找文件
    const fileStatus = document.getElementById("md5-file-status");
    if (fileStatus) {
      fileStatus.textContent = "正在寻找文件...";
      fileStatus.className = "file-status searching";
    }

    // SetMd5File不存在，临时处理
    // const filePath = await backend.SetMd5File(filename);
    const filePath = filename; // 简单替代，直接使用文件名

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

// Add a function to check if a backend function exists
function checkBackendFunction(functionName) {
  if (!wailsBackend) {
    console.error(`无法使用后端服务: wailsBackend 未初始化`);
    return false;
  }

  if (typeof wailsBackend[functionName] !== "function") {
    console.error(`后端函数 ${functionName} 未找到或不是函数`);
    return false;
  }

  return true;
}

// 监听标签页切换，添加对软件更新标签的处理
watch(activeTab, (newValue, oldValue) => {
  if (
    newValue === "devices" &&
    oldValue !== "devices" &&
    oldValue !== undefined
  ) {
    console.log("切换到设备管理标签页，自动更新设备列表");
    loadDevices();
  }

  // 当切换到软件更新标签时，检查更新功能是否可用
  if (newValue === "update" && oldValue !== "update") {
    console.log("切换到软件更新标签页，检查更新功能");

    // 检查后端更新功能是否可用
    if (!checkBackendFunction("UpdateDevicesFile")) {
      showNotification("软件更新功能暂不可用，请联系开发人员", "warning");
    }
  }
});

// 实现设备更新功能
async function updateDevices() {
  if (!selectedFile.value) {
    showNotification("请选择更新文件", "warning");
    return;
  }

  if (!username.value || !password.value) {
    showNotification("请输入用户名和密码", "warning");
    return;
  }

  const selectedDevices = selectedDevicesList.value;
  if (selectedDevices.length === 0) {
    showNotification("请至少选择一个在线设备进行更新", "warning");
    return;
  }

  try {
    isLoading.value = true;
    showNotification("正在更新设备...", "info");

    // 调用后端的更新方法
    const results = await wailsBackend.UpdateDevicesFile(
      selectedFile.value,
      0 // 更新所有选中的设备
    );

    // 处理结果
    updateResults.value = results || [];
    processUpdateResults(results);
  } catch (error) {
    console.error("更新设备失败:", error);
    showNotification(`更新失败: ${error}`, "error");
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <div class="container">
    <!-- 通知组件 -->
    <div
      v-if="notification.show"
      class="notification"
      :class="notification.type"
    >
      <span class="notification-message">{{ notification.message }}</span>
      <button class="notification-close" @click="notification.show = false">
        ×
      </button>
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
          <button @click="confirmDialog.onCancel" class="secondary-button">
            取消
          </button>
          <button @click="confirmDialog.onConfirm" class="danger-button">
            确认
          </button>
        </div>
      </div>
    </div>

    <div class="header">
      <h1>设备更新管理 <span class="version">v1.1.6</span></h1>
    </div>

    <div class="tabs">
      <button
        :class="{ active: activeTab === 'camera' }"
        @click="activeTab = 'camera'"
      >
        摄像头配置
      </button>
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
        :class="{ active: activeTab === 'time' }"
        @click="activeTab = 'time'"
      >
        时间同步
      </button>
      <button
        :class="{ active: activeTab === 'backup' }"
        @click="activeTab = 'backup'"
      >
        设备备份管理
      </button>
    </div>

    <!-- Device Management Tab -->
    <div v-if="activeTab === 'devices'" class="tab-content">
      <!-- 区域选择 -->
      <div class="card">
        <h2>区域管理</h2>
        <div class="form-group">
          <div v-if="!customRegion" class="region-select-container">
            <select
              v-model="currentRegion"
              class="region-select"
              :disabled="regionLoading"
            >
              <option value="">-- 选择区域 --</option>
              <option v-for="region in regions" :key="region" :value="region">
                {{ region }}
              </option>
            </select>
            <button
              @click="toggleCustomRegion"
              class="secondary-button"
              title="添加新区域"
            >
              新区域
            </button>
          </div>
          <div v-else class="region-input-container">
            <input
              v-model="newRegion"
              placeholder="输入新区域名称"
              class="region-input"
              @keyup.enter="applyCustomRegion"
            />
            <button
              @click="toggleCustomRegion"
              class="secondary-button"
              title="返回选择"
            >
              选择已有
            </button>
          </div>
          <div class="region-actions">
            <button
              @click="applyRegion"
              class="primary-button apply-region-button"
              :disabled="
                (!currentRegion && !newRegion) ||
                selectedDevicesList.length === 0
              "
            >
              应用区域到选中设备
            </button>
            <button
              @click="applyRegionToDevicesWithoutRegion"
              class="secondary-button"
              :disabled="
                (!currentRegion && !newRegion) || devicesWithoutRegion === 0
              "
              title="将当前区域应用到所有未分配区域的设备"
            >
              应用到无区域设备 ({{ devicesWithoutRegion }})
            </button>
          </div>
        </div>
        <div v-if="currentRegion" class="region-filter-notice">
          <p>
            当前显示区域
            <strong>{{ currentRegion }}</strong> 的设备和所有无区域设备
            <button class="text-button" @click="currentRegion = ''">
              显示所有设备
            </button>
          </p>
        </div>
      </div>

      <div class="card">
        <h2>添加设备{{ currentRegion ? " (" + currentRegion + ")" : "" }}</h2>
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
        <h2>扫描IP范围{{ currentRegion ? " (" + currentRegion + ")" : "" }}</h2>
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
          <h2>设备列表{{ currentRegion ? " (" + currentRegion + ")" : "" }}</h2>
          <div class="device-list-actions">
            <button
              @click="refreshDevices"
              :disabled="isLoading"
              class="refresh-button"
              title="刷新设备列表"
            >
              {{ isLoading ? "刷新中..." : "刷新" }}
            </button>
            <button
              @click="clearDeviceList"
              :disabled="isLoading"
              class="danger-button"
              :title="
                currentRegion
                  ? `清空${currentRegion}区域的设备`
                  : '清空所有设备'
              "
            >
              {{ currentRegion ? `清空区域设备` : "清空所有设备" }}
            </button>
          </div>
        </div>

        <div v-if="devices.length === 0" class="empty-state">
          未找到设备。请手动添加设备或扫描设备。
        </div>

        <table v-else class="device-table">
          <thead>
            <tr>
              <th style="width: 30px">
                <input
                  type="checkbox"
                  :checked="selectAll"
                  @change="toggleSelectAll"
                />
              </th>
              <th>IP地址</th>
              <th>构建时间</th>
              <th>区域</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="device in paginatedDevices" :key="device.ip">
              <td>
                <input type="checkbox" v-model="selectedDevices[device.id]" />
              </td>
              <td>{{ device.ip }}</td>
              <td>{{ device.buildTime }}</td>
              <td>{{ device.region || "-" }}</td>
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
                  @click="removeDevice(device)"
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
              v-for="(group, groupKey) in groupedDevices"
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
                    <span class="device-count">({{ group.length }}台)</span>
                  </span>
                </label>
                <div class="group-actions">
                  <button
                    v-if="
                      groupKey !== '未知版本' &&
                      group.some((d) => d.status === 'online')
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
                  v-for="device in group"
                  :key="device.id"
                  class="device-selection-item"
                  :class="{
                    'device-offline': device.status !== 'online',
                    'device-removing': device.status === 'removing',
                  }"
                >
                  <label class="checkbox-label">
                    <input
                      type="checkbox"
                      v-model="selectedDevices[device.id]"
                      :disabled="device.status !== 'online'"
                      @change="toggleDeviceSelection(device, groupKey)"
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
            isLoading
              ? "更新中..."
              : `更新选中的设备 (${selectedDevicesList.length})`
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

    <!-- Device Backup Tab -->
    <div v-if="activeTab === 'backup'" class="tab-content">
      <DeviceBackup />
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
  max-width: 100%;
  height: 100%;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.app-header {
  padding: 15px 20px;
  background-color: var(--card-background);
  border-bottom: 1px solid var(--border-color);
}

.tabs {
  display: flex;
  flex-wrap: nowrap;
  overflow-x: auto;
  background-color: var(--card-background);
  border-bottom: 1px solid var(--border-color);
  padding: 0 20px;
}

.tab-content {
  flex: 1;
  overflow: auto;
  padding: 0;
  background-color: var(--background-color);
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

/* 恢复被删除的样式 */
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

/* 新增区域选择样式 */
.region-select-container,
.region-input-container {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 15px;
  width: 100%;
}

.region-select {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background-color: white;
  font-size: 14px;
}

.region-input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 14px;
}

.secondary-button {
  background-color: #f0f0f0;
  color: var(--text-color);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  padding: 8px 16px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.secondary-button:hover:not(:disabled) {
  background-color: #e0e0e0;
}

.secondary-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.apply-region-button {
  margin-top: 10px;
  width: 100%;
  padding: 10px 20px;
  background-color: var(--primary-color);
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.apply-region-button:hover:not(:disabled) {
  background-color: var(--primary-hover);
}

.apply-region-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.save-button {
  padding: 6px 12px;
  background-color: var(--primary-color);
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.save-button:hover:not(:disabled) {
  background-color: var(--primary-hover);
}

.save-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.device-list-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

/* 让表格中的区域列显示更美观 */
.device-table td:nth-child(4) {
  font-style: italic;
  color: #555;
}

/* 保持高亮显示区域列中的实际值 */
.device-table td:nth-child(4):not(:empty) {
  font-style: normal;
  color: var(--primary-color);
  font-weight: 500;
}

/* 通知样式 */
.notification {
  position: fixed;
  top: 20px;
  right: 20px;
  padding: 12px 16px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  animation: slide-in 0.3s ease-out;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 1000;
  max-width: 350px;
}

@keyframes slide-in {
  from {
    transform: translateX(20px);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

.notification-message {
  flex-grow: 1;
  margin-right: 10px;
}

.notification-close {
  background: transparent;
  border: none;
  color: inherit;
  font-size: 18px;
  cursor: pointer;
  padding: 0 4px;
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

/* 添加的新样式 */
.region-actions {
  display: flex;
  gap: 10px;
  margin-top: 10px;
}

.region-filter-notice {
  margin-top: 10px;
  padding: 8px 12px;
  background-color: #e9f5ff;
  border-radius: 4px;
  border-left: 4px solid var(--primary-color);
}

.region-filter-notice p {
  margin: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.text-button {
  background: none;
  border: none;
  color: var(--primary-color);
  cursor: pointer;
  text-decoration: underline;
  padding: 2px 5px;
}

.text-button:hover {
  color: var(--primary-hover);
}
</style>
