<template>
  <div class="monitor-panel">
    <div class="monitor-identity">
      <span>{{ session?.host || '-' }}</span>
      <span>{{ session?.port || '-' }}</span>
      <span>{{ session?.username || '-' }}</span>
    </div>

    <div class="load-grid">
      <div class="load-cell">
        <span>CPU</span>
        <strong>{{ percent(snapshot?.loads?.cpuPercent) }}</strong>
      </div>
      <div class="load-cell">
        <span>内存</span>
        <strong>{{ percent(snapshot?.loads?.memoryPercent) }}</strong>
      </div>
      <div class="load-cell">
        <span>硬盘</span>
        <strong>{{ percent(snapshot?.loads?.diskPercent) }}</strong>
      </div>
    </div>

    <section class="monitor-section server-section">
      <div class="section-row">
        <span>服务器</span>
        <span>{{ loading ? '更新中' : serverClockText }}</span>
      </div>
      <div class="server-metric-grid">
        <div v-for="item in serverMetrics" :key="item.label" class="server-metric">
          <span>{{ item.label }}</span>
          <strong>{{ item.value }}</strong>
        </div>
      </div>
    </section>

    <section class="monitor-section network-section">
      <div class="section-row">
        <span>网速</span>
        <span>{{ networkPeakText }}</span>
      </div>
      <div class="network-chart">
        <div class="network-scale">
          <span>{{ speed(networkPeak) }}</span>
          <span>{{ speed(networkMidpoint) }}</span>
          <span>0</span>
        </div>
        <svg viewBox="0 0 100 48" preserveAspectRatio="none" aria-hidden="true">
          <line x1="0" y1="46" x2="100" y2="46" class="network-axis"></line>
          <line x1="0" y1="26" x2="100" y2="26" class="network-grid-line"></line>
          <line x1="0" y1="6" x2="100" y2="6" class="network-grid-line"></line>
          <polyline v-if="networkRxPoints" :points="networkRxPoints" class="network-line rx"></polyline>
          <polyline v-if="networkTxPoints" :points="networkTxPoints" class="network-line tx"></polyline>
        </svg>
      </div>
      <div class="network-legend">
        <span><i class="rx"></i>下行 {{ speed(totalNetwork.rxBps) }}</span>
        <span><i class="tx"></i>上行 {{ speed(totalNetwork.txBps) }}</span>
      </div>
      <div class="net-table">
        <div v-for="net in networks" :key="net.name" class="net-row">
          <strong>{{ net.name }}</strong>
          <span>↓ {{ speed(net.rxBps) }}</span>
          <span>↑ {{ speed(net.txBps) }}</span>
        </div>
        <div v-if="networks.length === 0" class="empty-tip compact">暂无数据</div>
      </div>
    </section>

    <section class="monitor-section">
      <div class="section-row">
        <span>进程</span>
        <div class="sort-buttons">
          <button :class="{ active: processSort === 'memory' }" @click="setSort('memory')">内存</button>
          <button :class="{ active: processSort === 'cpu' }" @click="setSort('cpu')">CPU</button>
        </div>
      </div>
      <div class="process-table">
        <div class="process-row head">
          <span>内存M</span>
          <span>CPU%</span>
          <span>进程</span>
        </div>
        <div v-for="proc in processes" :key="`${proc.name}-${proc.memoryMB}-${proc.cpuPercent}`" class="process-row">
          <span>{{ proc.memoryMB.toFixed(1) }}</span>
          <span>{{ proc.cpuPercent.toFixed(1) }}</span>
          <span class="process-name">{{ proc.name }}</span>
        </div>
        <div v-if="processes.length === 0" class="empty-tip compact">暂无数据</div>
      </div>
    </section>

    <section class="monitor-section partitions">
      <div class="section-row">
        <span>分区</span>
      </div>
      <div v-for="part in partitions" :key="`${part.fileSystem}-${part.mount}`" class="partition-row">
        <div>
          <strong>{{ part.mount }}</strong>
          <span>{{ part.fileSystem }}</span>
        </div>
        <div>{{ size(part.freeBytes) }} / {{ size(part.totalBytes) }}</div>
      </div>
      <div v-if="partitions.length === 0" class="empty-tip compact">暂无数据</div>
    </section>

    <div class="file-error">{{ errorMessage || '\u00A0' }}</div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { getMonitorSnapshot } from '../services/apiClient';

const HISTORY_WINDOW_MS = 60 * 1000;

const props = defineProps({
  session: {
    type: Object,
    default: null,
  },
});

const snapshot = ref(null);
const loading = ref(false);
const errorMessage = ref('');
const processSort = ref('memory');
const networkHistory = ref([]);
let timer = null;

