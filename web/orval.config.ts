import { defineConfig } from 'orval'

export default defineConfig({
  koditon: {
    input: {
      target: '../koditon/Packages/KoditonClient/Sources/KoditonClient/openapi.yaml',
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
