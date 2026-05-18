import { defineStore } from "pinia";
import { ref, computed } from "vue";
import {
  api,
  UnauthenticatedError,
  type User,
  type MetaCollection,
} from "@/api";

export const useUserStore = defineStore("user", () => {
  const user = ref<User | null>(null);
  const collections = ref<MetaCollection[]>([]);
  const initialized = ref(false);

  const isLoggedIn = computed(() => !!user.value);
  const isAdmin = computed(
    () => user.value?.roles?.includes("admin") ?? false,
  );

  const dynamicCollections = computed(() =>
    collections.value.filter((c) => c.source === "dynamic"),
  );
  const allVisibleCollections = computed(() =>
    collections.value.filter((c) => !c.internal),
  );

  async function init() {
    try {
      const me = await api.whoami();
      user.value = me.data;
      const cols = await api.collections();
      collections.value = cols.data;
    } catch (err) {
      if (err instanceof UnauthenticatedError) {
        user.value = null;
      } else {
        throw err;
      }
    } finally {
      initialized.value = true;
    }
  }

  async function refreshCollections() {
    const cols = await api.collections();
    collections.value = cols.data;
  }

  async function logout() {
    await api.logout();
    user.value = null;
    collections.value = [];
    initialized.value = false;
  }

  return {
    user,
    collections,
    dynamicCollections,
    allVisibleCollections,
    initialized,
    isLoggedIn,
    isAdmin,
    init,
    refreshCollections,
    logout,
  };
});