const processes = computed(() => snapshot.value?.processes || []);
const networks = computed(() => snapshot.value?.networks || []);
const partitions = computed(() => snapshot.value?.partitions || []);
const system = computed(() => snapshot.value?.system || {});
const hardware = computed(() => props.session?.hardware || {});
const totalNetwork = computed(() => networks.value.find((net) => net.name === 'total') || { rxBps: 0, txBps: 0 });
const networkPeak = computed(() =>
  Math.max(
    0,
    ...networkHistory.value.flatMap((sample) => [Number(sample.rxBps) || 0, Number(sample.txBps) || 0]),
  ),
);
const networkPeakText = computed(() => `峰值 ${speed(networkPeak.value)}`);
const networkMidpoint = computed(() => networkPeak.value / 2);
const networkRxPoints = computed(() => networkChartPoints('rxBps'));
const networkTxPoints = computed(() => networkChartPoints('txBps'));
const serverClockText = computed(() => formatServerTime(system.value.serverTime, system.value.timeZone));
const serverMetrics = computed(() => {
  const loads = snapshot.value?.loads || {};
  return [
    { label: '主机名', value: snapshot.value?.host?.name || props.session?.host || '-' },
    { label: '服务器时间', value: serverClockText.value },
    { label: '运行时间', value: uptime(system.value.uptimeSeconds) },
    { label: '系统', value: system.value.os || '-' },
    { label: '内核', value: system.value.kernel || '-' },
    { label: 'CPU型号', value: hardware.value.cpuModel || '-' },
    { label: '线程/核心', value: cpuShape(hardware.value) },
    { label: '内存总量', value: size(Number(hardware.value.memoryTotalBytes) || Number(loads.memoryTotalMB) * 1024 * 1024) },
    { label: '内存已用', value: memoryUsed(loads) },
    { label: '硬件读取', value: shortTime(hardware.value.readAt) },
  ];
});

watch(
  () => props.session?.connectionId,
  () => {
    snapshot.value = null;
    networkHistory.value = [];
    errorMessage.value = '';
    restartTimer();
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  stopTimer();
});

function restartTimer() {
  stopTimer();
  if (!props.session?.connectionId) {
    return;
  }
  refresh();
  timer = window.setInterval(refresh, 1000);
}

function stopTimer() {
  if (timer) {
    window.clearInterval(timer);
    timer = null;
  }
}

async function refresh() {
  if (!props.session?.connectionId || loading.value) {
    return;
  }

  loading.value = true;
  errorMessage.value = '';
  try {
    const result = await getMonitorSnapshot(props.session.connectionId, processSort.value);
    snapshot.value = result.snapshot;
    rememberNetworkSample(result.snapshot);
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '监控刷新失败';
  } finally {
    loading.value = false;
  }
}

function setSort(value) {
  processSort.value = value;
  refresh();
}

function rememberNetworkSample(nextSnapshot) {
  const total = (nextSnapshot?.networks || []).find((net) => net.name === 'total');
  const now = Date.now();
  networkHistory.value = [
    ...networkHistory.value,
    {
      at: now,
      rxBps: Number(total?.rxBps) || 0,
      txBps: Number(total?.txBps) || 0,
    },
  ].filter((sample) => now - sample.at <= HISTORY_WINDOW_MS);
}

function networkChartPoints(key) {
  const samples = networkHistory.value;
  if (samples.length === 0) {
    return '';
  }
  const now = Date.now();
  const start = now - HISTORY_WINDOW_MS;
  const peak = networkPeak.value || 1;
  return samples
    .map((sample) => {
      const x = Math.min(100, Math.max(0, ((sample.at - start) / HISTORY_WINDOW_MS) * 100));
      const y = 46 - Math.min(1, Math.max(0, (Number(sample[key]) || 0) / peak)) * 40;
      return `${x.toFixed(2)},${y.toFixed(2)}`;
    })
    .join(' ');
}

function percent(value) {
  if (typeof value !== 'number') {
    return '-';
  }
  return `${value.toFixed(1)}%`;
}

function speed(value) {
  if (!value) {
    return '0 B/s';
  }
  if (value < 1024) {
    return `${value.toFixed(0)} B/s`;
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KB/s`;
  }
  return `${(value / (1024 * 1024)).toFixed(1)} MB/s`;
}

function size(value) {
  if (!value) {
    return '-';
  }
  if (value < 1024 * 1024 * 1024) {
    return `${(value / (1024 * 1024)).toFixed(1)} MB`;
  }
  return `${(value / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

function uptime(value) {
  const seconds = Number(value) || 0;
  if (seconds <= 0) {
    return '-';
  }
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) {
    return `${days}天 ${hours}小时`;
  }
  if (hours > 0) {
    return `${hours}小时 ${minutes}分钟`;
  }
  return `${minutes}分钟`;
}

function cpuShape(value) {
  const threads = Number(value?.cpuThreads) || 0;
  const cores = Number(value?.cpuCores) || 0;
  if (!threads && !cores) {
    return '-';
  }
  if (!cores) {
    return `${threads} 线程`;
  }
  return `${threads} 线程 / ${cores} 核`;
}

function memoryUsed(loads) {
  const used = Number(loads?.memoryUsedMB) || 0;
  const total = Number(loads?.memoryTotalMB) || 0;
  if (!used && !total) {
    return '-';
  }
  return `${size(used * 1024 * 1024)} / ${size(total * 1024 * 1024)}`;
}

function formatServerTime(value, timeZone) {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  const text = Number.isNaN(date.getTime()) ? value : date.toLocaleString();
  return timeZone ? `${text} ${timeZone}` : text;
}

function shortTime(value) {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}
</script>
