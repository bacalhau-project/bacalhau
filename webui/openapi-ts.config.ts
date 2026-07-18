import { defineConfig } from '@hey-api/openapi-ts'

export default defineConfig({
  input: 'lib/api/schema/swagger.json',
  output: {
    path: 'lib/api/generated',
    postProcess: ['prettier'],
  },
  plugins: [
    '@hey-api/client-fetch',
    {
      name: '@hey-api/typescript',
      enums: {
        mode: 'typescript',
        case: 'PascalCase',
      },
      definitions: {
        name: (name) => name.replaceAll('.', '_').replaceAll('-', '_'),
        case: 'preserve',
      },
    },
    {
      name: '@hey-api/sdk',
      operations: {
        strategy: 'byTags',
        methods: 'static',
        nesting: (operation) => {
          const segments = (operation.operationId ?? operation.id).split(/[./]/)
          if (operation.tags?.includes('Ops')) {
            return [
              segments
                .map((segment, index) =>
                  index === 0
                    ? segment
                    : segment.charAt(0).toUpperCase() + segment.slice(1)
                )
                .join(''),
            ]
          }
          return [segments.at(-1) ?? operation.id]
        },
      },
    },
  ],
})
