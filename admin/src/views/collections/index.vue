<template>
  <div>
    <a-page-header title="集合配置" sub-title="管理数据集合和字段" :ghost="false" style="margin-bottom: 16px">
      <template #extra>
        <a-button type="primary" @click="showCreate = true">
          <PlusOutlined /> 新建集合
        </a-button>
      </template>
    </a-page-header>

    <a-card>
      <a-table
        :columns="columns"
        :data-source="tableData"
        :pagination="false"
        row-key="name"
        :loading="loading"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <a @click="$router.push(`/collections/${record.name}`)">
              {{ record.name }}
            </a>
          </template>
          <template v-else-if="column.key === 'source'">
            <a-tag :color="sourceColor(record.source)">{{ sourceLabel(record.source) }}</a-tag>
          </template>
          <template v-else-if="column.key === 'fields'">
            {{ record.fields.length }} 个字段
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button
                size="small"
                @click="$router.push(`/collections/${record.name}`)"
              >
                配置字段
              </a-button>
              <a-button
                size="small"
                @click="$router.push(`/c/${record.name}`)"
                :disabled="record.internal"
              >
                查看数据
              </a-button>
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
    </a-card>

    <!-- Create Modal -->
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
          <a-input
            v-model:value="createForm.name"
            placeholder="例如: orders, projects"
            :disabled="creating"
          />
        </a-form-item>
        <a-form-item label="显示名称">
          <a-input
            v-model:value="createForm.display"
            placeholder="例如: 订单, 项目"
            :disabled="creating"
          />
        </a-form-item>

        <a-divider>字段</a-divider>

        <div v-for="(f, i) in createForm.fields" :key="i" class="field-row">
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
                v-model:value="f.target"
                placeholder="关联集合"
                size="small"
              />
              <a-input-number
                v-else-if="f.type === 'string'"
                v-model:value="f.max_len"
                placeholder="最大长度"
                size="small"
                style="width: 100%"
                :min="0"
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

        <a-button type="dashed" block @click="addField" style="margin-top: 8px">
          <PlusOutlined /> 添加字段
        </a-button>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { api } from "@/api";
import { useUserStore } from "@/stores/user";
import { message } from "ant-design-vue";

const userStore = useUserStore();
const loading = ref(false);
const showCreate = ref(false);
const creating = ref(false);

const tableData = computed(() => userStore.collections);

const columns = [
  { title: "名称", key: "name", dataIndex: "name" },
  { title: "显示名", dataIndex: "display", key: "display" },
  { title: "来源", key: "source", dataIndex: "source", width: 100 },
  { title: "字段", key: "fields", width: 100 },
  { title: "操作", key: "actions", width: 280 },
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

interface FieldForm {
  name: string;
  type: string;
  required: boolean;
  max_len?: number;
  target?: string;
  through?: string;
}

const createForm = reactive<{
  name: string;
  display: string;
  fields: FieldForm[];
}>({
  name: "",
  display: "",
  fields: [{ name: "", type: "string", required: false }],
});

function addField() {
  createForm.fields.push({ name: "", type: "string", required: false });
}

function resetCreateForm() {
  createForm.name = "";
  createForm.display = "";
  createForm.fields = [{ name: "", type: "string", required: false }];
  showCreate.value = false;
}

async function handleCreate() {
  if (!createForm.name) {
    message.error("集合名称不能为空");
    return;
  }
  const validFields = createForm.fields.filter((f) => f.name && f.type);
  if (validFields.length === 0) {
    message.error("至少需要一个有效字段");
    return;
  }

  creating.value = true;
  try {
    await api.createCollection({
      name: createForm.name,
      display: createForm.display || undefined,
      fields: validFields.map((f) => ({
        name: f.name,
        type: f.type,
        required: f.required,
        max_len: f.type === "string" ? f.max_len : undefined,
        target: f.target || undefined,
        through: f.through || undefined,
      })),
    });
    message.success("集合创建成功");
    await userStore.refreshCollections();
    resetCreateForm();
  } catch (err) {
    message.error(err instanceof Error ? err.message : "创建失败");
  } finally {
    creating.value = false;
  }
}

async function handleDelete(name: string) {
  try {
    await api.deleteCollection(name);
    message.success("集合已删除");
    await userStore.refreshCollections();
  } catch (err) {
    message.error(err instanceof Error ? err.message : "删除失败");
  }
}

function sourceColor(source: string) {
  if (source === "builtin") return "orange";
  if (source === "dynamic") return "blue";
  return "green";
}

function sourceLabel(source: string) {
  if (source === "builtin") return "内置";
  if (source === "dynamic") return "动态";
  return "代码";
}
</script>

<style scoped>
.field-row {
  margin-bottom: 8px;
}
</style>
