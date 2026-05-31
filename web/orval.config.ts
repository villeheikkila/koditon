import { defineConfig } from 'orval'

export default defineConfig({
  koditon: {
    input: {
      target: 'http://localhost:8080/openapi.yaml',
    },
    output: {
      mode: 'single',
      target: 'src/api/koditon.ts',
      client: 'react-query',
      override: {
        mutator: {
          path: 'src/lib/axios-instance.ts',
          name: 'customInstance',
        },
      },
    },
  },
})
