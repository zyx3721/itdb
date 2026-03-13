<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import api from '../api/client'
import { useAuthStore } from '../stores/auth'
import { useNoticeStore } from '../stores/notice'

const auth = useAuthStore()
const noticeStore = useNoticeStore()
const saving = ref(false)
const testing = ref(false)
const loading = ref(false)
const error = ref('')
const savedSignature = ref('')

const form = reactive({
  useLdap: 0,
  ldapServer: '',
  ldapDn: '',
  ldapBindDn: '',
  ldapBindPassword: '',
  ldapGetUsers: '',
  ldapGetUsersFilter: '',
})

const ldapEnabled = computed(() => Number(form.useLdap ?? 0) === 1)

function buildSettingsSignature() {
  return JSON.stringify({
    useLdap: Number(form.useLdap ?? 0),
    ldapServer: form.ldapServer ?? '',
    ldapDn: form.ldapDn ?? '',
    ldapBindDn: form.ldapBindDn ?? '',
    ldapBindPassword: form.ldapBindPassword ?? '',
    ldapGetUsers: form.ldapGetUsers ?? '',
    ldapGetUsersFilter: form.ldapGetUsersFilter ?? '',
  })
}

function getRequiredFieldsError() {
  if (Number(form.useLdap ?? 0) !== 1) return ''

  const missing: string[] = []
  if (!String(form.ldapServer ?? '').trim()) missing.push('服务器地址')
  if (!String(form.ldapDn ?? '').trim()) missing.push('基准 DN')
  if (!String(form.ldapBindDn ?? '').trim()) missing.push('绑定 DN')
  if (!String(form.ldapBindPassword ?? '').trim()) missing.push('绑定密码')

  return missing.length > 0 ? `请完善必填项：${missing.join('、')}` : ''
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const { data } = await api.get('/settings')
    form.useLdap = Number(data.useldap ?? 0)
    form.ldapServer = data.ldap_server ?? ''
    form.ldapDn = data.ldap_dn ?? ''
    form.ldapBindDn = data.ldap_bind_dn ?? ''
    form.ldapBindPassword = data.ldap_bind_password ?? ''
    form.ldapGetUsers = data.ldap_getusers ?? ''
    form.ldapGetUsersFilter = data.ldap_getusers_filter ?? ''
    savedSignature.value = buildSettingsSignature()
  } catch (err: unknown) {
    error.value = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? '系统配置加载失败'
  } finally {
    loading.value = false
  }
}

async function save() {
  if (auth.isReadOnly) return

  const requiredError = getRequiredFieldsError()
  if (requiredError) {
    noticeStore.error(requiredError)
    return
  }

  saving.value = true
  error.value = ''
  try {
    const { data } = await api.put('/settings', { ...form })
    form.useLdap = Number(data.useldap ?? form.useLdap ?? 0)
    form.ldapServer = data.ldap_server ?? form.ldapServer
    form.ldapDn = data.ldap_dn ?? form.ldapDn
    form.ldapBindDn = data.ldap_bind_dn ?? form.ldapBindDn
    form.ldapBindPassword = data.ldap_bind_password ?? form.ldapBindPassword
    form.ldapGetUsers = data.ldap_getusers ?? form.ldapGetUsers
    form.ldapGetUsersFilter = data.ldap_getusers_filter ?? form.ldapGetUsersFilter
    savedSignature.value = buildSettingsSignature()
    noticeStore.success('设置已保存')
  } catch {
  } finally {
    saving.value = false
  }
}

