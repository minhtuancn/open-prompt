export function HotkeyTab() {
  return (
    <div className="flex flex-col gap-4">
      <p className="text-xs text-white/40">Hotkey được cấu hình trong Tauri và áp dụng khi khởi động ứng dụng.</p>
      <div className="bg-white/5 border border-white/10 rounded-xl p-4">
        <div className="text-xs text-white/40 mb-2">Hotkey hiện tại</div>
        <div className="flex items-center gap-2">
          <kbd className="px-3 py-1.5 bg-white/10 border border-white/20 rounded-lg text-sm text-white font-mono">Ctrl</kbd>
          <span className="text-white/40">+</span>
          <kbd className="px-3 py-1.5 bg-white/10 border border-white/20 rounded-lg text-sm text-white font-mono">Space</kbd>
        </div>
        <p className="text-xs text-white/30 mt-3">Để thay đổi, sửa <code className="text-indigo-400">tauri.conf.json</code> và rebuild.</p>
      </div>
    </div>
  )
}
