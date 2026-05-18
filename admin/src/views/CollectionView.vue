<script setup lang="ts">
import { api, type MetaCollection, type MetaField } from "@/api";
import { useUserStore } from "@/stores/user";
import { message } from "ant-design-vue";

const props = defineProps<{ name: string }>();
const userStore = useUserStore();

const rows = ref<Record<string, unknown>[]>([]);
const total = ref(0);
const page = ref(1);
const size = ref(20);
const loading = ref(false);
const error = ref("");
const sortField = ref("");
const sortOrder = ref<"ascend" | "descend" | null>(null);

const HIDDEN_FIELDS = new Set(["password_hash"]);

const meta = computed<MetaCollection | undefined>(() =>
  userStore.collections.find((c) => c.name === props.name),
);

const visibleFields = computed<MetaField[]>(() => {
  if (!meta.value) return [];
  return meta.value.fields.filter(
    (f) => !HIDDEN_FIELDS.has(f.name) && f.type !== "has_many",
  );
});

const editableFields = computed<MetaField[]>(() => {
  if (!meta.value) return [];
  return meta.value.fields.filter(
    (f) =>
      !HIDDEN_FIELDS.has(f.name) && f.name !== "id" && f.type !== "has_many",
  );
});

const hasStatusField = computed(
  () => meta.value?.fields.some((f) => f.name === "status") ?? false,
);
const pendingCount = computed(
  () => rows.value.filter((r) => r.status === "pending").length,
);
function isPending(row: Record<string, unknown>): boolean {
  return hasStatusField.value && row.status === "pending";
}

function getFieldType(fieldName: string): string | undefined {
  return meta.value?.fields.find((f) => f.name === fieldName)?.type;
}

const tableColumns = computed(() => {
  const idCol = {
    title: "ID",
    dataIndex: "id",
    key: "id",
    width: 70,
    sorter: true,
    sortOrder: sortField.value === "id" ? sortOrder.value : undefined,
  };
  const fieldCols = visibleFields.value.map((f) => {
    const col: Record<string, unknown> = {
      title: fieldLabel(f),
      dataIndex: f.name,
      key: f.name,
      ellipsis: true,
    };
    if (f.type !== "belongs_to") {
      col.sorter = true;
      col.sortOrder = sortField.value === f.name ? sortOrder.value : undefined;
    }
    if (f.type === "bool") col.width = 80;
    if (f.type === "datetime") col.width = 180;
    return col;
  });
  return [
    idCol,
    ...fieldCols,
    { title: "操作", key: "actions", fixed: "right" as const, width: 220 },
  ];
});

const tablePagination = computed(() => ({
  current: page.value,
  pageSize: size.value,
  total: total.value,
  showSizeChanger: true,
  showTotal: (t: number) => `共 ${t} 条`,
  pageSizeOptions: ["10", "20", "50", "100"],
}));

type FormMode = { kind: "create" } | { kind: "edit"; id: number | string };
const formMode = ref<FormMode | null>(null);
const formData = ref<Record<string, unknown>>({});
const formError = ref("");
const submitting = ref(false);

