import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
import CollectionView from './views/CollectionView.vue'
import Welcome from './views/Welcome.vue'

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/welcome' },
  { path: '/welcome', component: Welcome },
  { path: '/c/:name', component: CollectionView, props: true },
]

export const router = createRouter({
  history: createWebHashHistory(),
  routes,
})
