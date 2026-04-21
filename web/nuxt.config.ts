// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  modules: [
    '@nuxt/eslint',
    '@nuxt/ui',
    '@vueuse/nuxt'
  ],

  ssr: false,

  devtools: {
    enabled: true
  },

  css: ['~/assets/css/main.css'],

  runtimeConfig: {
    apiUrl: '',
    minioEndpoint: ''
  },

  compatibilityDate: '2024-07-11',

  nitro: {
    experimental: {
      websocket: true
    }
  },

  vite: {
    optimizeDeps: {
      include: [
        'date-fns',
        '@internationalized/date',
        '@unovis/vue',
        '@tanstack/vue-table',
        '@tanstack/table-core',
        '@tanstack/vue-virtual',
        '@tanstack/virtual-core'
      ]
    }
  },

  eslint: {
    config: {
      stylistic: {
        commaDangle: 'never',
        braceStyle: '1tbs'
      }
    }
  }
})
