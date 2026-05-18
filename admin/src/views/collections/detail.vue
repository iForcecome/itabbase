<template>
  <div>
    <a-page-header
      :title="meta?.display || name"
      :sub-title="`集合: ${name}`"
      :ghost="false"
      style="margin-bottom: 16px"
      @back="$router.push('/collections')"
    >
      <template #extra>
        <a-button
          v-if="isDynamic"
          type="primary"
          @click="showAddField = true"
        >
          <PlusOutlined /> 添加字段
        </a-button>
        <a-button @click="$router.push(`/c/${name}`)">
          查看数据
        </a-button>
      </template>
      <template #tags>
        <a-tag :color="meta?.source === 'dynamic' ? 'blue' : 'orange'">
          {{ meta?.source === 'dynamic' ? '动态' : meta?.source === 'builtin' ? '内置' : '代码' }}
        </a-tag>
      </template>
    </a-page-header>

    <a-card title="字段列表">
      <a-table
        :columns="fieldColumns"
        :data-source="meta?.fields || []"
        :pagination="false"
        row-key="name"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <a-tag>{{ record.type }}</a-tag>
            <span v-if="record.target" style="color: #999; font-size: 12px">
              → {{ record.target }}
            </span>
          </template>
          <template v-else-if="column.key === 'required'">
            <a-tag v-if="record.required" color="red">必填</a-tag>
            <span v-else style="color: #ccc">—</span>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-popconfirm
              v-if="isDynamic && record.name !== 'id'"
              title="确定删除此字段？（注意：数据库列不会被删除）"
              @confirm="handleDeleteField(record.name)"
            >
              <a-button size="small" danger>删除</a-button>
            </a-popconfirm>
            <span v-else style="color: #ccc">—</span>
          </template>
        </template>
      </a-table>
    </a-card>

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
          label="关联集合"
          required
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
  </div>
</template>

<script setup lang="ts">
import { api } from "@/api";
import type { MetaCollection } from "@/api";
import { useUserStore } from "@/stores/user";
import { message } from "ant-design-vue";

const props = defineProps<{ name: string }>();
const userStore = useUserStore();

const meta = computed<MetaCollection | undefined>(() =>
  userStore.collections.find((c) => c.name === props.name),
);
const isDynamic = computed(() => meta.value?.source === "dynamic");

const showAddField = ref(false);
const adding = ref(false);

const fieldColumns = [
  { title: "字段名", dataIndex: "name", key: "name" },
  { title: "类型", key: "type" },
  { title: "必填", key: "required", width: 80 },
  { title: "最大长度", dataIndex: "max_len", key: "max_len", width: 100 },
  { title: "默认值", dataIndex: "default", key: "default", width: 120 },
  { title: "操作", key: "actions", width: 100 },
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

const fieldForm = reactive({
  name: "",
  type: "string",
  display: "",
  required: false,
  max_len: undefined as number | undefined,
  target: "",
  through: "",
});

function resetFieldForm() {
  fieldForm.name = "";
  fieldForm.type = "string";
  fieldForm.display = "";
  fieldForm.required = false;
  fieldForm.max_len = undefined;
  fieldForm.target = "";
  fieldForm.through = "";
  showAddField.value = false;
}

async function handleAddField() {
  if (!fieldForm.name || !fieldForm.type) {
    message.error("字段名和类型不能为空");
    return;
  }
  adding.value = true;
  try {
    await api.addField(props.name, {
      name: fieldForm.name,
      type: fieldForm.type,
      display: fieldForm.display || undefined,
      required: fieldForm.required,
      max_len: fieldForm.type === "string" ? fieldForm.max_len : undefined,
      target: fieldForm.target || undefined,
      through: fieldForm.through || undefined,
    });
    message.success("字段添加成功");
    await userStore.refreshCollections();
    resetFieldForm();
  } catch (err) {
    message.error(err instanceof Error ? err.message : "添加失败");
  } finally {
    adding.value = false;
  }
}

async function handleDeleteField(fieldName: string) {
  try {
    await api.deleteField(props.name, fieldName);
    message.success("字段已删除");
    await userStore.refreshCollections();
  } catch (err) {
    message.error(err instanceof Error ? err.message : "删除失败");
  }
}
</script>
