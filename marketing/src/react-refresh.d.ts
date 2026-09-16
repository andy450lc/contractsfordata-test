declare module 'react-refresh/runtime' {
  interface RefreshRuntime {
    injectIntoGlobalHook(globalObject: Window): void
  }

  const runtime: RefreshRuntime
  export default runtime
}
