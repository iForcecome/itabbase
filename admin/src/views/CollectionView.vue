<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { api, type MetaCollection, type MetaField } from '../api'

const props = defineProps<{ name: string }>()

const rows = ref<Record<string, unknown>[]>([])
const meta = ref<MetaCollection | null>(null)
const total = ref(0)
const page = ref(1)
const size = ref(20)
const loading = ref(false)
const error = ref('')

// Fields hidden from the edit form (auto-managed or sensitive).
const HIDDEN_FIELDS = new Set(['id', 'password_hash'])

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / size.value)))
const columns = computed<string[]>(() => {
  if (rows.value.length === 0) return []
  return Object.keys(rows.value[0])
})
const editableFields = computed<MetaField[]>(() => {
  if (!meta.value) return []
  return meta.value.fields.filter(
    (f) => !HIDDEN_FIELDS.has(f.name) && f.type !== 'has_many',
  )
})

// Approval-flow awareness: any collection whose row has a `status` field
// supports the pending → active/rejected workflow inline.
const hasStatusField = computed(() =>
  meta.value?.fields.some((f) => f.name === 'status') ?? false,
)
const pendingCount = computed(() =>
  rows.value.filter((r) => (r as Record<string, unknown>).status === 'pending').length,
)
function isPending(row: Record<string, unknown>): boolean {
  return hasStatusField.value && row.status === 'pending'
}

// Edit/Create modal state
type FormMode = { kind: 'create' } | { kind: 'edit'; id: number | string }
const formMode = ref<FormMode | null>(null)
const formData = ref<Record<string, unknown>>({})
const formError = ref('')
const submitting = ref(false)

async function loadMeta() {
  const cols = await api.collections()
  meta.value = cols.data.find((c) => c.name === props.name) || null
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    await loadMeta()
    const r = await api.list(props.name, page.value, size.value)
    rows.value = r.data as Record<string, unknown>[]
    total.value = r.total
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  } finally {
    loading.value = false
  }
}

watch(
  () => props.name,
  () => {
    page.value = 1
    load()
  },
  { immediate: true },
)

function prev() {
  if (page.value > 1) { page.value--; load() }
}
function next() {
  if (page.value < totalPages.value) { page.value++; load() }
}

function formatCell(v: unknown): string {
  if (v === null || v === undefined) return '—'
  if (typeof v === 'object') return JSON.stringify(v)
  return String(v)
}

function openCreate() {
  formMode.value = { kind: 'create' }
  formData.value = {}
  for (const f of editableFields.value) {
    formData.value[f.name] = f.default ?? ''
  }
  formError.value = ''
}

function openEdit(row: Record<string, unknown>) {
  const id = row.id as number | string
  formMode.value = { kind: 'edit', id }
  formData.value = {}
  for (const f of editableFields.value) {
    formData.value[f.name] = row[f.name] ?? ''
  }
  formError.value = ''
}

function closeForm() {
  formMode.value = null
  formError.value = ''
}

async function submitForm() {
  if (!formMode.value) return
  submitting.value = true
  formError.value = ''
  try {
    // Coerce types per field schema
    const payload: Record<string, unknown> = {}
    for (const f of editableFields.value) {
      const v = formData.value[f.name]
      if (v === '' || v === null || v === undefined) continue
      if (f.type === 'int' || f.type === 'belongs_to') {
        payload[f.name] = Number(v)
      } else if (f.type === 'float') {
        payload[f.name] = Number(v)
      } else if (f.type === 'bool') {
        payload[f.name] = Boolean(v)
      } else {
        payload[f.name] = v
      }
    }
    if (formMode.value.kind === 'create') {
      await api.create(props.name, payload)
    } else {
      await api.update(props.name, formMode.value.id, payload)
    }
    closeForm()
    await load()
  } catch (err) {
    formError.value = err instanceof Error ? err.message : String(err)
  } finally {
    submitting.value = false
  }
}

async function doDelete(row: Record<string, unknown>) {
  const id = row.id
  if (id === undefined) return
  const label = row.display_name || row.username || row.name || row.title || id
  if (!confirm(`确定删除 ${props.name} #${id} (${label})?`)) return
  try {
    await api.remove(props.name, id as number | string)
    await load()
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  }
}

