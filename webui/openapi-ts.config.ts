import { defineConfig } from '@hey-api/openapi-ts'

export default defineConfig({
  client: '@hey-api/client-fetch',
  input: 'lib/api/schema/swagger.json',
  output: {
    path: 'lib/api/generated',
    format: 'prettier',
    lint: 'eslint',
  },
  types: {
    enums: 'typescript',
  },
  services: {
    asClass: true,
    name: '{{name}}', // This removes the 'Service' suffix
    methodNameBuilder: (operation) => {
      let methodName = operation.name
      if (
        operation.service &&
        methodName.toLowerCase().startsWith(operation.service.toLowerCase())
      ) {
        methodName = methodName.slice(operation.service.length)
      }
      // Ensure the first letter is lowercase
      return methodName.charAt(0).toLowerCase() + methodName.slice(1)
    },
  },
  schemas: {
    export: false,
  },
})