async function load() {
  loading.value = true;
  error.value = "";
  try {
    let params = "";
    if (sortField.value && sortOrder.value) {
      const prefix = sortOrder.value === "descend" ? "-" : "";
      params = `sort=${prefix}${sortField.value}`;
    }
    const r = await api.list(props.name, page.value, size.value, params);
    rows.value = r.data as Record<string, unknown>[];
    total.value = r.total;
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

watch(
  () => props.name,
  () => {
    page.value = 1;
    sortField.value = "";
    sortOrder.value = null;
    load();
  },
  { immediate: true },
);

function handleTableChange(
  pag: { current?: number; pageSize?: number },
  _filters: any,
  sorter: any,
) {
  page.value = pag.current ?? 1;
  size.value = pag.pageSize ?? 20;
  const s = Array.isArray(sorter) ? sorter[0] : sorter;
  if (s?.field) {
    sortField.value = String(s.field);
    sortOrder.value = s.order ?? null;
  } else {
    sortField.value = "";
    sortOrder.value = null;
  }
  load();
}

function fieldLabel(f: MetaField): string {
  return (f as any).display || f.name;
}

function formatCell(v: unknown, type?: string): string {
  if (v === null || v === undefined) return "—";
  if (type === "bool") return v ? "是" : "否";
  if (type === "datetime" && typeof v === "string") {
    try {
      return new Date(v).toLocaleString("zh-CN");
    } catch {
      return v;
    }
  }
  if (typeof v === "object") return JSON.stringify(v);
  return String(v);
}

function statusColor(status: unknown): string {
  switch (status) {
    case "pending":
      return "orange";
    case "active":
      return "green";
    case "rejected":
      return "red";
    default:
      return "default";
  }
}

function rowLabel(row: Record<string, unknown>): string {
  return String(
    row.display_name || row.username || row.name || row.title || row.id || "",
  );
}

function openCreate() {
  formMode.value = { kind: "create" };
  formData.value = {};
  for (const f of editableFields.value) {
    if (f.type === "bool") {
      formData.value[f.name] = f.default ?? false;
    } else if (
      f.type === "int" ||
      f.type === "float" ||
      f.type === "belongs_to"
    ) {
      formData.value[f.name] = f.default ?? null;
    } else {
      formData.value[f.name] = f.default ?? "";
    }
  }
  formError.value = "";
}

function openEdit(row: Record<string, unknown>) {
  const id = row.id as number | string;
  formMode.value = { kind: "edit", id };
  formData.value = {};
  for (const f of editableFields.value) {
    formData.value[f.name] = row[f.name] ?? "";
  }
  formError.value = "";
}

function closeForm() {
  formMode.value = null;
  formError.value = "";
}

async function submitForm() {
  if (!formMode.value) return;
  submitting.value = true;
  formError.value = "";
  try {
    const payload: Record<string, unknown> = {};
    for (const f of editableFields.value) {
      const v = formData.value[f.name];
      if (v === "" || v === null || v === undefined) continue;
      if (f.type === "int" || f.type === "belongs_to") {
        payload[f.name] = Number(v);
      } else if (f.type === "float") {
        payload[f.name] = Number(v);
      } else if (f.type === "bool") {
        payload[f.name] = Boolean(v);
      } else {
        payload[f.name] = v;
      }
    }
    if (formMode.value.kind === "create") {
      await api.create(props.name, payload);
      message.success("创建成功");
    } else {
      await api.update(props.name, formMode.value.id, payload);
      message.success("更新成功");
    }
    closeForm();
    await load();
  } catch (err) {
    formError.value = err instanceof Error ? err.message : String(err);
  } finally {
    submitting.value = false;
  }
}

async function doDelete(row: Record<string, unknown>) {
  const id = row.id;
  if (id === undefined) return;
  try {
    await api.remove(props.name, id as number | string);
    message.success("删除成功");
    await load();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}

async function approveRow(row: Record<string, unknown>) {
  const id = row.id;
  if (id === undefined) return;
  try {
    await api.update(props.name, id as number | string, { status: "active" });
    message.success("已批准");
    await load();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}

async function rejectRow(row: Record<string, unknown>) {
  const id = row.id;
  if (id === undefined) return;
  try {
    await api.update(props.name, id as number | string, {
      status: "rejected",
    });
    message.success("已拒绝");
    await load();
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}
</script>

<template>
  <div>
    <div
      style="
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 16px;
      "
    >
      <a-typography-title :level="4" style="margin: 0">
        {{ meta?.display || name }}
      </a-typography-title>
      <a-tag color="blue">共 {{ total }} 条</a-tag>
      <a-badge
        v-if="pendingCount > 0"
        :count="pendingCount"
        :overflow-count="99"
      >
        <a-tag color="orange">待审批</a-tag>
      </a-badge>
      <div style="flex: 1" />
      <a-button type="primary" :disabled="!meta" @click="openCreate">
        <PlusOutlined /> 新建
      </a-button>
    </div>

    <a-alert
      v-if="error"
      type="error"
      :message="error"
      show-icon
      closable
      style="margin-bottom: 16px"
      @close="error = ''"
    />

    <a-table
      :columns="tableColumns"
      :data-source="rows"
      :loading="loading"
      :pagination="tablePagination"
      :scroll="{ x: 'max-content' }"
      :row-class-name="
        (record: Record<string, unknown>) =>
          isPending(record) ? 'row-pending' : ''
      "
      row-key="id"
      size="middle"
      @change="handleTableChange"
    >
      <template #bodyCell="{ column, record, text }">
        <template v-if="column.key === 'status' && hasStatusField">
          <a-tag :color="statusColor(text)">{{ formatCell(text) }}</a-tag>
        </template>
        <template v-else-if="column.key === 'actions'">
          <a-space :size="0">
            <template v-if="isPending(record)">
              <a-popconfirm
                :title="`通过申请: ${rowLabel(record)}?`"
                ok-text="确定"
                cancel-text="取消"
                @confirm="approveRow(record)"
              >
                <a-button
                  type="link"
                  size="small"
                  style="color: #52c41a; padding: 0 6px"
                >
                  通过
                </a-button>
              </a-popconfirm>
              <a-popconfirm
                :title="`拒绝申请: ${rowLabel(record)}?`"
                ok-text="确定"
                cancel-text="取消"
                @confirm="rejectRow(record)"
              >
                <a-button
                  type="link"
                  size="small"
                  style="color: #fa8c16; padding: 0 6px"
                >
                  拒绝
                </a-button>
              </a-popconfirm>
            </template>
            <a-button
              type="link"
              size="small"
              style="padding: 0 6px"
              @click="openEdit(record)"
            >
              编辑
            </a-button>
            <a-popconfirm
              :title="`确定删除 ${name} #${record.id} (${rowLabel(record)})?`"
              ok-text="确定"
              cancel-text="取消"
              @confirm="doDelete(record)"
            >
              <a-button type="link" danger size="small" style="padding: 0 6px">
                删除
              </a-button>
            </a-popconfirm>
          </a-space>
        </template>
        <template v-else-if="column.key === 'id'">
          {{ text }}
        </template>
        <template v-else>
          <template v-if="getFieldType(column.key as string) === 'bool'">
            <a-tag :color="text ? 'green' : 'default'">{{
              text ? "是" : "否"
            }}</a-tag>
          </template>
          <template
            v-else-if="getFieldType(column.key as string) === 'datetime'"
          >
            {{ formatCell(text, "datetime") }}
          </template>
          <template v-else>
            {{ formatCell(text) }}
          </template>
        </template>
      </template>
    </a-table>

    <a-modal
      :open="!!formMode"
      :title="
        (formMode?.kind === 'create' ? '新建' : '编辑') +
        ' · ' +
        (meta?.display || name)
      "
      :confirm-loading="submitting"
      ok-text="保存"
      cancel-text="取消"
      :destroy-on-close="true"
      @ok="submitForm"
      @cancel="closeForm"
    >
      <a-form :model="formData" layout="vertical" style="margin-top: 16px">
        <a-form-item
          v-for="f in editableFields"
          :key="f.name"
          :label="fieldLabel(f)"
          :required="f.required"
        >
          <template #extra>
            {{ f.type }}{{ f.target ? ` → ${f.target}` : "" }}
          </template>
          <a-textarea
            v-if="f.type === 'text'"
            :value="(formData[f.name] as string)"
            @update:value="(val: any) => (formData[f.name] = val)"
            :rows="3"
          />
          <a-switch
            v-else-if="f.type === 'bool'"
            :checked="!!formData[f.name]"
            @update:checked="(val: any) => (formData[f.name] = !!val)"
          />
          <a-input-number
            v-else-if="
              f.type === 'int' || f.type === 'float' || f.type === 'belongs_to'
            "
            :value="formData[f.name] as number"
            @update:value="(val: any) => (formData[f.name] = val)"
            :step="f.type === 'float' ? 0.01 : 1"
            style="width: 100%"
          />
          <a-date-picker
            v-else-if="f.type === 'datetime'"
            :value="(formData[f.name] as any)"
            @update:value="(val: any) => (formData[f.name] = val)"
            show-time
            style="width: 100%"
            value-format="YYYY-MM-DD HH:mm:ss"
          />
          <a-input
            v-else
            :value="(formData[f.name] as string)"
            @update:value="(val: any) => (formData[f.name] = val)"
            :maxlength="f.max_len || undefined"
          />
        </a-form-item>
      </a-form>
      <a-alert v-if="formError" type="error" :message="formError" show-icon />
    </a-modal>
  </div>
</template>

<style scoped>
:deep(.row-pending) {
  background-color: #fffbe6;
}
:deep(.row-pending:hover > td) {
  background-color: #fff7d6 !important;
}
</style>
