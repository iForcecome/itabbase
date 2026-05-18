<template>
  <div>
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px">
      <a-typography-text type="secondary">管理数据集合和字段</a-typography-text>
      <a-button type="primary" @click="showCreate = true">
        <PlusOutlined /> 新建集合
      </a-button>
    </div>

    <a-table
      :columns="columns"
      :data-source="tableData"
      :pagination="false"
      row-key="name"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'name'">
          <a @click="openFields(record.name)">{{ record.name }}</a>
        </template>
        <template v-else-if="column.key === 'source'">
          <a-tag :color="sourceColor(record.source)">{{ sourceLabel(record.source) }}</a-tag>
        </template>
        <template v-else-if="column.key === 'fields'">
          {{ record.fields.length }} 个字段
        </template>
        <template v-else-if="column.key === 'actions'">
          <a-space>
            <a-button size="small" @click="openFields(record.name)">字段</a-button>
            <a-popconfirm
              v-if="record.source === 'dynamic'"
              title="确定删除此集合？表和数据都会被删除！"
              ok-text="删除"
              cancel-text="取消"
              @confirm="handleDelete(record.name)"
            >
              <a-button size="small" danger>删除</a-button>
            </a-popconfirm>
          </a-space>
        </template>
      </template>
    </a-table>

    <!-- Fields Drawer -->
    <a-drawer
      v-model:open="showFields"
      :title="`字段配置 — ${fieldsMeta?.display || fieldsName}`"
      width="640"
    >
      <template #extra>
        <a-button
          v-if="fieldsMeta?.source === 'dynamic'"
          type="primary"
          size="small"
          @click="showAddField = true"
        >
          <PlusOutlined /> 添加字段
        </a-button>
      </template>

      <a-table
        :columns="fieldColumns"
        :data-source="fieldsMeta?.fields || []"
        :pagination="false"
        row-key="name"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <a-tag>{{ record.type }}</a-tag>
            <span v-if="record.target" style="color: #999; font-size: 12px"> → {{ record.target }}</span>
          </template>
          <template v-else-if="column.key === 'required'">
            <a-tag v-if="record.required" color="red">必填</a-tag>
            <span v-else style="color: #ccc">—</span>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-popconfirm
              v-if="fieldsMeta?.source === 'dynamic' && record.name !== 'id'"
              title="确定删除此字段？"
              @confirm="handleDeleteField(record.name)"
            >
              <a-button size="small" danger>删除</a-button>
            </a-popconfirm>
            <span v-else style="color: #ccc">—</span>
          </template>
        </template>
      </a-table>
    </a-drawer>

    <!-- Add Field Modal -->
    <a-modal
      v-model:open="showAddField"
      title="添加字段"
      :confirm-loading="adding"
      @ok="handleAddField"
    >
      <a-form :model="fieldForm" layout="vertical" style="margin-top: 16px">
        <a-form-item label="字段名" required>
          <a-input v-model:value="fieldForm.name" placeholder="小写字母+下划线" />
        </a-form-item>
        <a-form-item label="类型" required>
          <a-select v-model:value="fieldForm.type" style="width: 100%">
            <a-select-option v-for="t in fieldTypes" :key="t.value" :value="t.value">
              {{ t.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="显示名">
          <a-input v-model:value="fieldForm.display" />
        </a-form-item>
        <a-form-item v-if="fieldForm.type === 'string'" label="最大长度">
          <a-input-number v-model:value="fieldForm.max_len" :min="0" style="width: 100%" />
        </a-form-item>
        <a-form-item
          v-if="fieldForm.type === 'belongs_to' || fieldForm.type === 'has_many'"
          label="关联集合" required
        >
          <a-input v-model:value="fieldForm.target" placeholder="目标集合名称" />
        </a-form-item>
        <a-form-item v-if="fieldForm.type === 'has_many'" label="外键列名" required>
          <a-input v-model:value="fieldForm.through" placeholder="目标表上的外键列" />
        </a-form-item>
        <a-form-item>
          <a-checkbox v-model:checked="fieldForm.required">必填</a-checkbox>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Create Collection Modal -->
    <a-modal
      v-model:open="showCreate"
      title="新建集合"
      :confirm-loading="creating"
      @ok="handleCreate"
      @cancel="resetCreateForm"
      width="640px"
    >
      <a-form :model="createForm" layout="vertical" style="margin-top: 16px">
        <a-form-item label="集合名称（英文，小写字母+下划线）" required>
          <a-input v-model:value="createForm.name" placeholder="例如: orders, projects" :disabled="creating" />
        </a-form-item>
        <a-form-item label="显示名称">
          <a-input v-model:value="createForm.display" placeholder="例如: 订单, 项目" :disabled="creating" />
        </a-form-item>
        <a-divider>字段</a-divider>
        <div v-for="(f, i) in createForm.fields" :key="i" style="margin-bottom: 8px">
          <a-row :gutter="8">
            <a-col :span="7">
              <a-input v-model:value="f.name" placeholder="字段名" size="small" />
            </a-col>
            <a-col :span="6">
              <a-select v-model:value="f.type" placeholder="类型" size="small" style="width: 100%">
                <a-select-option v-for="t in fieldTypes" :key="t.value" :value="t.value">
                  {{ t.label }}
                </a-select-option>
              </a-select>
            </a-col>
            <a-col :span="5">
              <a-input
                v-if="f.type === 'belongs_to' || f.type === 'has_many'"
                v-model:value="f.target" placeholder="关联集合" size="small"
              />
              <a-input-number
                v-else-if="f.type === 'string'"
                v-model:value="f.max_len" placeholder="最大长度" size="small" style="width: 100%" :min="0"
              />
            </a-col>
            <a-col :span="3">
              <a-checkbox v-model:checked="f.required" size="small">必填</a-checkbox>
            </a-col>
            <a-col :span="3">
              <a-button size="small" danger @click="createForm.fields.splice(i, 1)">
                <DeleteOutlined />
              </a-button>
            </a-col>
          </a-row>
        </div>
        <a-button type="dashed" block @click="addCreateField" style="margin-top: 8px">
          <PlusOutlined /> 添加字段
        </a-button>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { api } from "@/api";
import type { MetaCollection } from "@/api";
import { useUserStore } from "@/stores/user";
import { message } from "ant-design-vue";

const userStore = useUserStore();
const tableData = computed(() => userStore.collections);

const columns = [
  { title: "名称", key: "name", dataIndex: "name" },
  { title: "显示名", dataIndex: "display", key: "display" },
  { title: "来源", key: "source", dataIndex: "source", width: 100 },
  { title: "字段", key: "fields", width: 100 },
  { title: "操作", key: "actions", width: 160 },
];

const fieldColumns = [
  { title: "字段名", dataIndex: "name", key: "name" },
  { title: "类型", key: "type" },
  { title: "必填", key: "required", width: 80 },
  { title: "最大长度", dataIndex: "max_len", key: "max_len", width: 100 },
  { title: "操作", key: "actions", width: 80 },
];

const fieldTypes = [
  { value: "string", label: "文本 (string)" },
  { value: "text", label: "长文本 (text)" },
  { value: "int", label: "整数 (int)" },
  { value: "float", label: "浮点数 (float)" },
  { value: "bool", label: "布尔 (bool)" },
  { value: "datetime", label: "日期时间 (datetime)" },
  { value: "belongs_to", label: "关联 (belongs_to)" },
  { value: "has_many", label: "一对多 (has_many)" },
];

// --- Fields drawer ---
const showFields = ref(false);
const fieldsName = ref("");
const fieldsMeta = computed<MetaCollection | undefined>(() =>
  userStore.collections.find((c) => c.name === fieldsName.value),
);

function openFields(name: string) {
  fieldsName.value = name;
  showFields.value = true;
}

// --- Add field ---
const showAddField = ref(false);
const adding = ref(false);
const fieldForm = reactive({
  name: "", type: "string", display: "", required: false,
  max_len: undefined as number | undefined, target: "", through: "",
});

function resetFieldForm() {
  Object.assign(fieldForm, { name: "", type: "string", display: "", required: false, max_len: undefined, target: "", through: "" });
  showAddField.value = false;
}

async function handleAddField() {
  if (!fieldForm.name || !fieldForm.type) { message.error("字段名和类型不能为空"); return; }
  adding.value = true;
  try {
    await api.addField(fieldsName.value, {
      name: fieldForm.name, type: fieldForm.type,
      display: fieldForm.display || undefined, required: fieldForm.required,
      max_len: fieldForm.type === "string" ? fieldForm.max_len : undefined,
      target: fieldForm.target || undefined, through: fieldForm.through || undefined,
    });
    message.success("字段添加成功");
    await userStore.refreshCollections();
    resetFieldForm();
  } catch (err) { message.error(err instanceof Error ? err.message : "添加失败"); }
  finally { adding.value = false; }
}

async function handleDeleteField(fieldName: string) {
  try {
    await api.deleteField(fieldsName.value, fieldName);
    message.success("字段已删除");
    await userStore.refreshCollections();
  } catch (err) { message.error(err instanceof Error ? err.message : "删除失败"); }
}

// --- Create collection ---
const showCreate = ref(false);
const creating = ref(false);

interface FieldDef { name: string; type: string; required: boolean; max_len?: number; target?: string; through?: string }
const createForm = reactive<{ name: string; display: string; fields: FieldDef[] }>({
  name: "", display: "", fields: [{ name: "", type: "string", required: false }],
});

function addCreateField() { createForm.fields.push({ name: "", type: "string", required: false }); }

function resetCreateForm() {
  createForm.name = ""; createForm.display = "";
  createForm.fields = [{ name: "", type: "string", required: false }];
  showCreate.value = false;
}

async function handleCreate() {
  if (!createForm.name) { message.error("集合名称不能为空"); return; }
  const valid = createForm.fields.filter((f) => f.name && f.type);
  if (valid.length === 0) { message.error("至少需要一个有效字段"); return; }
  creating.value = true;
  try {
    await api.createCollection({
      name: createForm.name, display: createForm.display || undefined,
      fields: valid.map((f) => ({
        name: f.name, type: f.type, required: f.required,
        max_len: f.type === "string" ? f.max_len : undefined,
        target: f.target || undefined, through: f.through || undefined,
      })),
    });
    message.success("集合创建成功");
    await userStore.refreshCollections();
    resetCreateForm();
  } catch (err) { message.error(err instanceof Error ? err.message : "创建失败"); }
  finally { creating.value = false; }
}

async function handleDelete(name: string) {
  try {
    await api.deleteCollection(name);
    message.success("集合已删除");
    await userStore.refreshCollections();
  } catch (err) { message.error(err instanceof Error ? err.message : "删除失败"); }
}

function sourceColor(s: string) { return s === "builtin" ? "orange" : s === "dynamic" ? "blue" : "green"; }
function sourceLabel(s: string) { return s === "builtin" ? "内置" : s === "dynamic" ? "动态" : "代码"; }
</script>
