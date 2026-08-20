/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package middleware

import (
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/utils"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

var skipAuthPaths = map[string]bool{
	"/api/user/login":                  true,
	"/api/user/logout":                 true,
	"/api/user/refresh_token":          true,
	"/api/user/signup":                 true,
	"/api/user/profile":                true,
	"/api/user/codes":                  true,
	"/api/not_auth/getBindIps":         true,
	"/api/not_auth/getTreeNodeBindIps": true,
	"/api/public/visitor/submit":       true,
	"/api/public/visitor/upload":       true,
	"/public/visitor":                  true,
	"/favicon.ico":                     true,
}

// 静态资源和WebSocket路径前缀
var skipPrefixes = []string{
	"/api/ai/chat/ws",
	"/api/tree/local/terminal",
	"/public/",
}

// 登录后即可访问、不再走角色 API 授权的前缀（仅本人数据）
var skipPermissionPrefixes = []string{
	"/api/workorder/notification/inbox/",
}

type AuthMiddleware struct {
	roleService service.RoleService
}

func NewAuthMiddleware(roleService service.RoleService) *AuthMiddleware {
	return &AuthMiddleware{
		roleService: roleService,
	}
}

func hasPrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func (am *AuthMiddleware) CheckAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if skipAuthPaths[path] {
			c.Next()
			return
		}

		// 跳过静态资源和WebSocket路径
		if path == "/" || hasPrefix(path, skipPrefixes) || strings.Contains(path, "/exec") {
			c.Next()
			return
		}

		// 获取用户信息（兼容nginx代理）
		userVal, exists := c.Get("user")
		if !exists {
			// 兼容未登录时放行登录接口
			if skipAuthPaths[path] {
				c.Next()
				return
			}
			base.ForbiddenError(c, "未登录或登录已过期")
			c.Abort()
			return
		}
		user, ok := userVal.(jwt.UserClaims)
		if !ok {
			base.ForbiddenError(c, "用户信息异常")
			c.Abort()
			return
		}

		// 管理员放行
		if user.Username == "admin" {
			c.Next()
			return
		}

		// 服务账号放行
		if user.AccountType == 2 {
			c.Next()
			return
		}

		if hasPrefix(path, skipPermissionPrefixes) {
			c.Next()
			return
		}

		// 获取HTTP方法代码
		methodCode, ok := utils.MethodCode(c.Request.Method)
		if !ok {
			base.ErrorWithMessage(c, "不支持的HTTP方法")
			c.Abort()
			return
		}

		roles, err := am.roleService.GetUserRoles(c, user.Uid)
		if err != nil {
			base.ErrorWithMessage(c, "获取用户角色失败")
			c.Abort()
			return
		}

		for _, role := range roles.Items {
			// 跳过禁用角色
			if role.Status != 1 {
				continue
			}

			// 检查API权限
			for _, api := range role.Apis {
				if utils.MatchAPIPath(api.Path, path, methodCode, api.Method) {
					c.Next()
					return
				}
			}
		}

		// 无权限访问
		base.ForbiddenError(c, "无权限访问该接口")
		c.Abort()
	}
}
