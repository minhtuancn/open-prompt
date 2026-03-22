function App() {
  return (
    <div className="flex items-center justify-center w-full h-full bg-surface/90 backdrop-blur-md rounded-xl border border-accent/20">
      <div className="flex items-center gap-3 px-4 py-2 w-full">
        <span className="text-accent text-lg">⌘</span>
        <input
          type="text"
          placeholder="Ask anything..."
          className="flex-1 bg-transparent text-white placeholder-white/40 outline-none text-sm"
          autoFocus
        />
      </div>
    </div>
  );
}

export default App;