async function testConnection() {
  if (auth.isReadOnly) return
  if (savedSignature.value === '' || savedSignature.value !== buildSettingsSignature()) {
    noticeStore.error('请先保存当前 LDAP 配置，再进行连接测试')
    return
  }

  testing.value = true
  error.value = ''
  try {
    const { data } = await api.post('/settings/test-ldap', { ...form })
    noticeStore.success(data?.message ?? 'LDAP 连接成功')
  } catch {
  } finally {
    testing.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="page-shell">
    <header class="page-header">
      <h2>设置</h2>
      <button class="ghost-btn" @click="load">重载</button>
    </header>

    <p v-if="loading">加载中...</p>
    <p v-else-if="error" class="error-text section-gap">{{ error }}</p>

    <form v-else class="settings-grid settings-form" @submit.prevent="save">
      <div class="settings-ldap-intro">
        <h3>LDAP 配置</h3>
        <p class="muted-text">连接测试必须基于已保存配置执行。测试仅验证 LDAP 服务器连通和绑定认证是否成功，不会实际执行用户搜索。</p>
        <p class="muted-text settings-ldap-note">%{attr} 表示登录时参与匹配的 LDAP 属性名，%{user} 表示当前输入的用户名；这两个占位符仅作为搜索过滤模板保留使用。</p>
      </div>
      <label class="settings-field">
        <span class="settings-field-label">启用 LDAP</span>
        <select v-model.number="form.useLdap">
          <option :value="0">禁用</option>
          <option :value="1">启用</option>
        </select>
      </label>
      <label class="settings-field">
        <span class="settings-field-label">服务器地址</span>
        <input
          v-model="form.ldapServer"
          :disabled="!ldapEnabled"
          type="text"
          placeholder="例如：ldap://ad.example.com:389 或 ldaps://ad.example.com:636"
        />
      </label>
      <label class="settings-field">
        <span class="settings-field-label">基准 DN</span>
        <input v-model="form.ldapDn" :disabled="!ldapEnabled" type="text" placeholder="例如：OU=Users,DC=example,DC=com" />
      </label>
      <label class="settings-field">
        <span class="settings-field-label">绑定 DN</span>
        <input
          v-model="form.ldapBindDn"
          :disabled="!ldapEnabled"
          type="text"
          placeholder="例如：CN=ldap-reader,OU=Service Accounts,DC=example,DC=com"
        />
      </label>
      <label class="settings-field">
        <span class="settings-field-label">绑定密码</span>
        <input
          v-model="form.ldapBindPassword"
          :disabled="!ldapEnabled"
          type="password"
          autocomplete="new-password"
          placeholder="请输入绑定 DN 对应的密码"
        />
      </label>
      <label class="settings-field">
        <span class="settings-field-label">用户查询过滤模板</span>
        <textarea
          v-model="form.ldapGetUsers"
          :disabled="!ldapEnabled"
          rows="3"
          placeholder="例如：(&(objectClass=user)(|(cn=%s)(sAMAccountName=%s)(userPrincipalName=%s)))"
        />
      </label>
      <label class="settings-field">
        <span class="settings-field-label">附加搜索过滤器</span>
        <textarea
          v-model="form.ldapGetUsersFilter"
          :disabled="!ldapEnabled"
          rows="3"
          placeholder="例如：(%{attr}=%{user}) 或 (memberOf=CN=IT,OU=Groups,DC=example,DC=com)"
        />
        <small class="settings-field-help">可直接填写 (%{attr}=%{user}) 这类模板；其中 %{attr} 是属性名占位，%{user} 是用户名占位。</small>
      </label>

      <div class="settings-actions">
        <button :disabled="testing || saving || auth.isReadOnly" type="submit">
          {{ saving ? '保存中...' : auth.isReadOnly ? '只读模式' : '保存设置' }}
        </button>
        <button :disabled="!ldapEnabled || testing || saving || auth.isReadOnly" class="ghost-btn" type="button" @click="testConnection">
          {{ testing ? '测试中...' : auth.isReadOnly ? '只读模式' : '连接测试' }}
        </button>
      </div>
    </form>
  </section>
</template>

<style scoped>
.settings-grid {
  max-width: 960px;
}

.settings-form {
  grid-template-columns: minmax(0, 1fr);
  gap: 14px;
}

.settings-ldap-intro {
  padding: 18px 20px;
  border: 1px solid rgba(47, 127, 186, 0.14);
  border-radius: 16px;
  background: linear-gradient(180deg, rgba(47, 127, 186, 0.07), rgba(47, 127, 186, 0.02));
}

.settings-ldap-intro h3 {
  margin: 0 0 6px;
}

.settings-ldap-note {
  margin-top: 6px;
}

.settings-field {
  gap: 8px;
}

.settings-field-label {
  color: #16324f;
}

.settings-field-help {
  color: #64748b;
  font-size: 0.85rem;
  line-height: 1.45;
}

.settings-form :deep(input),
.settings-form :deep(select),
.settings-form :deep(textarea) {
  min-height: 44px;
  padding: 0.65rem 0.8rem;
}

.settings-form :deep(input:disabled),
.settings-form :deep(select:disabled),
.settings-form :deep(textarea:disabled) {
  color: #7b8da0;
  background: #eef3f7;
  border-color: #d4dee8;
  cursor: not-allowed;
}

.settings-grid textarea {
  min-height: 92px;
  resize: vertical;
}

.settings-actions {
  display: flex;
  justify-content: flex-start;
  padding-top: 4px;
  gap: 10px;
  flex-wrap: wrap;
}

.settings-actions button {
  min-width: 120px;
  min-height: 44px;
}
</style>
