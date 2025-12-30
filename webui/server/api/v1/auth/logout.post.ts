import { defineEventHandler, getCookie, deleteCookie, setResponseStatus } from 'h3'

/**
 * POST /api/v1/auth/logout
 * For @sidebase/nuxt-auth (local provider) signOut endpoint.
 */
export default defineEventHandler(async (event) => {
  // 1) 取出当前 token（如你要做服务端撤销/审计可用）
  const accessToken =
    getCookie(event, 'auth:token') ||
    getCookie(event, 'auth.token') ||
    null

  const refreshToken =
    getCookie(event, 'auth:refresh-token') ||
    getCookie(event, 'auth.refresh-token') ||
    null

  // 2) 可选：撤销 refresh token（如果你服务端有存 refresh token / 黑名单机制）
  // await revokeRefreshToken(refreshToken)

  // 3) 删除 nuxt-auth local provider 常见 cookie
  // ⚠️ 删除 cookie 需要 path 与当初设置时一致；一般是 '/'
  const cookieNames = [
    'auth:token',
    'auth:refresh-token',
    'auth:data',
    'auth:raw-token',
    // 兼容有人用点号命名的情况（项目里若你改过 key）
    'auth.token',
    'auth.refresh-token',
    'auth.data',
    'auth.raw-token'
  ]

  for (const name of cookieNames) {
    deleteCookie(event, name, { path: '/' })
  }

  // 4) 返回结果（不要在这里 redirect，交给 signOut({ callbackUrl })）
  setResponseStatus(event, 200)

  return {
    ok: true,
    // 方便你调试：生产可删除
    revoked: Boolean(refreshToken),
    hadAccessToken: Boolean(accessToken)
  }
})

/**
 * Example revocation hook (implement yourself)
 * - DB delete/mark refresh token as revoked
 * - Or call external auth service to revoke
 */
// async function revokeRefreshToken(token: string | null) {
//   if (!token) return
//   // TODO: your revoke logic
// }