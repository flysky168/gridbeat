export function useApi() {
  const { token } = useAuth()

  return $fetch.create({
    baseURL: '/api/v1',
    headers: token.value ? { Authorization: `Bearer ${token.value}` } : undefined,
  })
}