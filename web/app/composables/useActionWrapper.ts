export function useActionWrapper() {
  const toast = useToast()

  async function wrapAction(
    fn: () => Promise<void>,
    opts: { success?: string, error?: string } = {}
  ) {
    const { success = 'Done', error = 'Action failed' } = opts
    try {
      await fn()
      toast.add({ title: success, color: 'success' })
    } catch {
      toast.add({ title: error, color: 'error' })
    }
  }

  return { wrapAction }
}
