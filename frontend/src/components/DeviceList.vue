<script setup lang="ts">
import { ref, computed } from "vue";
import {
  GetDevices,
  UpdateDevices,
  RefreshDevices,
  GetMd5File,
} from "../../wailsjs/go/main/App";
import { main } from "../../wailsjs/go/models";

const devices = ref<main.Device[]>([]);
const username = ref("");
const password = ref("");
const updating = ref(false);
const updateResults = ref<main.UpdateResult[]>([]);
const selectedDevices = ref<Set<string>>(new Set()); // 用于存储选中的设备IP

// 按buildTime分组的计算属性
const groupedDevices = computed(() => {
  return devices.value.reduce(
    (groups, device) => {
      const key = device.buildTime || "Unknown";
      if (!groups[key]) {
        groups[key] = {
          buildTime: key,
          devices: [],
          selected: false,
        };
      }
      groups[key].devices.push(device);
      return groups;
    },
    {} as Record<
      string,
      {
        buildTime: string;
        devices: main.Device[];
        selected: boolean;
      }
    >
  );
});

// 加载设备列表
const loadDevices = async () => {
  try {
    devices.value = await GetDevices();
  } catch (error) {
    console.error("Failed to load devices:", error);
  }
};

// 刷新设备状态
const refreshDevices = async () => {
  try {
    devices.value = await RefreshDevices();
  } catch (error) {
    console.error("Failed to refresh devices:", error);
  }
};

// 切换设备组选择状态
const toggleGroupSelection = (buildTime: string) => {
  const group = groupedDevices.value[buildTime];
  const newSelected = !group.selected;
  group.selected = newSelected;

  group.devices.forEach((device) => {
    if (newSelected) {
      selectedDevices.value.add(device.ip);
    } else {
      selectedDevices.value.delete(device.ip);
    }
  });
};

// 切换单个设备选择状态
const toggleDeviceSelection = (ip: string, buildTime: string) => {
  if (selectedDevices.value.has(ip)) {
    selectedDevices.value.delete(ip);
  } else {
    selectedDevices.value.add(ip);
  }

  // 更新组的选择状态
  const group = groupedDevices.value[buildTime];
  group.selected = group.devices.every((d) => selectedDevices.value.has(d.ip));
};

// 更新选中的设备
const updateSelectedDevices = async () => {
  if (!username.value || !password.value) {
    alert("请输入用户名和密码");
    return;
  }

  if (selectedDevices.value.size === 0) {
    alert("请选择要更新的设备");
    return;
  }

  // 获取选中设备的 buildTime
  const selectedBuildTimes = new Set<string>();
  selectedDevices.value.forEach((deviceIP) => {
    const device = devices.value.find((d) => d.ip === deviceIP);
    if (device && device.buildTime) {
      selectedBuildTimes.add(device.buildTime);
    }
  });

  if (selectedBuildTimes.size > 1) {
    alert("请选择相同版本的设备进行更新");
    return;
  }

  const buildTime =
    selectedBuildTimes.size > 0 ? Array.from(selectedBuildTimes)[0] : "";

  try {
    updating.value = true;
    const results = await UpdateDevices(
      username.value,
      password.value,
      buildTime,
      await GetMd5File()
    );
    updateResults.value = results;
    await refreshDevices();
  } catch (error) {
    console.error("Update failed:", error);
    alert(`更新失败: ${error}`);
  } finally {
    updating.value = false;
  }
};

// 初始加载
loadDevices();
</script>

<template>
  <div class="device-list">
    <!-- 认证信息输入 -->
    <div class="auth-section">
      <input
        v-model="username"
        type="text"
        placeholder="用户名"
        :disabled="updating"
      />
      <input
        v-model="password"
        type="password"
        placeholder="密码"
        :disabled="updating"
      />
      <button @click="refreshDevices" :disabled="updating">刷新设备列表</button>
      <button
        @click="updateSelectedDevices"
        :disabled="updating || selectedDevices.size === 0"
        class="update-btn"
      >
        更新选中设备 ({{ selectedDevices.size }})
      </button>
    </div>

    <!-- 设备组列表 -->
    <div class="device-groups">
      <div
        v-for="(group, buildTime) in groupedDevices"
        :key="buildTime"
        class="device-group"
      >
        <div class="group-header">
          <label class="group-checkbox">
            <input
              type="checkbox"
              :checked="group.selected"
              @change="toggleGroupSelection(buildTime)"
              :disabled="updating"
            />
            <span class="version">版本: {{ buildTime }}</span>
            <span class="count">({{ group.devices.length }}台设备)</span>
          </label>
        </div>

        <!-- 设备列表 -->
        <div class="device-items">
          <div
            v-for="device in group.devices"
            :key="device.ip"
            class="device-item"
          >
            <label class="device-checkbox">
              <input
                type="checkbox"
                :checked="selectedDevices.has(device.ip)"
                @change="toggleDeviceSelection(device.ip, buildTime)"
                :disabled="updating"
              />
              <span class="ip">{{ device.ip }}</span>
              <span :class="['status', device.status]">{{
                device.status
              }}</span>
            </label>
          </div>
        </div>
      </div>
    </div>

    <!-- 更新结果显示 -->
    <div v-if="updateResults.length" class="update-results">
      <h4>更新结果:</h4>
      <div
        v-for="result in updateResults"
        :key="result.ip"
        :class="['result-item', { success: result.success }]"
      >
        {{ result.ip }}: {{ result.message }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.device-list {
  padding: 20px;
}

.auth-section {
  margin-bottom: 20px;
  display: flex;
  gap: 10px;
  align-items: center;
}

.auth-section input {
  padding: 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
}

.update-btn {
  background-color: #28a745;
}

.update-btn:disabled {
  background-color: #6c757d;
}

.device-group {
  margin-bottom: 20px;
  padding: 15px;
  border: 1px solid #ddd;
  border-radius: 8px;
  background: #f8f9fa;
}

.group-header {
  margin-bottom: 10px;
}

.group-checkbox,
.device-checkbox {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.group-checkbox input[type="checkbox"],
.device-checkbox input[type="checkbox"] {
  margin: 0;
}

.version {
  font-weight: bold;
}

.count {
  color: #666;
}

.device-items {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: 10px;
  margin-left: 24px;
}

.device-item {
  padding: 8px;
  border: 1px solid #eee;
  border-radius: 4px;
}

.status {
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 0.9em;
}

.status.online {
  background-color: #d4edda;
  color: #155724;
}

.status.offline {
  background-color: #f8d7da;
  color: #721c24;
}

.update-results {
  margin-top: 20px;
  padding: 15px;
  border: 1px solid #ddd;
  border-radius: 8px;
}

.result-item {
  padding: 4px 0;
  color: #dc3545;
}

.result-item.success {
  color: #28a745;
}

button {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  background: #007bff;
  color: white;
  cursor: pointer;
}

button:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}
</style>
