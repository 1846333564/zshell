<template>
  <div>
    <h2>连接配置</h2>

    <div class="form-row">
      <label>连接名称</label>
      <input v-model.trim="form.name" placeholder="例如: 生产机-01" />
    </div>

    <div class="form-row">
      <label>主机 IP</label>
      <input v-model.trim="form.host" placeholder="192.168.1.10" />
    </div>

    <div class="form-row">
      <label>端口</label>
      <input v-model.number="form.port" type="number" min="1" max="65535" placeholder="22" />
    </div>

    <div class="form-row">
      <label>用户名</label>
      <input v-model.trim="form.username" placeholder="root" />
    </div>

    <div class="form-row">
      <label>密码</label>
      <input v-model="form.password" type="password" placeholder="请输入密码" />
    </div>

    <button class="button-main" :disabled="busy" @click="submit">
      {{ busy ? '连接中...' : '测试并连接' }}
    </button>

    <div class="error">{{ error || '\u00A0' }}</div>

    <p class="tip">
      连接成功后将进入工作台页面，可使用多标签终端和下方文件管理器。
    </p>
  </div>
</template>

<script setup>
import { reactive, watch } from 'vue';

const props = defineProps({
  busy: {
    type: Boolean,
    default: false,
  },
  error: {
    type: String,
    default: '',
  },
  initialValue: {
    type: Object,
    default: () => ({
      name: '默认服务器',
      host: '127.0.0.1',
      port: 22,
      username: 'root',
      password: '',
    }),
  },
});

const emit = defineEmits(['connect']);

const form = reactive({
  name: '',
  host: '',
  port: 22,
  username: '',
  password: '',
});

watch(
  () => props.initialValue,
  (value) => {
    form.name = value?.name || '默认服务器';
    form.host = value?.host || '127.0.0.1';
    form.port = Number(value?.port) || 22;
    form.username = value?.username || 'root';
    form.password = value?.password || '';
  },
  { immediate: true, deep: true },
);

function submit() {
  if (props.busy) {
    return;
  }

  emit('connect', {
    name: form.name,
    host: form.host,
    port: Number(form.port) || 22,
    username: form.username,
    password: form.password,
  });
}
</script>
