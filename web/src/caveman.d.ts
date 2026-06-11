declare module "caveman" {
  const Caveman: {
    options: Record<string, unknown>
    register: (name: string, template: string) => void
    render: (name: string, data?: unknown) => string
  }
  export = Caveman
}
