<template>
  <div class="h-screen flex flex-col overflow-hidden">
    <header class="h-16 border-b border-gray-200 dark:border-gray-800 flex items-center justify-between px-4 bg-white dark:bg-gray-900 z-10">
      <div class="flex items-center gap-8">
        <div class="font-bold text-xl text-primary">工业网关 Pro</div>
        
        <UHorizontalNavigation :links="mainNavLinks" />
      </div>

      <div class="flex items-center gap-4">
        <USelectMenu v-model="selectedLang" :options="['中文', 'English']" />
        <UButton color="gray" variant="ghost" icon="i-heroicons-arrow-left-on-rectangle" @click="handleLogout" />
      </div>
    </header>

    <div class="flex-1 flex overflow-hidden">
      <aside class="w-64 border-r border-gray-200 dark:border-gray-800 bg-gray-50/50 dark:bg-gray-900/50 p-4">
        <p class="text-xs font-semibold text-gray-400 mb-4 uppercase tracking-wider">子菜单</p>
        <UVerticalNavigation :links="sideNavLinks" />
      </aside>

      <main class="flex-1 overflow-y-auto p-6 bg-gray-50 dark:bg-black">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup>
const { menus, activeMainMenuId } = useNavigation()

// 映射主菜单
const mainNavLinks = menus.map(m => ({
  label: m.label,
  click: () => { activeMainMenuId.value = m.id }
}))

// 动态获取当前侧边栏菜单
const sideNavLinks = computed(() => {
  const current = menus.find(m => m.id === activeMainMenuId.value)
  return current ? current.children : []
})

const selectedLang = ref('中文')
const handleLogout = () => { /* 退出逻辑 */ }
</script>