async function approveRow(row: Record<string, unknown>) {
  const id = row.id
  if (id === undefined) return
  const label = row.display_name || row.username || id
  if (!confirm(`通过申请: ${label}?`)) return
  try {
    await api.update(props.name, id as number | string, { status: 'active' })
    await load()
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  }
}

async function rejectRow(row: Record<string, unknown>) {
  const id = row.id
  if (id === undefined) return
  const label = row.display_name || row.username || id
  if (!confirm(`拒绝申请: ${label}?`)) return
  try {
    await api.update(props.name, id as number | string, { status: 'rejected' })
    await load()
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  }
}
</script>

<template>
  <div class="collection-view">
    <header class="cv-head">
      <h2>{{ meta?.display || name }}</h2>
      <span class="meta">共 {{ total }} 条</span>
      <span v-if="pendingCount > 0" class="badge-pending">
        {{ pendingCount }} 待审批
      </span>
      <span class="spacer" />
      <button class="btn primary" @click="openCreate" :disabled="!meta">新建</button>
    </header>
    <p v-if="error" class="error">{{ error }}</p>
    <div v-if="loading" class="loading">加载中…</div>
    <div v-else-if="rows.length === 0" class="empty">无数据</div>
    <div v-else class="table-wrap">
      <table class="data-table">
        <thead>
          <tr>
            <th v-for="c in columns" :key="c">{{ c }}</th>
            <th class="actions-col">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(row, i) in rows"
            :key="i"
            :class="{ 'row-pending': isPending(row) }"
          >
            <td v-for="c in columns" :key="c">
              <template v-if="c === 'status' && hasStatusField">
                <span class="status-badge" :data-status="String(row[c])">
                  {{ formatCell(row[c]) }}
                </span>
              </template>
              <template v-else>{{ formatCell(row[c]) }}</template>
            </td>
            <td class="actions-col">
              <template v-if="isPending(row)">
                <button class="link approve" @click="approveRow(row)">通过</button>
                <button class="link reject" @click="rejectRow(row)">拒绝</button>
              </template>
              <button class="link" @click="openEdit(row)">编辑</button>
              <button class="link danger" @click="doDelete(row)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <footer class="pager">
      <button class="link" :disabled="page <= 1" @click="prev">«</button>
      <span>{{ page }} / {{ totalPages }}</span>
      <button class="link" :disabled="page >= totalPages" @click="next">»</button>
    </footer>

    <!-- Edit / Create modal -->
    <div v-if="formMode" class="modal-backdrop" @click.self="closeForm">
      <div class="modal">
        <header class="modal-head">
          <h3>{{ formMode.kind === 'create' ? '新建' : '编辑' }} · {{ meta?.display || name }}</h3>
          <button class="link" @click="closeForm">×</button>
        </header>
        <form class="modal-form" @submit.prevent="submitForm">
          <div v-for="f in editableFields" :key="f.name" class="field">
            <label>
              <span>{{ f.name }}<em v-if="f.required" class="req">*</em></span>
              <span class="field-hint">{{ f.type }}<span v-if="f.target"> → {{ f.target }}</span></span>
            </label>
            <textarea
              v-if="f.type === 'text'"
              v-model="formData[f.name] as string"
              rows="3"
            />
            <input
              v-else-if="f.type === 'bool'"
              type="checkbox"
              :checked="!!formData[f.name]"
              @change="formData[f.name] = ($event.target as HTMLInputElement).checked"
            />
            <input
              v-else-if="f.type === 'int' || f.type === 'float' || f.type === 'belongs_to'"
              v-model="formData[f.name] as number | string"
              type="number"
              :step="f.type === 'float' ? 'any' : '1'"
            />
            <input
              v-else
              v-model="formData[f.name] as string"
              type="text"
              :maxlength="f.max_len || undefined"
            />
          </div>
          <p v-if="formError" class="form-error">{{ formError }}</p>
          <div class="modal-actions">
            <button type="button" class="btn" @click="closeForm">取消</button>
            <button type="submit" class="btn primary" :disabled="submitting">
              {{ submitting ? '保存中…' : '保存' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>
