import { createRouter, createWebHistory } from "vue-router";
import NProgress from "nprogress";
import routes from "./routes";
import { useUserStore } from "@/stores/user";

NProgress.configure({ showSpinner: false });

function resolveBase(): string {
  const { pathname } = window.location;
  const idx = pathname.indexOf("/admin");
  if (idx >= 0) return pathname.substring(0, idx) + "/admin/";
  return "/";
}

const router = createRouter({ history: createWebHistory(resolveBase()), routes });

router.beforeEach(async (to) => {
  NProgress.start();

  if (to.query.pending === "1" && to.path !== "/pending") {
    return "/pending";
  }

  if (to.path === "/pending") return true;
  if (to.path === "/403" || to.name === "NotFound") return true;

  const userStore = useUserStore();
  if (!userStore.initialized) {
    await userStore.init();
  }

  if (to.path === "/login" && userStore.isLoggedIn) {
    if (!userStore.isAdmin) return "/403";
    return (to.query.redirect as string) || "/";
  }

  if (!userStore.isLoggedIn && to.path !== "/login") {
    return { path: "/login", query: { redirect: to.fullPath } };
  }

  // Logged in but not admin — block access to admin panel
  if (userStore.isLoggedIn && !userStore.isAdmin && to.path !== "/login") {
    return "/403";
  }

  if (to.path === "/welcome") {
    return { path: "/dashboard", replace: true };
  }

  return true;
});

router.afterEach((to) => {
  NProgress.done();
  if (to.meta.title)
    document.title = `${to.meta.title} - ${import.meta.env.VITE_APP_TITLE}`;
});

export default router;
