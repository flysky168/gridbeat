export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const body = await readBody<{ username: string; password: string }>(event)

  if (config.public.apiMode === 'mock') {
    if (body.username === 'root' && body.password === 'admin') return { data: {token: 'mock-token-123' }}
    throw createError({ statusCode: 401, statusMessage: 'Invalid username or password' })
  }

  return await $fetch('/api/v1/auth/login', {
    baseURL: config.upstream,
    method: 'POST',
    body,
  })
})