# Engine JSON-RPC API

Go Engine giao tiếp qua Unix socket sử dụng JSON-RPC 2.0.

## Auth

| Method | Params | Mô tả |
|--------|--------|-------|
| `auth.register` | `username, password, display_name` | Tạo tài khoản |
| `auth.login` | `username, password` | Đăng nhập, trả về JWT token |
| `auth.me` | `token` | Thông tin user hiện tại |
| `auth.is_first_run` | — | Kiểm tra lần chạy đầu |

## Query

| Method | Params | Mô tả |
|--------|--------|-------|
| `query.stream` | `token, input, provider?, model?, conversation_id?` | Stream response từ AI provider |

## Providers

| Method | Params | Mô tả |
|--------|--------|-------|
| `providers.list` | `token` | Danh sách providers |
| `providers.detect` | `token` | Auto-detect providers |
| `providers.connect` | `token, provider_id, api_key` | Kết nối provider |
| `providers.validate` | `token, provider_id` | Validate API key |
| `providers.remove` | `token, provider_id` | Xoá provider |
| `providers.add_gateway` | `token, name, base_url, api_key?, models?` | Thêm gateway |
| `providers.set_priority` | `token, priorities[]` | Đặt thứ tự ưu tiên |
| `providers.oauth_start` | `token, provider_id` | Bắt đầu OAuth flow |
| `providers.oauth_finish` | `token, provider_id, code, state` | Hoàn tất OAuth |
| `providers.oauth_poll` | `token, provider_id, device_code` | Poll Device Flow |

## Prompts

| Method | Params | Mô tả |
|--------|--------|-------|
| `prompts.list` | `token` | Danh sách prompts |
| `prompts.create` | `token, title, content, category?, tags?, is_slash?, slash_name?` | Tạo prompt |
| `prompts.update` | `token, id, ...fields` | Cập nhật prompt |
| `prompts.delete` | `token, id` | Xoá prompt |
| `prompts.export` | `token` | Export tất cả prompts (JSON) |
| `prompts.import` | `token, prompts[]` | Import prompts |

## Skills

| Method | Params | Mô tả |
|--------|--------|-------|
| `skills.list` | `token` | Danh sách skills |
| `skills.create` | `token, name, prompt_text, model?, provider?` | Tạo skill |
| `skills.update` | `token, id, ...fields` | Cập nhật skill |
| `skills.delete` | `token, id` | Xoá skill |

## History

| Method | Params | Mô tả |
|--------|--------|-------|
| `history.list` | `token, limit?, offset?` | Lịch sử queries |
| `history.search` | `token, query, limit?` | Tìm kiếm history |

## Conversations

| Method | Params | Mô tả |
|--------|--------|-------|
| `conversations.list` | `token` | Danh sách conversations |
| `conversations.create` | `token, title?` | Tạo conversation |
| `conversations.messages` | `token, conversation_id` | Lấy messages |
| `conversations.delete` | `token, conversation_id` | Xoá conversation |

## Analytics

| Method | Params | Mô tả |
|--------|--------|-------|
| `analytics.summary` | `token` | Tổng quan usage |
| `analytics.by_provider` | `token` | Usage theo provider |
| `analytics.aggregate` | `token` | Rollup history → usage_daily |
| `analytics.daily` | `token, days?` | Daily usage stats |

## Plugins

| Method | Params | Mô tả |
|--------|--------|-------|
| `plugins.list` | `token` | Danh sách plugins |
| `plugins.install` | `token, name, type, config?` | Cài plugin |
| `plugins.toggle` | `token, id, enabled` | Bật/tắt plugin |
| `plugins.uninstall` | `token, id` | Gỡ plugin |

## Settings

| Method | Params | Mô tả |
|--------|--------|-------|
| `settings.get` | `token, key` | Lấy setting |
| `settings.set` | `token, key, value` | Đặt setting |

## Health

| Method | Params | Mô tả |
|--------|--------|-------|
| `health.check` | `token` | Kiểm tra provider health |
