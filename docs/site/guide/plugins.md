# Plugins

## Loại plugin

| Type | Mô tả |
|------|-------|
| `provider` | Thêm AI provider mới |
| `skill` | Thêm skill tuỳ chỉnh |
| `formatter` | Thêm output formatter |

## Quản lý plugins

### Cài đặt
```
Settings → Plugins → Install
```

Nhập tên plugin, chọn type, nhấn **Install**.

### Bật/tắt
Toggle ON/OFF trong danh sách plugins.

### Gỡ cài đặt
Nhấn **Uninstall** cạnh plugin muốn gỡ.

## API

```json
// Liệt kê plugins
{"method": "plugins.list", "params": {"token": "..."}}

// Cài đặt
{"method": "plugins.install", "params": {"token": "...", "name": "my-plugin", "type": "skill", "config": {}}}

// Bật/tắt
{"method": "plugins.toggle", "params": {"token": "...", "id": 1, "enabled": true}}

// Gỡ
{"method": "plugins.uninstall", "params": {"token": "...", "id": 1}}
```
