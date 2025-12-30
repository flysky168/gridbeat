export const useNavigation = () => {
  const menus = [
    {
      label: '实时监控',
      id: 'monitor',
      children: [
        { label: '设备状态', icon: 'i-heroicons-cpu-chip', to: '/monitor/status' },
        { label: '实时数据', icon: 'i-heroicons-chart-bar', to: '/monitor/data' }
      ]
    },
    {
      label: '设备管理',
      id: 'device',
      children: [
        { label: '网关配置', icon: 'i-heroicons-cog-6-tooth', to: '/device/config' },
        { label: '从机列表', icon: 'i-heroicons-list-bullet', to: '/device/list' }
      ]
    }
  ]

  const activeMainMenuId = ref('monitor') // 默认为监控

  return { menus, activeMainMenuId }
}