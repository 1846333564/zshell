<template>
  <div>
    <h2>{{ title }}</h2>

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
      <label>认证方式</label>
      <div class="auth-options">
        <label>
          <input v-model="form.authMethod" type="radio" value="password" />
          密码
        </label>
        <label>
          <input v-model="form.authMethod" type="radio" value="id_rsa" />
          ~/.ssh/id_rsa
        </label>
      </div>
    </div>

    <div v-if="form.authMethod === 'password'" class="form-row">
      <label>密码</label>
      <input v-model="form.password" type="password" :placeholder="passwordPlaceholder" />
    </div>

    <button class="button-main" :disabled="busy" @click="submit">
      {{ busy ? '处理中...' : submitLabel }}
    </button>

    <div class="error">{{ error || '\u00A0' }}</div>

    <p class="tip">
      连接配置保存在 Windows 当前用户配置中。
    </p>
  </div>
</template>

<script setup>
import { computed, reactive, watch } from 'vue';

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
      id: '',
      name: '默认服务器',
      host: '127.0.0.1',
      port: 22,
      username: 'root',
      password: '',
      authMethod: 'password',
    }),
  },
  mode: {
    type: String,
    default: 'create',
  },
  submitLabel: {
    type: String,
    default: '保存并连接',
  },
  title: {
    type: String,
    default: '连接配置',
  },
});

const emit = defineEmits(['connect']);

const form = reactive({
  id: '',
  name: '',
  host: '',
  port: 22,
  username: '',
  password: '',
  authMethod: 'password',
});

const passwordPlaceholder = computed(() => (props.mode === 'edit' ? '留空则保留已保存密码' : '请输入密码'));

watch(
  () => props.initialValue,
  (value) => {
    form.id = value?.id || '';
    form.name = value?.name || '默认服务器';
    form.host = value?.host || '127.0.0.1';
    form.port = Number(value?.port) || 22;
    form.username = value?.username || 'root';
    form.password = value?.password || '';
    form.authMethod = value?.authMethod || 'password';
  },
  { immediate: true, deep: true },
);

function submit() {
  if (props.busy) {
    return;
  }

  emit('connect', {
    id: form.id,
    name: form.name,
    host: form.host,
    port: Number(form.port) || 22,
    username: form.username,
    password: form.authMethod === 'password' ? form.password : '',
    authMethod: form.authMethod,
  });
}
</script>
